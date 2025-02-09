package db

import (
	"database/sql"
	"os"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"k8s.io/klog/v2"
)

func RunDatabaseMigration(db *sql.DB, dbName string) error {
	// Creates a new migration instance
	migrationsPath := "file:///" + os.Getenv("KO_DATA_PATH") + "/migrations"
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		klog.Errorf("failed to setup db migration driver: %v", err)
		return err
	}
	m, err := migrate.NewWithDatabaseInstance(migrationsPath, dbName, driver)
	if err != nil {
		klog.Errorf("failed to create migration connection to db: %v", err)
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
