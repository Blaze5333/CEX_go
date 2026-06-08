package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/matching"
	"github.com/Blaze5333/cex/internal/routes"
	"github.com/Blaze5333/cex/internal/ws"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"golang.org/x/net/websocket"
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
	matchingConfig := matching.MatchingEngine{
		Rdb: &redisConfig,
		DB:  *q,
	}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if origin == "" {
				return true
			}
			if origin == "http://localhost:8080" || origin == "http://localhost:5173" {
				return true
			}
			return strings.HasPrefix(origin, "https://") &&
				(origin == "https://blink-trade-hub.lovable.app" || strings.HasSuffix(origin, ".lovable.app"))
		},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))
	log.Printf("%s registering routes", mainTag)
	routes.UserRoutes(r, q)
	routes.MarketRoutes(r, q)
	routes.BalanceRoutes(r, q)
	routes.BalanceAdminRoutes(r, q)
	routes.MarketAdminRoutes(r, q)
	routes.OrderRoutes(r, q, &redisConfig, &matchingConfig)
	wsServer := ws.WSServer{
		Rdb:   redisClient,
		Rooms: make(map[string]*ws.Room),
	}
	//websocket server
	r.GET("/ws/:marketId", func(c *gin.Context) {
		marketId := c.Param("marketId")
		server := websocket.Server{
			Handshake: func(*websocket.Config, *http.Request) error {
				return nil
			},
			Handler: func(conn *websocket.Conn) {
				wsServer.HandleConnection(conn, marketId)
			},
		}
		server.ServeHTTP(c.Writer, c.Request)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("%s server started on port %s", mainTag, port)
	log.Fatal(r.Run(":" + port))
}
