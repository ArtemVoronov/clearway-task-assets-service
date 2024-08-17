package services

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/ArtemVoronov/clearway-task-assets-service/internal/app"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgreSQLService struct {
	ConnectionsPool *pgxpool.Pool
	queryTimeout    time.Duration
}

func CreatePostgreSQLService(connString string) (*PostgreSQLService, error) {
	queryTimeoutStr, ok := os.LookupEnv("DATABASE_QUERY_TIMEOUT")
	if !ok {
		queryTimeoutStr = app.DefaultDatabaseQueryTimeout
	}

	queryTimeout, err := time.ParseDuration(queryTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), queryTimeout)
	defer cancel()

	dbpool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	err = dbpool.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to ping database %v: %w", connString, err)
	}

	return &PostgreSQLService{
		ConnectionsPool: dbpool,
		queryTimeout:    queryTimeout,
	}, nil
}

func (s *PostgreSQLService) Shutdown() error {
	if s == nil || s.ConnectionsPool == nil {
		return nil
	}

	s.ConnectionsPool.Close()

	return nil
}

type SqlQueryFunc func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) (any, error)

type SqlQueryFuncVoid func(tx pgx.Tx, ctx context.Context, cancel context.CancelFunc) error

func (s *PostgreSQLService) Tx(f SqlQueryFunc, options pgx.TxOptions) func() (any, error) {
	return func() (any, error) {
		ctx, cancel := context.WithTimeout(context.Background(), s.queryTimeout)
		defer cancel()
		tx, err := s.ConnectionsPool.BeginTx(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("error at creating tx: %w", err)
		}
		defer tx.Rollback(ctx)

		result, err := f(tx, ctx, cancel)
		if err != nil {
			return result, err
		}
		err = tx.Commit(ctx)
		if err != nil {
			return nil, fmt.Errorf("error at commiting tx: %w", err)
		}
		return result, err
	}
}

func (s *PostgreSQLService) TxVoid(f SqlQueryFuncVoid, options pgx.TxOptions) func() error {
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), s.queryTimeout)
		defer cancel()
		tx, err := s.ConnectionsPool.BeginTx(ctx, options)
		if err != nil {
			return fmt.Errorf("error at creating tx: %w", err)
		}
		defer tx.Rollback(ctx)

		err = f(tx, ctx, cancel)
		if err != nil {
			return err
		}
		err = tx.Commit(ctx)
		if err != nil {
			return fmt.Errorf("error at commiting tx: %w", err)
		}
		return err
	}
}
