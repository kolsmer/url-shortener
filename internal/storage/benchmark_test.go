package storage

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
)

func setupPostgresDB() (*sql.DB, error) {
	connStr := "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS urls (
		id BIGSERIAL PRIMARY KEY,
		original_url TEXT NOT NULL,
		short_code TEXT UNIQUE
	)`)
	if err != nil {
		return nil, err
	}

	return db, nil
}
func BenchmarkCreateURL(b *testing.B) {
	db, err := setupPostgresDB()
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}

	s := NewSQLStorage(db)
	ctx := context.Background()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0

		for pb.Next() {
			url := fmt.Sprintf("https://example.com/%d", i)
			_, err := s.CreateURL(ctx, url)
			if err != nil {
				b.Fatalf("CreateURL failed: %v", err)
			}
			i++
		}
	})
}
func BenchmarkFullFlow(b *testing.B) {
	db, err := setupPostgresDB()
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}
	b.SetParallelism(2)
	s := NewSQLStorage(db)
	ctx := context.Background()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			url := fmt.Sprintf("https://example.com/%d", i)

			id, err := s.CreateURL(ctx, url)
			if err != nil {
				b.Fatalf("create failed: %v", err)
			}

			code := fmt.Sprintf("code-%d", id)

			err = s.UpdateCode(ctx, id, code)
			if err != nil {
				b.Fatalf("update failed: %v", err)
			}

			_, err = s.GetURL(ctx, code)
			if err != nil {
				b.Fatalf("get failed: %v", err)
			}

			i++
		}
	})
}

func BenchmarkReadHeavy(b *testing.B) {
	db, err := setupPostgresDB()
	if err != nil {
		b.Fatalf("setup failed: %v", err)
	}

	s := NewSQLStorage(db)
	ctx := context.Background()
	b.SetParallelism(2)
	for i := 0; i < 1000; i++ {
		id, _ := s.CreateURL(ctx, fmt.Sprintf("https://example.com/%d", i))
		_ = s.UpdateCode(ctx, id, fmt.Sprintf("code-%d", i))
	}

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			code := fmt.Sprintf("code-%d", i%1000)

			_, err := s.GetURL(ctx, code)
			if err != nil {
				b.Fatalf("get failed: %v", err)
			}
			i++
		}
	})
}
