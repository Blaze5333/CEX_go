package main

import (
	"context"
	"log"
	"os"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

const mainTag = "[main]"

func main() {
	ctx := context.Background()
	defer ctx.Done()
	log.Printf("%s starting CEX application", mainTag)

	if err := godotenv.Load(); err != nil {
		log.Printf("%s .env file not found, using environment variables: %v", mainTag, err)
	}

	log.Printf("%s connecting to postgres", mainTag)
	database, err := db.NewPostgres(db.PostgresConfig{
		DSN: os.Getenv("POSTGRES_URI"),
	})
	if err != nil {
		log.Fatalf("%s failed to connect to postgres: %v", mainTag, err)
	}
	log.Printf("%s postgres connected successfully", mainTag)
	defer database.Close()
	log.Printf("%s connecting to redis", mainTag)
	redisConfig := db.RedisConfig{
		Addr:     os.Getenv("REDIS_ADDR"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	}
	redisClient, err := redisConfig.NewRedisClient()
	if err != nil {
		log.Fatalf("%s failed to connect to redis: %v", mainTag, err)
	}
	log.Printf("%s redis connected successfully", mainTag)
	defer redisClient.Close()

	q := queries.New(database)
	log.Printf("%s loading order book to redis", mainTag)

	if err := redisConfig.LoadOrderBookToRedis(ctx, q); err != nil {
		log.Printf("%s failed to load order book to redis: %v", mainTag, err)
	} else {
		log.Printf("%s order book loaded to redis successfully", mainTag)
	}

	r := gin.Default()
	log.Printf("%s registering routes", mainTag)
	routes.UserRoutes(r, q)
	routes.MarketRoutes(r, q)
	routes.BalanceRoutes(r, q)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("%s server starting on port %s", mainTag, port)
	log.Fatal(r.Run(":" + port))
}
