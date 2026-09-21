package storage

import (
	"context"
	"database/sql"
	"errors"
	"log"
	"os"
	"strings"
	"time"
	"url-shortener/internal/models"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("url-shortener/storage")

type SQLStorage struct {
	db *sql.DB
}

func NewSQLStorage(db *sql.DB) *SQLStorage {
	return &SQLStorage{db: db}
}

func (s *SQLStorage) CreateURL(ctx context.Context, originalURL string, userID int64) (int64, error) {
	ctx, span := tracer.Start(ctx, "CreateURL")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	if originalURL == "" {
		span.RecordError(ErrEmptyURL)
		span.SetStatus(codes.Error, ErrEmptyURL.Error())
		return 0, ErrEmptyURL
	}
	var id int64
	err := s.db.QueryRowContext(ctx, "INSERT INTO urls (original_url, user_id) VALUES ($1, $2) RETURNING id", originalURL, userID).Scan(&id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return 0, err
	}
	return id, nil
}

func (s *SQLStorage) UpdateCode(ctx context.Context, id int64, shortCode string) error {
	ctx, span := tracer.Start(ctx, "UpdateCode")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	res, err := s.db.ExecContext(ctx, "UPDATE urls SET short_code = $1 WHERE id = $2", shortCode, id)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			span.RecordError(ErrShortCodeAlreadyExists)
			span.SetStatus(codes.Error, ErrShortCodeAlreadyExists.Error())
			return ErrShortCodeAlreadyExists
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return err
	}
	if rows == 0 {
		span.RecordError(ErrURLNotFound)
		span.SetStatus(codes.Error, ErrURLNotFound.Error())
		return ErrURLNotFound
	}
	return nil
}

func (s *SQLStorage) GetCodeByURL(ctx context.Context, originalURL string) (string, error) {
	ctx, span := tracer.Start(ctx, "GetCodeByURL")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	var code string
	err := s.db.QueryRowContext(ctx, "SELECT short_code FROM urls WHERE original_url = $1 AND short_code IS NOT NULL AND short_code <> ''", originalURL).Scan(&code)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrURLNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return code, nil
}

func (s *SQLStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	ctx, span := tracer.Start(ctx, "GetURL")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	var originalURL string
	err := s.db.QueryRowContext(ctx, "SELECT original_url FROM urls WHERE short_code = $1", shortCode).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			span.RecordError(ErrURLNotFound)
			span.SetStatus(codes.Error, ErrURLNotFound.Error())
			return "", ErrURLNotFound
		}
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return originalURL, nil
}

func (s *SQLStorage) GetURLsByUserID(ctx context.Context, userID int64) ([]models.URL, error) {
	ctx, span := tracer.Start(ctx, "GetURLsByUserID")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	rows, err := s.db.QueryContext(ctx, "SELECT original_url, short_code, created_at, user_id FROM urls WHERE user_id = $1", userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}
	defer rows.Close()

	var urls []models.URL
	for rows.Next() {
		var u models.URL
		if err := rows.Scan(&u.OriginalURL, &u.ShortCode, &u.CreatedAt, &u.UserID); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}
		urls = append(urls, u)
	}
	return urls, rows.Err()
}

func SetupPostgres() *sql.DB {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		connStr = "postgres://postgres:postgres@localhost:5432/url_shortener?sslmode=disable"
	}
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
