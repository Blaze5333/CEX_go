package db

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

var rdb *redis.Client

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

func (c *RedisConfig) NewRedisClient() (*redis.Client, error) {
	rdb = redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pingResponse, err := rdb.Ping(ctx).Result()
	println("Ping response from redis:", pingResponse)
	return rdb, err
}
