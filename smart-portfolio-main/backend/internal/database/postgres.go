package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ZRishu/smart-portfolio/internal/config"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rs/zerolog/log"
)

// Postgres wraps a pgxpool.Pool and provides helpers for the application.
type Postgres struct {
	Pool *pgxpool.Pool
}

// New creates a new connection pool to PostgreSQL using the provided config.
// It validates the connection with a ping before returning.
func New(ctx context.Context, cfg config.DatabaseConfig) (*Postgres, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("database: failed to parse connection URL: %w", err)
	}

	poolCfg.MaxConns = int32(cfg.MaxOpenConns)
	poolCfg.MinConns = int32(cfg.MaxIdleConns)
	poolCfg.MaxConnLifetime = cfg.ConnMaxLifetime
	poolCfg.MaxConnIdleTime = 5 * time.Minute
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: failed to create connection pool: %w", err)
	}

	// Validate the connection is alive.
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping failed: %w", err)
	}

	log.Info().
		Int("max_open_conns", cfg.MaxOpenConns).
		Int("max_idle_conns", cfg.MaxIdleConns).
		Dur("conn_max_lifetime", cfg.ConnMaxLifetime).
		Msg("database: connected to PostgreSQL")

	return &Postgres{Pool: pool}, nil
}

// Close gracefully shuts down the connection pool.
func (pg *Postgres) Close() {
	if pg.Pool != nil {
		pg.Pool.Close()
		log.Info().Msg("database: connection pool closed")
	}
}

// RunMigrations executes all .sql files found in the given migrations directory.
// Files are sorted lexicographically so naming them 001_, 002_, etc. controls order.
// Each file is executed in a single transaction. Migrations are idempotent by design
// (using IF NOT EXISTS / IF EXISTS in SQL).
func (pg *Postgres) RunMigrations(ctx context.Context, migrationsDir string) error {
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("database: failed to read migrations directory %q: %w", migrationsDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		ext := filepath.Ext(entry.Name())
		if ext != ".sql" {
			continue
		}

		filePath := filepath.Join(migrationsDir, entry.Name())
		sqlBytes, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("database: failed to read migration file %q: %w", filePath, err)
		}

		sql := string(sqlBytes)
		if sql == "" {
			continue
		}

		tx, err := pg.Pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("database: failed to begin transaction for %q: %w", entry.Name(), err)
		}

		if _, err := tx.Exec(ctx, sql); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("database: migration %q failed: %w", entry.Name(), err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("database: failed to commit migration %q: %w", entry.Name(), err)
		}

		log.Info().Str("file", entry.Name()).Msg("database: migration applied")
	}

	log.Info().Msg("database: all migrations applied successfully")
	return nil
}

// HealthCheck pings the database and returns an error if unreachable.
func (pg *Postgres) HealthCheck(ctx context.Context) error {
	checkCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return pg.Pool.Ping(checkCtx)
}
