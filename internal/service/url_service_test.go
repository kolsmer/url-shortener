package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"url-shortener/internal/storage"
	"url-shortener/internal/models"
)

type mockStorage struct {
	createURL func(ctx context.Context, originalURL string, userID int64) (int64, error)
	getURL func(ctx context.Context, shortCode string) (string, error)
	updateCode func(ctx context.Context, id int64, shortCode string) error
	getCodeByURL func(ctx context.Context, originalURL string) (string, error)
	getUrlsByUserID func(ctx context.Context, userID int64) ([]models.URL, error)
}

func (m *mockStorage) CreateURL(ctx context.Context, originalURL string, userID int64) (int64, error) {
	return m.createURL(ctx, originalURL, userID)
}

func (m *mockStorage) GetURL(ctx context.Context, shortCode string) (string, error){
	return m.getURL(ctx, shortCode)
}

func (m *mockStorage) UpdateCode(ctx context.Context, id int64, shortCode string) error {
	return m.updateCode(ctx, id, shortCode)
}

func (m *mockStorage) GetCodeByURL(ctx context.Context, originalURL string) (string, error) {
	return m.getCodeByURL(ctx, originalURL)
}

func (m *mockStorage) GetURLsByUserID(ctx context.Context, userID int64) ([]models.URL, error) {
	return m.getUrlsByUserID(ctx, userID)
}


func TestShortenURL_NewURL(t *testing.T) {
	mock := &mockStorage{
		getCodeByURL: func(ctx context.Context, url string) (string, error) { 
			return "", storage.ErrURLNotFound
		},
		createURL: func(ctx context.Context, url string, userID int64) (int64, error) {
			return 1, nil
		},
		updateCode: func(ctx context.Context, id int64, shortCode string) error {
			return nil
		},
	}

	svc := NewURLService(mock)
	code, err := svc.ShortenURL(context.Background(), "https://example.com", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	} 
	if code == "" {
		t.Fatal("expected non-empty short code")
	}
}

func TestShortenURL_DuplicateURL(t *testing.T) {
	mock := &mockStorage{
		getCodeByURL: func(ctx context.Context, url string) (string, error) {
			return "abc123", nil
		},
	}

	svc := NewURLService(mock)
	code, err := svc.ShortenURL(context.Background(), "https://example.com", 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if code != "abc123" {
		t.Fatalf("expected code 'abc123', got %s", code)
	}
}

func TestShortenURL_Normalization(t *testing.T) {
	var capturedURL string
	mock := &mockStorage{
		getCodeByURL: func(ctx context.Context, url string) (string, error) {
			return "", storage.ErrURLNotFound
		},
		createURL: func(ctx context.Context, url string, userID int64) (int64, error) {
			capturedURL = url
			return 1, nil
		},
		updateCode: func(ctx context.Context, id int64, shortCode string) error {
			return nil
		},
	}
		
	svc := NewURLService(mock)

	code, err := svc.ShortenURL(context.Background(), "https://EXAMPLE.com", 1)
	if err != nil {
		t.Fatalf("shorting URL failed: %v", err)
	}

	if code == "" {
		t.Fatal("expected non-empty short code")
	}

	if capturedURL != "https://example.com" { 
		t.Fatalf("expected normalized url, got %s", capturedURL)
	}

}
func TestShortenURLEmpty(t *testing.T) {
	ctx := context.Background()
	svc := NewURLService(&mockStorage{})

	originalURL := ""

	_, err := svc.ShortenURL(ctx, originalURL, 1)
	if err == nil {
		t.Fatal("expected error for empty got nil")
	}
	if !errors.Is(err, ErrEmptyURL) {
		t.Fatalf("expected ErrEmptyURL got: %v", err)
	}

}
func TestShortenURLInvalid(t *testing.T) {
	ctx := context.Background()
	svc := NewURLService(&mockStorage{})

	cases := []string{
		"htp://example.com",
		"example.com",
		"ftp://example.com",
		"http//example.com",
		"http://exam ple.com",

		strings.Repeat("a", maxURLLength+1),
	}
	for _, originalURL := range cases {
		_, err := svc.ShortenURL(ctx, originalURL, 1)
		if err == nil {
			t.Fatal("expected error for invalid url got nil")
		}
		if !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("expected ErrInvalidURL got: %v", err)
		}

	}

}
