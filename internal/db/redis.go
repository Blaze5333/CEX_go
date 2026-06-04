package db

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/redis/go-redis/v9"
)

type RedisConfig struct {
	Addr      string
	Password  string
	DB        int
	RdbClient *redis.Client
}
type MarketDepth struct {
	MarketId string         `json:"market_id"`
	Buys     []models.Order `json:"buys"`  // sorted highest → lowest
	Sells    []models.Order `json:"sells"` // sorted lowest → highest
}

func (c *RedisConfig) NewRedisClient() (*RedisConfig, error) {
	c.RdbClient = redis.NewClient(&redis.Options{
		Addr:     c.Addr,
		Password: c.Password,
		DB:       c.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	pingResponse, err := c.RdbClient.Ping(ctx).Result()
	println("Ping response from redis:", pingResponse)
	return c, err
}

func (c *RedisConfig) GetRedisClient() *redis.Client {
	return c.RdbClient
}
func (c *RedisConfig) Close() error {
	return c.RdbClient.Close()
}
func (c *RedisConfig) FlushDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return c.RdbClient.FlushDB(ctx).Err()
}
func (c *RedisConfig) LoadOrderBookToRedis(ctx context.Context, query *queries.Queries) error {
	pipeline := c.RdbClient.Pipeline()
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

func FetchOrderWithSideFromRedis(ctx context.Context, rdb *redis.Client, key, side string) ([]models.Order, error) {

	entries := []redis.Z{}

	var err error
	if side == "Buy" {
		entries, err = rdb.ZRevRangeWithScores(ctx, key, 0, -1).Result()
	} else {
		entries, err = rdb.ZRangeWithScores(ctx, key, 0, -1).Result()
	}
	if err != nil {
		log.Printf("Failed to fetch orders from redis for key %s: %v", key, err)
		return nil, err
	}
	orders := make([]models.Order, 0, len(entries))
	for _, entry := range entries {
		detailsKey := fmt.Sprintf("order:%s", entry.Member.(string))
		details, err := rdb.HGetAll(ctx, detailsKey).Result()
		if err != nil {
			log.Printf("Failed to get order details from redis for order %s: %v", entry.Member.(string), err)
			continue
		}
		order := models.Order{
			ID:        entry.Member.(string),
			UserID:    details["user_id"],
			MarketID:  details["market_id"],
			OrderType: details["order_type"],
			Side:      details["side"],
			Status:    details["status"],
		}
		fmt.Sscanf(details["price"], "%f", &order.Price)
		fmt.Sscanf(details["quantity"], "%f", &order.Quantity)
		fmt.Sscanf(details["filled_quantity"], "%f", &order.FilledQuantity)
		fmt.Sscanf(details["created_at"], "%d", &order.CreatedAt)
		orders = append(orders, order)
	}
	fmt.Printf("Orders retrieved from redis: %v\n", orders)
	return orders, nil
}

func (c *RedisConfig) GetOrderBookFromRedisByMarketId(marketId string) (*MarketDepth, error) {
	sellKey := fmt.Sprintf("orderbook:%s:Sell", marketId)
	buyKey := fmt.Sprintf("orderbook:%s:Buy", marketId)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	var sells, buys []models.Order
	var sellError, buyError error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		buys, buyError = FetchOrderWithSideFromRedis(ctx, c.RdbClient, buyKey, "Buy")
	}()

	go func() {
		defer wg.Done()
		sells, sellError = FetchOrderWithSideFromRedis(ctx, c.RdbClient, sellKey, "Sell")
	}()

	wg.Wait()

	if sellError != nil {
		log.Printf("Failed to get sell orders from redis: %v", sellError)
		return nil, sellError
	} else {
		log.Printf("Sell orders for market %s: %v", marketId, sells)
	}

	if buyError != nil {
		log.Printf("Failed to get buy orders from redis: %v", buyError)
		return nil, buyError
	} else {
		log.Printf("Buy orders for market %s: %v", marketId, buys)
	}

	return &MarketDepth{
		MarketId: marketId,
		Buys:     buys,
		Sells:    sells,
	}, nil
}
func (C *RedisConfig) BestAskFromRedis(ctx context.Context, marketId string) (*models.Order, error) {
	sellKey := fmt.Sprintf("orderbook:%s:Sell", marketId)
	entries, err := C.RdbClient.ZRangeWithScores(ctx, sellKey, 0, 0).Result()
	if err != nil {
		log.Printf("Failed to fetch best ask from redis for market %s: %v", marketId, err)
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no sell orders in order book for market %s", marketId)
	}
	detailsKey := fmt.Sprintf("order:%s", entries[0].Member.(string))
	details, err := C.RdbClient.HGetAll(ctx, detailsKey).Result()
	if err != nil {
		log.Printf("Failed to get best ask order details from redis for market %s: %v", marketId, err)
		return nil, err
	}
	order := &models.Order{
		ID:        entries[0].Member.(string),
		UserID:    details["user_id"],
		MarketID:  details["market_id"],
		OrderType: details["order_type"],
		Side:      details["side"],
		Status:    details["status"],
	}
	fmt.Sscanf(details["price"], "%f", &order.Price)
	fmt.Sscanf(details["quantity"], "%f", &order.Quantity)
	fmt.Sscanf(details["filled_quantity"], "%f", &order.FilledQuantity)
	fmt.Sscanf(details["created_at"], "%d", &order.CreatedAt)
	return order, nil
}
func (C *RedisConfig) BestBidFromRedis(ctx context.Context, marketId string) (*models.Order, error) {
	buyKey := fmt.Sprintf("orderbook:%s:Buy", marketId)
	entries, err := C.RdbClient.ZRevRangeWithScores(ctx, buyKey, 0, 0).Result()
	if err != nil {
		log.Printf("Failed to fetch best bid from redis for market %s: %v", marketId, err)
		return nil, err
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("no buy orders in order book for market %s", marketId)
	}
	detailsKey := fmt.Sprintf("order:%s", entries[0].Member.(string))
	details, err := C.RdbClient.HGetAll(ctx, detailsKey).Result()
	if err != nil {
		log.Printf("Failed to get best bid order details from redis for market %s: %v", marketId, err)
		return nil, err
	}
	order := &models.Order{
		ID:        entries[0].Member.(string),
		UserID:    details["user_id"],
		MarketID:  details["market_id"],
		OrderType: details["order_type"],
		Side:      details["side"],
		Status:    details["status"],
	}
	fmt.Sscanf(details["price"], "%f", &order.Price)
	fmt.Sscanf(details["quantity"], "%f", &order.Quantity)
	fmt.Sscanf(details["filled_quantity"], "%f", &order.FilledQuantity)
	fmt.Sscanf(details["created_at"], "%d", &order.CreatedAt)
	return order, nil
}
func (C *RedisConfig) UpdateOrderInRedis(ctx context.Context, order models.Order) error {
	bookKey := fmt.Sprintf("orderbook:%s:%s", order.MarketID, order.Side)
	detailsKey := fmt.Sprintf("order:%s", order.ID)
	pipe := C.RdbClient.Pipeline()
	//if order is full filled then we can remove it from the order book sorted set, otherwise we need to update the score if price or created_at has changed and also update the details hash with the new filled quantity and status
	if order.Status == string(models.CANCELLED) || order.FilledQuantity >= order.Quantity {
		pipe.ZRem(ctx, bookKey, order.ID)
		//remove from details hash as well since we won't need it anymore
		pipe.Del(ctx, detailsKey)
		_, err := pipe.Exec(ctx)
		if err != nil {
			log.Printf("Failed to remove order from redis for order %s: %v", order.ID, err)
		}
		return err
	}
	pipe.HSet(ctx, detailsKey, map[string]interface{}{
		"user_id":         order.UserID,
		"market_id":       order.MarketID,
		"order_type":      order.OrderType,
		"side":            order.Side,
		"price":           order.Price,
		"quantity":        order.Quantity,
		"status":          order.Status,
		"created_at":      order.CreatedAt.UnixMilli(),
		"filled_quantity": order.FilledQuantity,
	})
	//why are we updating the score even if price and created_at haven't changed? we can optimize this by only updating the score if price or created_at has changed, but for simplicity let's just update it every time for now
	score := buildScore(order.Price, order.CreatedAt.UnixMilli())
	pipe.ZAdd(ctx, bookKey, redis.Z{
		Score:  score,
		Member: order.ID,
	})

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Failed to update order in redis for order %s: %v", order.ID, err)
	}
	return err
}
