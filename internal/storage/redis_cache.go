package storage

import (
	"context"
	"errors"
	"github.com/redis/go-redis/v9"
	"log"
	"sync/atomic"
	"time"
	"url-shortener/internal/metrics"
)

func SetupRedis() *redis.Client {
	return redis.NewClient(&redis.Options{Addr: "localhost:6379"})
}

type CachedStorage struct {
	base  Storage
	redis *redis.Client

	hits int64
	miss int64
}

func NewCachedStorage(redis *redis.Client) *CachedStorage {
	return &CachedStorage{}
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

func (s *CachedStorage) UpdateURL(ctx context.Context, id int64, shortCode string) error {
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

func (s *CachedStorage) Stats() (hits int64, miss int64, hitRate float64) {
	h := atomic.LoadInt64(&s.hits)
	m := atomic.LoadInt64(&s.miss)
	total := h + m
	if total == 0 {
		return h, m, 0
	}
	return h, m, float64(h) / float64(total)
}
