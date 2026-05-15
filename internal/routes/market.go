package routes

// Routes: GET /markets (public), GET /markets/:id (public), POST /markets (admin only)

import (
	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/controllers"
	"github.com/gin-gonic/gin"
)

func MarketRoutes(incomingRoutes *gin.Engine, q *queries.Queries) {
	incomingRoutes.GET("/markets", controllers.GetMarkets(q))
	incomingRoutes.GET("/markets/:id", controllers.GetMarket(q))
	incomingRoutes.POST("/markets", controllers.CreateMarket(q)) // TODO: add admin middleware
}
