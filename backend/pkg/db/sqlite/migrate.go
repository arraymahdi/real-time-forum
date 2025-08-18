package sqlite

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

// Connect initializes the database connection and applies migrations
func Connect() *sql.DB {
	var err error
	db, err = sql.Open("sqlite3", "forum.db")
	if err != nil {
		log.Fatal(err)
	}

	// Apply migrations
	ApplyMigrations()

	return db
}

// ApplyMigrations runs database migrations
func ApplyMigrations() {
	m, err := migrate.New(
		"file://pkg/db/migrations/sqlite",
		"sqlite3://forum.db",
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatal(err)
	}

	fmt.Println("Migrations applied successfully!")
}

func RollbackLastMigration() {
	m, err := migrate.New(
		"file://backend/pkg/db/migrations/sqlite",
		"sqlite3://forum.db",
	)
	if err != nil {
		log.Fatal("Failed to initialize migrations:", err)
	}

	if err := m.Steps(-1); err != nil && err != migrate.ErrNoChange {
		log.Fatal("Failed to rollback migration:", err)
	}

	fmt.Println("Rolled back last migration successfully!")
}
