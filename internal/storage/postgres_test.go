package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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



func runStorageTests(t *testing.T, factory func() Storage) {
	t.Run("CreateUpdateGet", func(t *testing.T) {
		s := factory()
		ctx := context.Background()

		id, err := s.CreateURL(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("createURL failed: %v", err)
		}

		err = s.UpdateCode(ctx, id, "abc123")
		if err != nil {
			t.Fatalf("updateCode failed: %v", err)
		}

		url, err := s.GetURL(ctx, "abc123")
		if err != nil {
			t.Fatalf("getURL failed: %v", err)
		}

		if url != "https://example.com" {
			t.Fatalf("expected https://example.com got %s", url)
		}

	})

	t.Run("DuplicateCode", func(t *testing.T) {
		s := factory()
		ctx := context.Background()
		id1, _ := s.CreateURL(ctx, "https://example1.com")
		id2, _ := s.CreateURL(ctx, "https://example2.com")

		_ = s.UpdateCode(ctx, id1, "abc123")
		err := s.UpdateCode(ctx, id2, "abc123")
		if err == nil {
			t.Fatalf("expected error for duplicate code, got nil")
		}
	})

	t.Run("NotFound", func(t *testing.T) {
		s := factory()
		ctx := context.Background()
		_, err := s.GetURL(ctx, "nonexistent")
		if err == nil {
			t.Fatalf("expected error for non-existent code, got nil")
		}
	})

	t.Run("ConcurrentCreateUpdate", func(t *testing.T) {
		s := factory()
		ctx := context.Background()
		n := 200
		errChan := make(chan error, n)
		for i := 0; i < n; i++ {
			go func(i int) {
				url := fmt.Sprintf("https://example.com/%d", i)
				id, err := s.CreateURL(ctx, url)
				if err != nil {
					errChan <- err
					return
				}
				code := fmt.Sprintf("abc%d", i)
				err = s.UpdateCode(ctx, id, code)
				if err != nil {
					errChan <- err
					return
				}

				got, err := s.GetURL(ctx, code)
				if err != nil {
					errChan <- err
					return
				}

				if got != url {
					errChan <- fmt.Errorf("expected %s got %s", url, got)
				}
				errChan <- nil
			}(i)
		}

		for i := 0; i < n; i++ {
			err := <-errChan
			if err != nil {
				t.Fatalf("concurrent storage error: %v", err)
			}
		}
	})
	t.Run("GetURLIdempotent", func(t *testing.T) {
		s := factory()
		ctx := context.Background()
		id, err := s.CreateURL(ctx, "https://example.com")
		if err != nil {
			t.Fatalf("createURL failed: %v", err)
		}
		err = s.UpdateCode(ctx, id, "abc123")
		if err != nil {
			t.Fatalf("updateCode failed: %v", err)
		}
		for i := 0; i <= 100; i++ {
			url, err := s.GetURL(ctx, "abc123")
			if err != nil {
				t.Fatalf("getURL failed: %v", err)
			}
			if url != "https://example.com" {
				t.Fatalf("expected https://example.com got %s", url)
			}
		}
	})
	t.Run("CodeUniqueAcrossURLs", func(t *testing.T) {
		s := factory()
		ctx := context.Background()

		id1, _ := s.CreateURL(ctx, "https://example1.com")
		id2, _ := s.CreateURL(ctx, "https://example2.com")

		err := s.UpdateCode(ctx, id1, "abc123")
		if err != nil {
			t.Fatalf("updateCode failed: %v", err)
		}
		err = s.UpdateCode(ctx, id2, "abc123")
		if err == nil {
			t.Fatalf("updateCode should have failed due to duplicate code")
		}
		if !errors.Is(err, ErrShortCodeAlreadyExists) {
			t.Fatalf("expected ErrShortCodeAlreadyExists got %v", err)
		}
	})

	t.Run("IDMonotonic", func(t *testing.T) {
		s := factory()
		ctx := context.Background()
		prevID := int64(0)
		for i := 0; i < 100; i++ {
			id, err := s.CreateURL(ctx, "https://example.com")
			if err != nil {
				t.Fatalf("createURL failed: %v", err)
			}
			if id <= prevID {
				t.Fatalf("id not increasing: prev=%d current=%d", prevID, id)
			}
			prevID = id
		}
	})
}
