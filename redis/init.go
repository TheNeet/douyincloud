package redis

import (
	"context"
	"os"

	"github.com/go-redis/redis/v8"
)

var client *redis.Client

func Init() {
	user := os.Getenv("REDIS_USERNAME")
	pwd := os.Getenv("REDIS_PASSWORD")
	addr := os.Getenv("REDIS_ADDRESS")
	client = redis.NewClient(&redis.Options{
		Username: user,
		Password: pwd,
		Addr:     addr,
		DB:       1,
	})
	if res := client.Ping(context.Background()); res.Err() != nil {
		panic(res.Err())
	}
}
