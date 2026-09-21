package storage

import (
	"context"
	"errors"
	"log"
	"os"
	"sync/atomic"
	"time"
	"url-shortener/internal/metrics"
	"url-shortener/internal/models"

	"github.com/redis/go-redis/v9"
)

func SetupRedis() *redis.Client {
	addr := os.Getenv("REDIS_URL")
	if addr == "" {
		addr = "localhost:6379"
	}
	return redis.NewClient(&redis.Options{Addr: addr})
}

type CachedStorage struct {
	base  Storage
	redis *redis.Client

	hits int64
	miss int64
}

func NewCachedStorage(base Storage, redis *redis.Client) *CachedStorage {
	return &CachedStorage{
		base:  base,
		redis: redis,
	}
}

var nullCacheValue = "NULL"

func (s *CachedStorage) GetURL(ctx context.Context, shortCode string) (string, error) {
	cachedURL, err := s.redis.Get(ctx, shortCode).Result()
	if err == nil {
		if cachedURL == nullCacheValue {
			atomic.AddInt64(&s.miss, 1)
			metrics.CacheMisses.Inc()
			return "", ErrURLNotFound
		}
		atomic.AddInt64(&s.hits, 1)
		metrics.CacheHits.Inc()
		return cachedURL, nil
	} else if !errors.Is(err, redis.Nil) {
		log.Printf("redis get URL error: %v", err)
	}
	metrics.CacheMisses.Inc()
	atomic.AddInt64(&s.miss, 1)
	url, err := s.base.GetURL(ctx, shortCode)
	if err != nil {
		if errors.Is(err, ErrURLNotFound) {
			if err = s.redis.Set(ctx, shortCode, nullCacheValue, time.Minute).Err(); err != nil {
				log.Printf("redis set NULL error: %v", err)
			}
		}

		return "", err
	}
	if err = s.redis.Set(ctx, shortCode, url, time.Minute*5).Err(); err != nil {
		log.Printf("failed to cache URL in Redis: %v", err)
	}

	return url, nil
}

func (s *CachedStorage) UpdateCode(ctx context.Context, id int64, shortCode string) error {
	err := s.base.UpdateCode(ctx, id, shortCode)
	if err != nil {
		return err
	}
	_, err = s.redis.Del(ctx, shortCode).Result()
	if err != nil {
		log.Printf("redis del URL error: %v", err)
	}
	return nil
}

func (s *CachedStorage) CreateURL(ctx context.Context, originalURL string, userID int64) (int64, error) {
	return s.base.CreateURL(ctx, originalURL, userID)
}

func (s *CachedStorage) Stats() (hits int64, miss int64, hitRate float64) {
	h := atomic.LoadInt64(&s.hits)
	m := atomic.LoadInt64(&s.miss)
	total := h + m
	if total == 0 {
		return h, m, 0
	}
	return h, m, float64(h) / float64(total)
}
func (s *CachedStorage) GetCodeByURL(ctx context.Context, originalURL string) (string, error) {
	cacheKey := "url:" + originalURL
	cachedCode, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		if cachedCode == nullCacheValue {
			atomic.AddInt64(&s.miss, 1)
			metrics.CacheMisses.Inc()
			return "", ErrURLNotFound
		}
		atomic.AddInt64(&s.hits, 1)
		metrics.CacheHits.Inc()
		return cachedCode, nil
	} else if !errors.Is(err, redis.Nil) {
		log.Printf("redis get code by URL error: %v", err)
	}
	metrics.CacheMisses.Inc()
	atomic.AddInt64(&s.miss, 1)
	code, err := s.base.GetCodeByURL(ctx, originalURL)
	if err != nil {
		if errors.Is(err, ErrURLNotFound) {
			if err = s.redis.Set(ctx, cacheKey, nullCacheValue, time.Minute).Err(); err != nil {
				log.Printf("redis set NULL error: %v", err)
			}
		}
		return "", err
	}
	if err = s.redis.Set(ctx, cacheKey, code, time.Minute*10).Err(); err != nil {
		log.Printf("failed to cache code in Redis: %v", err)
	}
	return code, nil
}

func (s *CachedStorage) GetURLsByUserID(ctx context.Context, userID int64) ([]models.URL, error) {
	return s.base.GetURLsByUserID(ctx, userID)
}
