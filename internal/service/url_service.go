package service

import (
	"context"
	"errors"
	"math/rand/v2"
	"url-shortener/internal/storage"
)

var ErrEmptyURL = errors.New("URL cannot be empty")

type URLService struct {
	storage storage.Storage
}

func NewURLService(storage storage.Storage) *URLService {
	return &URLService{storage: storage}
}

func (service *URLService) ShortenURL(ctx context.Context, originalURL string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if originalURL == "" {
		return "", ErrEmptyURL
	}

	for {
		code := generateShortCode()
		err := service.storage.SaveURL(ctx, originalURL, code)
		if errors.Is(err, storage.ErrShortCodeAlreadyExists) {
			continue
		} else if err != nil {
			return "", err
		}
		return code, nil
	}

}

func (service *URLService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return service.storage.GetURL(ctx, shortCode)
}

func generateShortCode() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	const length = 8
	code := make([]byte, length)
	for i := range code {
		code[i] = charset[rand.IntN(len(charset))]
	}
	return string(code)
}
