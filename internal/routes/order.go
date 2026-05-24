package routes

import (
	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/controllers"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/gin-gonic/gin"
)

func OrderRoutes(incomingRoutes *gin.Engine, q *queries.Queries, redisClient *db.RedisConfig) {
	incomingRoutes.POST("/orders", controllers.CreateOrder(q, redisClient)) // TODO: add auth middleware
	// incomingRoutes.GET("/orders/:id", controllers.GetOrder(q))                // TODO: add auth middleware
	incomingRoutes.GET("/order_book/:market_id", controllers.GetOrderBook(q, redisClient)) // TODO: add auth middleware
}
