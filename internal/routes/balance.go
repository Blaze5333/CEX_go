package routes

import (
	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/controllers"
	"github.com/Blaze5333/cex/internal/middleware"
	"github.com/gin-gonic/gin"
)

func BalanceRoutes(incomingRoutes *gin.Engine, q *queries.Queries) {
	incomingRoutes.POST("/deposit", middleware.VerifyUser(q), controllers.Deposit(q))     // mock deposit any asset
	incomingRoutes.GET("/balances", middleware.VerifyUser(q), controllers.GetBalances(q)) // get all balances for a user
}
