package cache

import "github.com/go-redis/redis/v8"

func NewClient(addr string) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: addr})
}
