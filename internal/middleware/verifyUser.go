package middleware

import (
	"strings"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/auth"
	"github.com/gin-gonic/gin"
)

func VerifyUser(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {

		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			c.JSON(401, gin.H{"error": "Authorization header is missing", "message": "Please provide a valid token"})
			c.Abort()
			return
		}
		token := strings.Split(tokenString, " ")[1]
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			c.JSON(401, gin.H{"error": err.Error(), "message": "Invalid token"})
			c.Abort()
			return
		}

		email, err := q.GetUserByID(claims.UserID)
		if err != nil {
			c.JSON(401, gin.H{"error": "User not found", "message": "Invalid token"})
			c.Abort()
			return
		}
		c.Set("user_id", claims.UserID)
		c.Set("email", email)
		c.Next()
	}
}
