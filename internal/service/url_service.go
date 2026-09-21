package service

import (
	"context"
	"errors"
	"math"
	"net/url"
	"path"
	"slices"
	"strings"
	"url-shortener/internal/models"
	"url-shortener/internal/storage"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
)

var tracer = otel.Tracer("url-shortener/service")

var ErrEmptyURL = errors.New("URL cannot be empty")
var ErrInvalidURL = errors.New("invalid URL")

const maxURLLength = 2048

type URLService struct {
	storage storage.Storage
}

func NewURLService(storage storage.Storage) *URLService {
	return &URLService{storage: storage}
}

func (service *URLService) ShortenURL(ctx context.Context, originalURL string, userID int64) (string, error) {
	ctx, span := tracer.Start(ctx, "ShortenURL")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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
		if err == nil {
			span.SetStatus(codes.Error, "invalid URL scheme")
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return "", ErrInvalidURL
	}
	u.Host = strings.ToLower(u.Host)
	if u.Path != "" && u.Path != "/" {
		u.Path = path.Clean(u.Path)
	}

	normalizedURL := u.String()
	code, err := service.storage.GetCodeByURL(ctx, normalizedURL)
	if err == nil && code != "" {
		return code, nil
	}
	if err == nil {
		err = storage.ErrURLNotFound
	}
	if !errors.Is(err, storage.ErrURLNotFound) {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}

	id, err := service.storage.CreateURL(ctx, normalizedURL, userID)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	shortCode := encodeBase36(id)
	err = service.storage.UpdateCode(ctx, id, shortCode)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return "", err
	}
	return shortCode, nil
}

func (service *URLService) GetOriginalURL(ctx context.Context, shortCode string) (string, error) {
	ctx, span := tracer.Start(ctx, "GetOriginalURL")
	defer span.End()
	if err := ctx.Err(); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
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

func (service *URLService) GetUserURLs(ctx context.Context, userID int64) ([]models.URL, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return service.storage.GetURLsByUserID(ctx, userID)
}
