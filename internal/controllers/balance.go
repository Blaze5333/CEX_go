package controllers

import (
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

func Deposit(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.DepositRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}

		if err := q.CreditBalance(req.UserID, req.Asset, req.Amount); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to credit balance"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"message": "Balance credited successfully",
			"user_id": req.UserID,
			"asset":   req.Asset,
			"amount":  req.Amount,
		})
	}
}

// GetBalances returns all asset balances for a user.
func GetBalances(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.Param("user_id")
		balances, err := q.GetBalances(userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch balances"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"balances": balances})
	}
}
