package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"strings"
	"time"
)

type SQLStorage struct {
	db *sql.DB
}

func NewSQLStorage(db *sql.DB) *SQLStorage {
	return &SQLStorage{db: db}
}

func (s *SQLStorage) CreateURL(ctx context.Context, originalURL string) (int64, error) {
	if originalURL == "" {
		return 0, ErrEmptyURL
	}
	var id int64
	err := s.db.QueryRowContext(ctx, "INSERT INTO urls (original_url) VALUES ($1) RETURNING id", originalURL).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (s *SQLStorage) UpdateCode(ctx context.Context, id int64, shortCode string) error {
	res, err := s.db.ExecContext(ctx, "UPDATE urls SET short_code = $1 WHERE id = $2", shortCode, id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return ErrShortCodeAlreadyExists
		}
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return ErrURLNotFound
	}
	return nil
}

func (s *SQLStorage) GetCodeByURL(ctx context.Context, originalURL string) (string, error) {
	var code string
	err := s.db.QueryRowContext(ctx, "SELECT short_code FROM urls WHERE original_url = $1 AND short_code IS NOT NULL", originalURL).Scan(&code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		return "", err
	}
	return code, nil
}

func (s *SQLStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	var originalURL string
	err := s.db.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		return "", err
	}
	return originalURL, nil
}

func SetupPostgres() *sql.DB {
	connStr := "postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping postgres: %v", err)
	}
	db.SetMaxOpenConns(200)
	db.SetMaxIdleConns(50)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(2 * time.Minute)
	return db
}
