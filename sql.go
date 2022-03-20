package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"

	"github.com/sipki-group/database/internal"
)

// Default values for config.
const (
	DefaultSetConnMaxLifetime    = time.Second * 60
	DefaultSetConnMaxIdleTime    = time.Second * 10
	DefaultSetMaxOpenConnections = 50
	DefaultSetMaxIdleConnections = 50
)

// SQLConfig for set additional properties.
type SQLConfig struct {
	ReturnErrs            []error
	Metrics               MetricCollector
	SetConnMaxLifetime    time.Duration
	SetConnMaxIdleTime    time.Duration
	SetMaxOpenConnections int
	SetMaxIdleConnections int
}

func (c SQLConfig) setDefault() SQLConfig {
	if c.Metrics == nil {
		c.Metrics = NoMetric{}
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

// SQL is a wrapper for sql database.
type SQL struct {
	conn       *sqlx.DB
	returnErrs []error
	metrics    MetricCollector
}

// NewSQL build and returns new SQL client.
func NewSQL(ctx context.Context, driver string, cfg SQLConfig, connector Connector) (*SQL, error) {
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

	db := &SQL{
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
func (db *SQL) strict(err error) error {
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
func (db *SQL) Close() error {
	return db.conn.Close()
}

// NoTx provides DAL method wrapper with:
// - converting sqlx errors which are actually bugs into panics,
// - general metrics for DAL methods,
// - wrapping errors with DAL method name.
func (db *SQL) NoTx(f func(*sqlx.DB) error) (err error) {
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
func (db *SQL) Tx(ctx context.Context, opts *sql.TxOptions, f func(*sqlx.Tx) error) (err error) {
	methodName := internal.CallerMethodName(1)
	return db.strict(db.metrics.Collecting(methodName, func() error {
		tx, err := db.conn.BeginTxx(ctx, opts)
		if err == nil { //nolint:nestif // No idea how to simplify.
			defer func() {
				if err := recover(); err != nil {
					if errRollback := tx.Rollback(); errRollback != nil {
						err = fmt.Errorf("%w: %s", err, errRollback)
					}
					panic(err)
				}
			}()
			err = f(tx)
			if err == nil {
				err = tx.Commit()
			} else if errRollback := tx.Rollback(); errRollback != nil {
				err = fmt.Errorf("%w: %s", err, errRollback)
			}
		}
		if err != nil {
			err = fmt.Errorf("%s: %w", methodName, err)
		}
		return err
	})())
}
