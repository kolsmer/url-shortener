package storage

import (
	"context"
	"errors"
	"sync"
)

var ErrURLNotFound = errors.New("URL not found")
var ErrShortCodeAlreadyExists = errors.New("short code already exists")

type Storage interface {
	SaveURL(ctx context.Context, originalURL, shortCode string) error
	GetURL(ctx context.Context, shortCode string) (string, error)
}
type InMemoryStorage struct {
	urls  map[string]string
	mutex sync.RWMutex
}

func NewInMemoryStorage() *InMemoryStorage {
	urls := make(map[string]string)
	return &InMemoryStorage{urls: urls}
}

func (s *InMemoryStorage) SaveURL(ctx context.Context, originalURL, shortCode string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, exists := s.urls[shortCode]; exists {
		return ErrShortCodeAlreadyExists
	}
	s.urls[shortCode] = originalURL
	return nil
}

func (s *InMemoryStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	if url, ok := s.urls[shortCode]; ok {
		return url, nil
	}
	return "", ErrURLNotFound
}
