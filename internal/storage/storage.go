package storage

import (
	"context"
	"errors"
	"url-shortener/internal/models"
)

var ErrURLNotFound = errors.New("URL not found")
var ErrShortCodeAlreadyExists = errors.New("short code already exists")
var ErrEmptyURL = errors.New("URL cannot be empty")



type Storage interface {
	UpdateCode(ctx context.Context, id int64, shortCode string) error
	GetURL(ctx context.Context, shortCode string) (string, error)
	CreateURL(ctx context.Context, originalURL string, userID int64) (int64, error)
	GetCodeByURL(ctx context.Context, originalURL string) (string, error)
	GetURLsByUserID(ctx context.Context, userID int64) ([]models.URL, error)
}