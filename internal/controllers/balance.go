package controllers

import (
	"log"
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

const balanceCtrlTag = "[controllers/balance]"

func Deposit(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s Deposit: handling deposit request", balanceCtrlTag)
		var req models.DepositRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("%s Deposit: invalid request body: %v", balanceCtrlTag, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		userId, exists := c.Get("user_id")
		if !exists {
			log.Printf("%s Deposit: user_id not found in context", balanceCtrlTag)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context", "message": "Unauthorized"})
			return
		}

		log.Printf("%s Deposit: crediting userID=%s asset=%s amount=%f", balanceCtrlTag, userId.(string), req.Asset, req.Amount)
		if err := q.CreditBalance(userId.(string), req.Asset, req.Amount); err != nil {
			log.Printf("%s Deposit: failed to credit balance for userID=%s asset=%s: %v", balanceCtrlTag, userId.(string), req.Asset, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to credit balance"})
			return
		}

		log.Printf("%s Deposit: successfully deposited asset=%s amount=%f for userID=%s", balanceCtrlTag, req.Asset, req.Amount, userId.(string))
		c.JSON(http.StatusOK, gin.H{
			"message": "Balance credited successfully",
			"user_id": userId.(string),
			"asset":   req.Asset,
			"amount":  req.Amount,
		})
	}
}

// GetBalances returns all asset balances for a user.
func GetBalances(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s GetBalances: handling get balances request", balanceCtrlTag)
		userID, exists := c.Get("user_id")
		if !exists {
			log.Printf("%s GetBalances: user_id not found in context", balanceCtrlTag)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context", "message": "Unauthorized"})
			return
		}
		log.Printf("%s GetBalances: fetching balances for userID=%s", balanceCtrlTag, userID.(string))
		balances, err := q.GetBalances(userID.(string))
		if err != nil {
			log.Printf("%s GetBalances: failed to fetch balances for userID=%s: %v", balanceCtrlTag, userID.(string), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch balances"})
			return
		}
		log.Printf("%s GetBalances: returning %d balance(s) for userID=%s", balanceCtrlTag, len(balances), userID.(string))
		c.JSON(http.StatusOK, gin.H{"balances": balances})
	}
}
