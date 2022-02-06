package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/Meat-Hook/repo"
	"github.com/Meat-Hook/repo/internal"
)

// Default values for config.
const (
	DefaultSetConnMaxLifetime    = time.Second * 10
	DefaultSetConnMaxIdleTime    = time.Second * 10
	DefaultSetMaxOpenConnections = 50
	DefaultSetMaxIdleConnections = 50
)

// Config for set additional properties.
type Config struct {
	ReturnErrs            []error
	Metrics               repo.MetricCollector
	SetConnMaxLifetime    time.Duration
	SetConnMaxIdleTime    time.Duration
	SetMaxOpenConnections int
	SetMaxIdleConnections int
}

func (c Config) setDefault() Config {
	if c.Metrics == nil {
		c.Metrics = repo.NoMetric{}
	}
	if c.SetConnMaxLifetime == 0 {
		c.SetConnMaxLifetime = DefaultSetConnMaxLifetime
	}
	if c.SetConnMaxIdleTime == 0 {
		c.SetConnMaxIdleTime = DefaultSetConnMaxIdleTime
	}
	if c.SetMaxOpenConnections == 0 {
		c.SetMaxOpenConnections = DefaultSetMaxOpenConnections
	}
	if c.SetMaxIdleConnections == 0 {
		c.SetMaxIdleConnections = DefaultSetMaxIdleConnections
	}
	return c
}

// Connector for making connection.
type Connector interface {
	// DSN returns connection string.
	DSN() (string, error)
}

// DB is a wrapper for sql database.
type DB struct {
	conn       *sqlx.DB
	returnErrs []error
	metrics    repo.MetricCollector
}

// New build and returns new DB.
func New(ctx context.Context, driver string, cfg Config, connector Connector) (*DB, error) {
	cfg = cfg.setDefault()

	dsn, err := connector.DSN()
	if err != nil {
		return nil, fmt.Errorf("connector.DSN: %w", err)
	}

	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}

	err = conn.PingContext(ctx)
	for err != nil {
		nextErr := conn.PingContext(ctx)
		if errors.Is(nextErr, context.DeadlineExceeded) || errors.Is(nextErr, context.Canceled) {
			return nil, fmt.Errorf("db.PingContext: %w", err)
		}
		err = nextErr
	}

	db := &DB{
		conn:       sqlx.NewDb(conn, driver),
		returnErrs: cfg.ReturnErrs,
		metrics:    cfg.Metrics,
	}

	db.conn.SetConnMaxLifetime(cfg.SetConnMaxLifetime)
	db.conn.SetConnMaxIdleTime(cfg.SetConnMaxIdleTime)
	db.conn.SetMaxOpenConns(cfg.SetMaxOpenConnections)
	db.conn.SetMaxIdleConns(cfg.SetMaxIdleConnections)

	return db, nil
}

// Turn sqlx errors like `missing destination …` into panics
// https://github.com/jmoiron/sqlx/issues/529. As we can't distinguish
// between sqlx and other errors except driver ones, let's hope filtering
// driver errors is enough and there are no other non-driver regular errors.
func (db *DB) strict(err error) error {
	switch {
	case err == nil:
	case errors.As(err, new(*pq.Error)):
	case errors.Is(err, sql.ErrNoRows):
	case errors.Is(err, context.Canceled):
	case errors.Is(err, context.DeadlineExceeded):
	default:
		for i := range db.returnErrs {
			if errors.Is(err, db.returnErrs[i]) {
				return err
			}
		}
		panic(err)
	}
	return err
}

// Close implements io.Closer.
func (db *DB) Close() error {
	return db.conn.Close()
}

// NoTx provides DAL method wrapper with:
// - converting sqlx errors which are actually bugs into panics,
// - general metrics for DAL methods,
// - wrapping errors with DAL method name.
func (db *DB) NoTx(f func(*sqlx.DB) error) (err error) {
	methodName := internal.CallerMethodName(1)
	return db.strict(db.metrics.Collecting(methodName, func() error {
		err := f(db.conn)
		if err != nil {
			err = fmt.Errorf("%s: %w", methodName, err)
		}
		return err
	})())
}

// Tx provides DAL method wrapper with:
// - converting sqlx errors which are actually bugs into panics,
// - general metrics for DAL methods,
// - wrapping errors with DAL method name,
// - transaction.
func (db *DB) Tx(ctx context.Context, opts *sql.TxOptions, f func(*sqlx.Tx) error) (err error) {
	methodName := internal.CallerMethodName(1)
	return db.strict(db.metrics.Collecting(methodName, func() error {
		tx, err := db.conn.BeginTxx(ctx, opts)
		if err == nil { //nolint:nestif // No idea how to simplify.
			defer func() {
				if err := recover(); err != nil {
					if errRollback := tx.Rollback(); errRollback != nil {
						err = fmt.Errorf("%v: %s", err, errRollback)
					}
					panic(err)
				}
			}()
			err = f(tx)
			if err == nil {
				err = tx.Commit()
			} else if errRollback := tx.Rollback(); errRollback != nil {
				err = fmt.Errorf("%v: %s", err, errRollback)
			}
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", methodName, err)
		}
		return err
	})())
}
