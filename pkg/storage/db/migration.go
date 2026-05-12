package db

import (
	"database/sql"
	"embed"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

//go:embed migrations/*.sql
var migrations embed.FS

func RunDatabaseMigration(db *sql.DB, dbName string) error {
	// Creates a new migration instance
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		klog.Errorf("failed to setup db migration driver: %v", err)
		return err
	}
	d, err := iofs.New(migrations, "migrations")
	if err != nil {
		return err
	}

	m, err := migrate.NewWithInstance(
		"iofs",
		d,
		dbName,
		driver,
	)
	if err != nil {
		return err
	}

	// Run migration
	if err = m.Up(); err != nil {
		if err != migrate.ErrNoChange {
			klog.Errorf("failed to run db migration: %v", err)
			return err
		}
		klog.Info("db migrations are up to date")
	}

	return nil
}
