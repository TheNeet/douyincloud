package redis

import (
	"context"
	"time"
)

func Get(ctx context.Context, key string) (string, error) {
	return client.Get(ctx, key).Result()
}

func Set(ctx context.Context, key string, val string, expiration time.Duration) error {
	return client.Set(ctx, key, val, expiration).Err()
}

func Del(ctx context.Context, key string) error {
	return client.Del(ctx, key).Err()
}

func Push(ctx context.Context, key string, val string) error {
	return client.RPush(ctx, key, val).Err()
}

func Pop(ctx context.Context, key string) ([]string, error) {
	l, err := client.LLen(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if l == 0 {
		return nil, nil
	}
	return client.LPopCount(ctx, key, int(l)).Result()
}
