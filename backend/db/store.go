package db

import (
	"context"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{db: db}
}

// withTx runs fn in one transaction and commits only when fn succeeds, so
// writes that span several rows land together or not at all.
func (s *Store) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := fn(tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

const (
	// maxConns is a few more connections than the library's default on a
	// small server, so a slow request cannot hold up every other one.
	maxConns = 10
	// statementTimeout stops a runaway query before it ties up a connection
	// for long. The largest legitimate writes, imports, take well under it.
	statementTimeout = 15 * time.Second
)

func NewDBPool(ctx context.Context, connURL string) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(connURL)
	if err != nil {
		return nil, err
	}
	config.MaxConns = maxConns
	config.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(statementTimeout.Milliseconds(), 10)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}

	return pool, nil
}
