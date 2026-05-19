package controllers

import (
	"log"
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

const marketsCtrlTag = "[controllers/markets]"

func GetMarkets(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s GetMarkets: fetching all markets", marketsCtrlTag)
		markets, err := q.GetAllMarkets()
		if err != nil {
			log.Printf("%s GetMarkets: failed to fetch markets: %v", marketsCtrlTag, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch markets"})
			return
		}
		log.Printf("%s GetMarkets: returning %d market(s)", marketsCtrlTag, len(markets))
		c.JSON(http.StatusOK, gin.H{"markets": markets})
	}
}

func GetMarket(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		log.Printf("%s GetMarket: fetching market id=%s", marketsCtrlTag, id)
		market, err := q.GetMarketByID(id)
		if err != nil {
			log.Printf("%s GetMarket: market not found id=%s: %v", marketsCtrlTag, id, err)
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error(), "message": "Market not found"})
			return
		}
		log.Printf("%s GetMarket: found market id=%s name=%s", marketsCtrlTag, market.ID, market.Name)
		c.JSON(http.StatusOK, gin.H{"market": market})
	}
}

func CreateMarket(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s CreateMarket: handling create market request", marketsCtrlTag)
		var req models.CreateMarketRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("%s CreateMarket: invalid request body: %v", marketsCtrlTag, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		log.Printf("%s CreateMarket: creating market name=%s baseAsset=%s quoteAsset=%s", marketsCtrlTag, req.Name, req.BaseAsset, req.QuoteAsset)
		id, err := q.CreateMarket(req.Name, req.BaseAsset, req.QuoteAsset, req.MinOrderSize, req.MaxOrderSize, req.TakerFee, req.MakerFee)
		if err != nil {
			log.Printf("%s CreateMarket: failed to create market name=%s: %v", marketsCtrlTag, req.Name, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to create market"})
			return
		}
		log.Printf("%s CreateMarket: successfully created market id=%s name=%s", marketsCtrlTag, id, req.Name)
		c.JSON(http.StatusCreated, gin.H{"id": id, "name": req.Name})
	}
}
