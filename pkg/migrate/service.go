package migrate

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	sqlmigrate "github.com/rubenv/sql-migrate"
	"go.uber.org/zap"

	"stg/db"
	"stg/pkg/connections/postgres"
)

type Migrations struct {
	logger *zap.Logger
	config Config

	pgPool *postgres.Pool
}

func New(logger *zap.Logger, config Config, pgPool *postgres.Pool) *Migrations {
	return &Migrations{
		logger: logger,
		config: config,
		pgPool: pgPool,
	}
}

func (m *Migrations) Start(_ context.Context) error {
	m.logger.Info("Starting migrations")

	pool, err := m.pgPool.GetPool(m.config.PoolName)
	if err != nil {
		return fmt.Errorf("get pool: %w", err)
	}

	rp := pool.RawPool()
	sqlDB := stdlib.OpenDBFromPool(rp)
	defer func() {
		err = sqlDB.Close()
		if err != nil {
			m.logger.Warn("Failed to close database connection", zap.Error(err))
		}
	}()

	if err = m.runMigrations(sqlDB); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	m.logger.Info("Migrations completed")

	return nil
}

func (m *Migrations) runMigrations(dbConnection *sql.DB) error {
	migrationsList := &sqlmigrate.EmbedFileSystemMigrationSource{
		FileSystem: db.Migrations,
		Root:       "migrations",
	}

	n, err := sqlmigrate.Exec(dbConnection, "postgres", migrationsList, sqlmigrate.Up)
	if err != nil {
		return fmt.Errorf("pg migrate error: %w", err)
	}

	m.logger.Info("Migrations completed", zap.Int("n", n))

	return nil
}
