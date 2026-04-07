package storage

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
)

func setupPostgresTestDB(t *testing.T) *sql.DB {
	connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		t.Fatalf("failed to connect to postgres: %v", err)
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(0)
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
					id BIGSerial PRIMARY KEY,
					original_url TEXT NOT NULL,
					short_code TEXT UNIQUE
                )
		`)
	if err != nil {
		t.Fatalf("failed to create table: %v", err)
	}
	return db
}

func clearTable(t *testing.T, db *sql.DB) {
	_, err := db.Exec(`TRUNCATE urls RESTART IDENTITY `)
	if err != nil {
		t.Fatalf("failed to clear table: %v", err)
	}
}

func TestSQLStorage(t *testing.T) {
	runStorageTests(t, func() Storage {
		db := setupPostgresTestDB(t)
		clearTable(t, db)
		return NewSQLStorage(db)
	})
}
