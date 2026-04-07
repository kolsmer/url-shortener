package service

import (
	"context"
	"errors"
	"net/url"
	"path"
	"slices"
	"strings"
	"url-shortener/internal/storage"
)

var ErrEmptyURL = errors.New("URL cannot be empty")
var ErrInvalidURL = errors.New("invalid URL")

const maxURLLength = 2048

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
	if len(originalURL) > maxURLLength {
		return "", ErrInvalidURL
	}

	u, err := url.Parse(originalURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return "", ErrInvalidURL
	}
	u.Host = strings.ToLower(u.Host)
	if u.Path != "" && u.Path != "/" {
		u.Path = path.Clean(u.Path)
	}

	normalizedURL := u.String()

	id, err := service.storage.CreateURL(ctx, normalizedURL)
	if err != nil {
		return "", err
	}
	shortCode := encodeBase62(id)
	err = service.storage.UpdateCode(ctx, id, shortCode)
	if err != nil {
		return "", err
	}
	return shortCode, nil
}

func (service *URLService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return service.storage.GetURL(ctx, shortCode)
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func encodeBase62(id int64) string {
	var remainder int64
	var result []byte
	for id > 0 {
		remainder = id % 62
		result = append(result, charset[remainder])
		id /= 62
	}
	slices.Reverse(result)
	return string(result)
}
