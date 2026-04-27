package service

import (
	"context"
	"errors"
	"math"
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
	// Check if URL already exists
	code, err := service.storage.GetCodeByURL(ctx, normalizedURL)
	if err == nil {
		return code, nil
	}
	if !errors.Is(err, storage.ErrURLNotFound) {
		return "", err
	}

	// Not found, create new	id, err := service.storage.CreateURL(ctx, normalizedURL)
	id, err := service.storage.CreateURL(ctx, normalizedURL)
	if err != nil {
		return "", err
	}
	shortCode := encodeBase36(id)
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

const charset = "abcdefghijklmnopqrstuvwxyz0123456789"

const minCodeLength = 6

var offset int64

func init() {
	offset = int64(math.Pow(36, float64(minCodeLength-1)))
}

func encodeBase36(id int64) string {
	actualID := id + offset
	var remainder int64
	var result []byte
	for actualID > 0 {
		remainder = actualID % 36
		result = append(result, charset[remainder])
		actualID /= 36
	}
	slices.Reverse(result)
	return string(result)
}
