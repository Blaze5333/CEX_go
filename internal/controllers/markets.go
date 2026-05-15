package controllers

import (
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

func GetMarkets(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		markets, err := q.GetAllMarkets()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch markets"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"markets": markets})
	}
}

func GetMarket(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		market, err := q.GetMarketByID(id)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error(), "message": "Market not found"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"market": market})
	}
}

func CreateMarket(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req models.CreateMarketRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		id, err := q.CreateMarket(req.Name, req.BaseAsset, req.QuoteAsset, req.MinOrderSize, req.MaxOrderSize, req.TakerFee, req.MakerFee)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to create market"})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
	}
}
