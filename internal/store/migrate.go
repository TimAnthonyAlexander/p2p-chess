package store

import (
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"

	"log"
)

func RunMigrations(dsn string) error {
	m, err := migrate.New("file://migrations", dsn)
	if err != nil {
		return err
	}
	log.Println("Starting migrations...")
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}
	log.Println("Migrations completed (or no changes needed)")
	return nil
}
