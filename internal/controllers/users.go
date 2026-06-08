package controllers

import (
	"log"
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/auth"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

const usersCtrlTag = "[controllers/users]"

func RegisterUser(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s RegisterUser: handling registration request", usersCtrlTag)
		var user models.UserRegistrationRequest
		if err := c.ShouldBindJSON(&user); err != nil {
			log.Printf("%s RegisterUser: invalid request body: %v", usersCtrlTag, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		id, _, err := q.GetUserByEmail(user.Email)
		if err == nil || id != "" {
			log.Printf("%s RegisterUser: user already exists with email=%s", usersCtrlTag, user.Email)
			c.JSON(http.StatusConflict, gin.H{"error": "User already exists", "message": "Email is already registered"})
			return
		}
		passwordHash, err := auth.HashPassword(user.Password)
		if err != nil {
			log.Printf("%s RegisterUser: failed to hash password for email=%s: %v", usersCtrlTag, user.Email, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to hash password"})
			return
		}
		id, err = q.CreateUser(user.Email, passwordHash)
		if err != nil {
			log.Printf("%s RegisterUser: failed to create user with email=%s: %v", usersCtrlTag, user.Email, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to create user"})
			return
		}
		log.Printf("%s RegisterUser: successfully registered user id=%s email=%s", usersCtrlTag, id, user.Email)
		c.JSON(http.StatusCreated, gin.H{"id": id, "email": user.Email})
	}
}

func LoginUser(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s LoginUser: handling login request", usersCtrlTag)
		var user models.UserLoginRequest
		if err := c.ShouldBindJSON(&user); err != nil {
			log.Printf("%s LoginUser: invalid request body: %v", usersCtrlTag, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		id, passwordHash, err := q.GetUserByEmail(user.Email)
		if err != nil || id == "" {
			log.Printf("%s LoginUser: user not found for email=%s", usersCtrlTag, user.Email)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "message": "User not found"})
			return
		}
		if !auth.CheckPasswordHash(user.Password, passwordHash) {
			log.Printf("%s LoginUser: incorrect password for email=%s", usersCtrlTag, user.Email)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password", "message": "Incorrect password"})
			return
		}
		dbUser, err := q.GetUserByID(id)
		if err != nil {
			log.Printf("%s LoginUser: failed to fetch user role for userID=%s email=%s: %v", usersCtrlTag, id, user.Email, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch user profile"})
			return
		}
		authToken, err := auth.GenerateJWT(id, user.Email)
		if err != nil {
			log.Printf("%s LoginUser: failed to generate auth token for userID=%s email=%s: %v", usersCtrlTag, id, user.Email, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to generate auth token"})
			return
		}
		log.Printf("%s LoginUser: successful login for userID=%s email=%s", usersCtrlTag, id, user.Email)
		c.JSON(http.StatusOK, gin.H{"id": id, "email": user.Email, "role": dbUser.Role, "token": authToken})
	}
}
