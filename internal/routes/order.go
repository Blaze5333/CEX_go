package routes

import (
	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/controllers"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/matching"
	"github.com/Blaze5333/cex/internal/middleware"
	"github.com/gin-gonic/gin"
)

func OrderRoutes(incomingRoutes *gin.Engine, q *queries.Queries, redisClient *db.RedisConfig, matchingConfig *matching.MatchingEngine) {
	incomingRoutes.POST("/orders", middleware.VerifyUser(q), controllers.CreateOrder(q, redisClient, matchingConfig)) // TODO: add auth middleware
	incomingRoutes.GET("/trades", middleware.VerifyUser(q), controllers.GetUserTrades(q))
	// incomingRoutes.GET("/orders/:id", controllers.GetOrder(q))                // TODO: add auth middleware
	incomingRoutes.GET("/order_book/:market_id", controllers.GetOrderBook(q, redisClient)) // TODO: add auth middleware
	incomingRoutes.GET("/active_user_orders", middleware.VerifyUser(q), controllers.GetActiveUserOrders(q))
}
