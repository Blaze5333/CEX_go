package controllers

import (
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/auth"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

func RegisterUser(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.UserRegistrationRequest
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		id, _, err := q.GetUserByEmail(user.Email)
		//checking if user already exists
		if err == nil || id != "" {
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists", "message": "Email is already registered"})
			return
		}
		passwordHash, err := auth.HashPassword(user.Password)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to hash password"})
			return
		}
		id, err = q.CreateUser(user.Email, passwordHash)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to create user"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "email": user.Email})
	}
}

func LoginUser(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var user models.UserLoginRequest
		if err := c.ShouldBindJSON(&user); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		id, passwordHash, err := q.GetUserByEmail(user.Email)
		if err != nil || id == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "message": "User not found"})
			return
		}
		if !auth.CheckPasswordHash(user.Password, passwordHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "message": "Incorrect password"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"id": id, "email": user.Email})
	}
}
