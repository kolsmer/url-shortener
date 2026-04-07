package storage

import (
	"context"
	"errors"
	"sync"
)

var ErrURLNotFound = errors.New("URL not found")
var ErrShortCodeAlreadyExists = errors.New("short code already exists")
var ErrEmptyURL = errors.New("URL cannot be empty")

type Storage interface {
	UpdateCode(ctx context.Context, id int64, shortCode string) error
	GetURL(ctx context.Context, shortCode string) (string, error)
	CreateURL(ctx context.Context, originalURL string) (int64, error)
}
type InMemoryStorage struct {
	urlsByID map[int64]string
	idByCode map[string]int64
	nextID   int64
	mutex    sync.RWMutex
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{urlsByID: make(map[int64]string), idByCode: make(map[string]int64)}
}

func (s *InMemoryStorage) CreateURL(ctx context.Context, originalURL string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if originalURL == "" {
		return 0, ErrEmptyURL
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.nextID++
	id := s.nextID
	s.urlsByID[id] = originalURL
	return id, nil
}

func (s *InMemoryStorage) UpdateCode(ctx context.Context, id int64, shortCode string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if _, exists := s.urlsByID[id]; !exists {
		return ErrURLNotFound
	}
	if _, exists := s.idByCode[shortCode]; exists {
		return ErrShortCodeAlreadyExists
	}
	s.idByCode[shortCode] = id
	return nil
}

func (s *InMemoryStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	id, exists := s.idByCode[shortCode]
	if !exists {
		return "", ErrURLNotFound
	}
	url, exists := s.urlsByID[id]
	if !exists {
		return "", ErrURLNotFound
	}
	return url, nil
}
