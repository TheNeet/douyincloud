package redis

import (
	"context"
	"os"

	"github.com/go-redis/redis/v8"
)

var (
	Nil    = redis.Nil
	client *redis.Client
)

func Init() {
	user := os.Getenv("REDIS_USERNAME")
	pwd := os.Getenv("REDIS_PASSWORD")
	addr := os.Getenv("REDIS_ADDRESS")
	if user == "" || pwd == "" || addr == "" {
		addr = "127.0.0.1:6379"
	}
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
