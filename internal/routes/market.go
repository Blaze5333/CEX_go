package routes

// Routes: GET /markets (public), GET /markets/:id (public), POST /markets (admin only)

import (
	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/controllers"
	"github.com/Blaze5333/cex/internal/middleware"
	"github.com/gin-gonic/gin"
)

func MarketRoutes(incomingRoutes *gin.Engine, q *queries.Queries) {
	incomingRoutes.GET("/markets", controllers.GetMarkets(q))
	incomingRoutes.GET("/markets/:id", controllers.GetMarket(q))
}

//in golang how to continue a function which is in another file but in same package?

func MarketAdminRoutes(incomingRoutes *gin.Engine, q *queries.Queries) {
	incomingRoutes.POST("/assets", middleware.VerifyAdmin(q), controllers.CreateAsset(q)) // create new asset
}
