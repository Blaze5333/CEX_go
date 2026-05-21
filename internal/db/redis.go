package db

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Blaze5333/cex/db/queries"
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

func GetRedisClient() *redis.Client {
	return rdb
}
func (c *RedisConfig) Close() error {
	return rdb.Close()
}
func (c *RedisConfig) FlushDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return rdb.FlushDB(ctx).Err()
}
func (c *RedisConfig) LoadOrderBookToRedis(ctx context.Context, query *queries.Queries) error {
	pipeline := rdb.Pipeline()
	order, err := query.GetAllActiveOrders()
	if err != nil {
		log.Printf("Failed to load order book to redis: %v", err)
		return err
	}
	for _, o := range order {
		log.Printf("Loading order to redis: ID=%s MarketID=%s Side=%s Price=%f Quantity=%f", o.ID, o.MarketID, o.Side, o.Price, o.Quantity)
		bookKey := fmt.Sprintf("orderbook:%s:%s", o.MarketID, o.Side)
		pipeline.ZAdd(ctx, bookKey, redis.Z{
			Score:  buildScore(o.Price, o.CreatedAt.UnixMilli()),
			Member: o.ID,
		})
		detailsKey := fmt.Sprintf("order:%s", o.ID)
		pipeline.HSet(ctx, detailsKey, map[string]interface{}{
			"user_id":         o.UserID,
			"market_id":       o.MarketID,
			"order_type":      o.OrderType,
			"side":            o.Side,
			"price":           o.Price,
			"quantity":        o.Quantity,
			"status":          o.Status,
			"created_at":      o.CreatedAt.UnixMilli(),
			"filled_quantity": o.FilledQuantity,
		})

	}
	_, err = pipeline.Exec(ctx)
	if err != nil {
		log.Printf("Failed to execute pipeline for loading order book to redis: %v", err)
	}
	return err
}
func buildScore(price float64, createdAt int64) float64 {

	const baseTime = 1700000000000 // some past epoch in ms
	timeFraction := float64(createdAt-baseTime) / 1e13
	return price + timeFraction
}
