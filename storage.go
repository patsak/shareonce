package main

import (
	"context"
	"time"

	redis "github.com/go-redis/redis/v9"
)

type storage struct {
	client *redis.Client
}

const defaultTTL = 3 * 24 * time.Hour

func NewStorage(addr string) *storage {
	return &storage{
		client: redis.NewClient(&redis.Options{
			Addr: addr,
		}),
	}
}

func (s *storage) Put(ctx context.Context, key string, value string) error {
	if err := s.client.Set(ctx, key, value, defaultTTL).Err(); err != nil {
		return err
	}
	return nil
}

func (s *storage) Get(ctx context.Context, key string) (string, error) {
	return s.client.Get(ctx, key).Result()
}

func (s *storage) Delete(ctx context.Context, key string) error {
	return s.client.Del(ctx, key).Err()
}
