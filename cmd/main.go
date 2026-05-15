package main

import (
	"fmt"
	"log"
	"os"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/routes"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	//load env
	godotenv.Load()
	database, err := db.NewPostgres(db.PostgresConfig{
		DSN: os.Getenv("POSTGRES_URI"),
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print("Postgres db connected")
	defer database.Close()

	q := queries.New(database)

	r := gin.Default()
	routes.UserRoutes(r, q)
	routes.MarketRoutes(r, q)
	routes.BalanceRoutes(r, q)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Fatal(r.Run(":" + port))
}
