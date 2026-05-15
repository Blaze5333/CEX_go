package routes

import (
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/controllers"
	"github.com/gin-gonic/gin"
)

func UserRoutes(incomingRoutes *gin.Engine, q *queries.Queries) {
	incomingRoutes.POST("/register", controllers.RegisterUser(q))
	incomingRoutes.POST("/login", controllers.LoginUser(q))
}
