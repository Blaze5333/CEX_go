package main

import (
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

	q := queries.New(database)

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
