package middleware

import (
	"log"
	"strings"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/auth"
	"github.com/gin-gonic/gin"
)

const middlewareTag = "[middleware/verifyUser]"

func VerifyUser(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s VerifyUser: verifying request to %s", middlewareTag, c.Request.URL.Path)

		tokenString := c.GetHeader("Authorization")
		if tokenString == "" {
			log.Printf("%s VerifyUser: authorization header missing for %s", middlewareTag, c.Request.URL.Path)
			c.JSON(401, gin.H{"error": "Authorization header is missing", "message": "Please provide a valid token"})
			c.Abort()
			return
		}
		token := strings.Split(tokenString, " ")[1]
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			log.Printf("%s VerifyUser: invalid token for %s: %v", middlewareTag, c.Request.URL.Path, err)
			c.JSON(401, gin.H{"error": err.Error(), "message": "Invalid token"})
			c.Abort()
			return
		}

		email, err := q.GetUserByID(claims.UserID)
		if err != nil {
			log.Printf("%s VerifyUser: user not found for userID=%s: %v", middlewareTag, claims.UserID, err)
			c.JSON(401, gin.H{"error": "User not found", "message": "Invalid token"})
			c.Abort()
			return
		}
		log.Printf("%s VerifyUser: authenticated userID=%s email=%s", middlewareTag, claims.UserID, email)
		c.Set("user_id", claims.UserID)
		c.Set("email", email)
		c.Next()
	}
}
