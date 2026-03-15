package migrations

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"

	"websitego/internal/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/golang-migrate/migrate/v4"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"gorm.io/gorm"
)

//go:embed sql/*.sql
var migrationFiles embed.FS

func Run(cfg *config.Config, db *gorm.DB) error {
	return Up(cfg, db)
}

func Up(cfg *config.Config, db *gorm.DB) error {
	runner, err := newMigrator(cfg, db)
	if err != nil {
		return err
	}
	defer func() {
		_ = runner.Close()
	}()

	if err := runner.m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run up migrations: %w", err)
	}

	return nil
}

func Down(cfg *config.Config, db *gorm.DB) error {
	runner, err := newMigrator(cfg, db)
	if err != nil {
		return err
	}
	defer func() {
		_ = runner.Close()
	}()

	if err := runner.m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("failed to run down migrations: %w", err)
	}

	return nil
}

func Force(cfg *config.Config, db *gorm.DB, version int) error {
	runner, err := newMigrator(cfg, db)
	if err != nil {
		return err
	}
	defer func() {
		_ = runner.Close()
	}()

	if err := runner.m.Force(version); err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	return nil
}

type runner struct {
	m     *migrate.Migrate
	sqlDB *sql.DB
}

func (r *runner) Close() error {
	var closeErr error
	if r.m != nil {
		if sourceErr, databaseErr := r.m.Close(); sourceErr != nil || databaseErr != nil {
			closeErr = fmt.Errorf("failed to close migrator, sourceErr=%v databaseErr=%v", sourceErr, databaseErr)
		}
	}
	if r.sqlDB != nil {
		if err := r.sqlDB.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
	}
	return closeErr
}

func newMigrator(cfg *config.Config, db *gorm.DB) (*runner, error) {
	_ = db

	sqlDB, err := sql.Open("mysql", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("failed to open mysql connection for migration: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to ping mysql for migration: %w", err)
	}

	driver, err := migratemysql.WithInstance(sqlDB, &migratemysql.Config{})
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to init mysql migration driver: %w", err)
	}

	sub, err := fs.Sub(migrationFiles, "sql")
	if err != nil {
		return nil, fmt.Errorf("failed to load migration files: %w", err)
	}

	sourceDriver, err := iofs.New(sub, ".")
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to init migration source: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", sourceDriver, cfg.DBName, driver)
	if err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("failed to init migrator: %w", err)
	}
	return &runner{
		m:     m,
		sqlDB: sqlDB,
	}, nil
}
