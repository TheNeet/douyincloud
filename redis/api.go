package redis

import (
	"context"
)

func Push(ctx context.Context, key string, val string) error {
	return client.RPush(ctx, key, val).Err()
}

func Pop(ctx context.Context, key string) ([]string, error) {
	return client.LPopCount(ctx, key, -1).Result()
}
