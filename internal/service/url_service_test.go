package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"url-shortener/internal/storage"
)

func TestShortenURLSuccess(t *testing.T) {
	ctx := context.Background()
	s := storage.NewInMemoryStorage()
	svc := NewURLService(s)

	originalURL := "https://example.com"

	code, err := svc.ShortenURL(ctx, originalURL)
	if err != nil {
		t.Fatalf("shorting URL failed: %v", err)
	}

	if code == "" {
		t.Fatal("expected non-empty short code")
	}

	retrievedURL, err := svc.GetOriginalURL(ctx, code)
	if err != nil {
		t.Fatalf("getting original URL failed: %v", err)
	}
	if retrievedURL != originalURL {
		t.Fatalf("expected %s got %s", originalURL, retrievedURL)
	}

}

func TestShortenURLNormalization(t *testing.T) {
	ctx := context.Background()
	s := storage.NewInMemoryStorage()
	svc := NewURLService(s)

	originalURL := "https://EXAMPLE.com"

	code, err := svc.ShortenURL(ctx, originalURL)
	if err != nil {
		t.Fatalf("shorting URL failed: %v", err)
	}

	if code == "" {
		t.Fatal("expected non-empty short code")
	}

	retrievedURL, err := svc.GetOriginalURL(ctx, code)
	if err != nil {
		t.Fatalf("getting original URL failed: %v", err)
	}
	if retrievedURL != strings.ToLower(originalURL) {
		t.Fatalf("expected %s got %s", originalURL, retrievedURL)
	}

}
func TestShortenURLEmpty(t *testing.T) {
	ctx := context.Background()
	s := storage.NewInMemoryStorage()
	svc := NewURLService(s)

	originalURL := ""

	_, err := svc.ShortenURL(ctx, originalURL)
	if err == nil {
		t.Fatal("expected error for empty got nil")
	}
	if !errors.Is(err, ErrEmptyURL) {
		t.Fatalf("expected ErrEmptyURL got: %v", err)
	}

}
func TestShortenURLInvalid(t *testing.T) {
	ctx := context.Background()
	s := storage.NewInMemoryStorage()
	svc := NewURLService(s)

	cases := []string{
		"htp://example.com",
		"example.com",
		"ftp://example.com",
		"http//example.com",
		"http://exam ple.com",

		strings.Repeat("a", maxURLLength+1),
	}
	for _, originalURL := range cases {
		_, err := svc.ShortenURL(ctx, originalURL)
		if err == nil {
			t.Fatal("expected error for invalid url got nil")
		}
		if !errors.Is(err, ErrInvalidURL) {
			t.Fatalf("expected ErrInvalidURL got: %v", err)
		}

	}

}
