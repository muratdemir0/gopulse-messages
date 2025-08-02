package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/doug-martin/goqu/v9/dialect/postgres"
	"github.com/google/uuid"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"

	"github.com/doug-martin/goqu/v9"
)

const (
	MaxOpenConnections = 40
	MaxIdleConnections = 10
)

var (
	ErrNoRows     = sql.ErrNoRows
	ErrTxNotFound = errors.New("transaction not found in context")
)

type Client struct {
	db   *sqlx.DB
	Goqu *goqu.Database
}

type sqlResult struct {
	lastInsertId int64
	rowsAffected int64
}

type key int

const txKey key = 0

func (r sqlResult) LastInsertId() (int64, error) { return r.lastInsertId, nil }
func (r sqlResult) RowsAffected() (int64, error) { return r.rowsAffected, nil }

func NewDB(dsn string) *Client {
	dialectOptions := postgres.DialectOptions()
	dialectOptions.SupportsWithCTE = true
	goqu.RegisterDialect("default", dialectOptions)
	goqu.SetDefaultPrepared(true)

	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		panic(err)
	}

	db.SetMaxOpenConns(MaxOpenConnections)
	db.SetMaxIdleConns(MaxIdleConnections)
	db.SetConnMaxIdleTime(time.Minute * 5)

	err = db.Ping()
	if err != nil {
		fmt.Println("Error connecting to the database:", err)
		panic(err)
	}

	return &Client{db: db, Goqu: goqu.New("default", db)}
}

func (c *Client) Close() error {
	return c.db.Close()
}

func (c *Client) BeginTx(ctx context.Context) (context.Context, error) {
	tx, err := c.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	return context.WithValue(ctx, txKey, tx), nil
}

func (c *Client) CommitTx(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(*sqlx.Tx)
	if !ok {
		return ErrTxNotFound
	}
	return tx.Commit()
}

func (c *Client) RollbackTx(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(*sqlx.Tx)
	if !ok {
		return ErrTxNotFound
	}
	return tx.Rollback()
}

func (c *Client) getTx(ctx context.Context) *sqlx.Tx {
	if tx, ok := ctx.Value(txKey).(*sqlx.Tx); ok {
		return tx
	}
	return nil
}

func (c *Client) QueryRow(ctx context.Context, dest any, query *goqu.SelectDataset) error {
	q, args, err := query.ToSQL()
	if err != nil {
		return fmt.Errorf("unable to build query: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var row *sqlx.Row
	if tx := c.getTx(ctx); tx != nil {
		row = tx.QueryRowxContext(ctx, q, args...)
	} else {
		row = c.db.QueryRowxContext(ctx, q, args...)
	}

	outType := reflect.TypeOf(dest)
	if outType.Kind() == reflect.Ptr {
		outType = outType.Elem()
	}

	var scanErr error
	if outType.Kind() == reflect.Struct {
		scanErr = row.StructScan(dest)
	} else {
		scanErr = row.Scan(dest)
	}

	if errors.Is(scanErr, sql.ErrNoRows) {
		return ErrNoRows
	}

	if scanErr != nil {
		return fmt.Errorf("unable to scan row: %w", scanErr)
	}

	return nil
}

func (c *Client) Select(ctx context.Context, dest any, query *goqu.SelectDataset) error {
	q, args, err := query.ToSQL()
	if err != nil {
		return fmt.Errorf("unable to build query: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	if tx := c.getTx(ctx); tx != nil {
		err = tx.SelectContext(ctx, dest, q, args...)
	} else {
		err = c.db.SelectContext(ctx, dest, q, args...)
	}

	if err != nil {
		return fmt.Errorf("unable to execute select query: %w", err)
	}
	return nil
}

func (c *Client) Insert(ctx context.Context, query *goqu.InsertDataset) (sql.Result, error) {
	q, args, err := query.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("unable to build query: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var lastInsertId int64
	var row *sql.Row

	if tx := c.getTx(ctx); tx != nil {
		row = tx.QueryRowContext(ctx, q+" RETURNING id", args...)
	} else {
		row = c.db.QueryRowContext(ctx, q+" RETURNING id", args...)
	}

	if err := row.Scan(&lastInsertId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("insert succeeded but returned no id")
		}
		return nil, fmt.Errorf("failed to scan inserted id: %w", err)
	}

	return sqlResult{
		lastInsertId: lastInsertId,
		rowsAffected: 1,
	}, nil
}

func (c *Client) InsertWithReturnUUID(ctx context.Context, query *goqu.InsertDataset) (uuid.UUID, error) {
	q, args, err := query.ToSQL()
	if err != nil {
		return uuid.Nil, fmt.Errorf("unable to build query: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var lastInsertId uuid.UUID
	var row *sql.Row

	if tx := c.getTx(ctx); tx != nil {
		row = tx.QueryRowContext(ctx, q+" RETURNING id", args...)
	} else {
		row = c.db.QueryRowContext(ctx, q+" RETURNING id", args...)
	}

	if err := row.Scan(&lastInsertId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, errors.New("insert succeeded but returned no id")
		}
		return uuid.Nil, fmt.Errorf("failed to scan inserted id: %w", err)
	}

	return lastInsertId, nil
}

func (c *Client) Update(ctx context.Context, query *goqu.UpdateDataset) (sql.Result, error) {
	q, args, err := query.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("unable to build query: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var res sql.Result
	if tx := c.getTx(ctx); tx != nil {
		res, err = tx.ExecContext(ctx, q, args...)
	} else {
		res, err = c.db.ExecContext(ctx, q, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to execute update query: %w", err)
	}
	return res, nil
}

func (c *Client) Delete(ctx context.Context, query *goqu.DeleteDataset) (sql.Result, error) {
	q, args, err := query.ToSQL()
	if err != nil {
		return nil, fmt.Errorf("unable to build query: %w", err)
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	var res sql.Result
	if tx := c.getTx(ctx); tx != nil {
		res, err = tx.ExecContext(ctx, q, args...)
	} else {
		res, err = c.db.ExecContext(ctx, q, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("unable to execute delete query: %w", err)
	}
	return res, nil
}
