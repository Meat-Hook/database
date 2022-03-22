package migrations

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"

	"github.com/jmoiron/sqlx"

	"github.com/sipki-corp/database"
)

// Command of migration
type Command uint8

// Enum.
const (
	_    Command = iota
	Up           // up
	Down         // down
)

// Run execute every 'delimUp' instructions for every migration.
func Run(ctx context.Context, driver string, connector database.Connector, cmd Command, migrations Migrations) error {
	dsn, err := connector.DSN()
	if err != nil {
		return fmt.Errorf("connector.DSN: %w", err)
	}

	conn, err := sql.Open(driver, dsn)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}

	err = conn.PingContext(ctx)
	for err != nil {
		nextErr := conn.PingContext(ctx)
		if errors.Is(nextErr, context.DeadlineExceeded) || errors.Is(nextErr, context.Canceled) {
			return fmt.Errorf("db.PingContext: %w", err)
		}
		err = nextErr
	}

	db := sqlx.NewDb(conn, driver)

	sort.Sort(migrations)
	switch cmd {
	case Up:
		err = upAll(ctx, db, migrations)
	case Down:
		err = rollback(ctx, db, migrations)
	default:
		err = fmt.Errorf("unknown command: %d", cmd)
	}
	if err != nil {
		return err
	}

	return db.Close()
}

func rollback(ctx context.Context, db *sqlx.DB, migrations Migrations) error {
	version, err := currentVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("currentVersion: %w", err)
	}

	sort.Sort(sort.Reverse(migrations))

	for _, migration := range migrations {
		if version < migration.Version {
			continue
		}

		err := down(ctx, db, migration)
		if err != nil {
			return fmt.Errorf("upOneVersion: %w", err)
		}
	}

	return nil
}

func upAll(ctx context.Context, db *sqlx.DB, migrations Migrations) error {
	version, err := currentVersion(ctx, db)
	if err != nil {
		return fmt.Errorf("currentVersion: %w", err)
	}

	for _, migration := range migrations {
		if version >= migration.Version {
			continue
		}

		err := upOneVersion(ctx, db, migration)
		if err != nil {
			return fmt.Errorf("upOneVersion: %w", err)
		}
	}

	return nil
}

func upOneVersion(ctx context.Context, db *sqlx.DB, migration Migration) (err error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTxx: %w", err)
	}
	defer func() {
		if err != nil {
			errRollback := tx.Rollback()
			if errRollback != nil {
				err = fmt.Errorf("%w: %s", err, errRollback)
			}
		}
	}()

	_, err = tx.ExecContext(ctx, migration.Up)
	if err != nil {
		return fmt.Errorf("tx.ExecContext %w", err)
	}

	_, err = tx.ExecContext(ctx, "insert into migration (version) values ($1)", migration.Version)
	if err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}

	return tx.Commit()
}

func down(ctx context.Context, db *sqlx.DB, migration Migration) (err error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("db.BeginTxx: %w", err)
	}
	defer func() {
		if err != nil {
			errRollback := tx.Rollback()
			if errRollback != nil {
				err = fmt.Errorf("%w: %s", err, errRollback)
			}
		}
	}()

	_, err = tx.ExecContext(ctx, migration.Down)
	if err != nil {
		return fmt.Errorf("tx.ExecContext %w", err)
	}

	_, err = tx.ExecContext(ctx, "delete from migration where version = $1", migration.Version)
	if err != nil {
		return fmt.Errorf("tx.ExecContext: %w", err)
	}

	return tx.Commit()
}

func currentVersion(ctx context.Context, db *sqlx.DB) (uint, error) {
	const initTable = `create table if not exists migration
(
    version integer         		not null,
    time    timestamp default now() not null,
    unique (version),
    primary key (version)
);`

	_, err := db.ExecContext(ctx, initTable)
	if err != nil {
		return 0, fmt.Errorf("db.ExecContext: %w", err)
	}

	const query = `SELECT version FROM migration ORDER BY version DESC LIMIT 1`
	version := uint(0)
	err = db.QueryRowContext(ctx, query).Scan(&version)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("db.QueryRowContext: %w", err)
	}

	return version, nil
}
