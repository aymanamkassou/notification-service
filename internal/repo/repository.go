package repo

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository wraps the generated Queries with connection pooling and transaction support
type Repository struct {
	pool *pgxpool.Pool
	*Queries
}

// NewRepository creates a new repository with a connection pool
func NewRepository(connString string) (*Repository, error) {
	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	// Configure connection pool
	poolConfig.MaxConns = 25
	poolConfig.MinConns = 5

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test the connection
	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &Repository{
		pool:    pool,
		Queries: New(pool),
	}, nil
}

// Close closes the connection pool
func (r *Repository) Close() {
	if r.pool != nil {
		r.pool.Close()
	}
}

// WithTx executes a function within a database transaction
func (r *Repository) WithTx(ctx context.Context, fn func(*Queries) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	q := New(tx)

	if err := fn(q); err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx error: %v, rollback error: %w", err, rbErr)
		}
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Pool returns the underlying connection pool for advanced usage
func (r *Repository) Pool() *pgxpool.Pool {
	return r.pool
}

// Health checks if the database connection is healthy
func (r *Repository) Health(ctx context.Context) error {
	return r.pool.Ping(ctx)
}

// Stats returns connection pool statistics
func (r *Repository) Stats() *pgxpool.Stat {
	return r.pool.Stat()
}
