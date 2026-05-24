package controllers

import (
	"log"
	"net/http"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/models"
	"github.com/gin-gonic/gin"
)

const orderCtrlTag = "[controllers/order]"

// 1. Order model + CreateOrderRequest done
// 2. LockBalance query done in balance.go
// 3. CreateOrder query (done in order.go) - insert new order with status "open"
// 4. GetOpenOrdersByMarket query - this is not needed as we will keep the order book in memory, but we can add it later if needed for order history or other features
// 5. Matching engine (in-memory order book)
// 6. CreateTrade + UpdateOrderStatus queries
// 7. UnlockAndTransferBalance query
// 8. Controller + route

func CreateOrder(q *queries.Queries, redisClient *db.RedisConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s CreateOrder: handling create order request", orderCtrlTag)
		var req models.CreateOrderRequest
		dbTxn, err := q.GetDB().Begin()
		if err != nil {
			log.Printf("%s CreateOrder: failed to begin database transaction: %v", orderCtrlTag, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to start database transaction"})
			return
		}
		defer func() {
			if p := recover(); p != nil {
				dbTxn.Rollback()
				panic(p)
			} else if err != nil {
				log.Printf("%s CreateOrder: rolling back transaction due to error: %v", orderCtrlTag, err)
				dbTxn.Rollback()
			} else {
				err = dbTxn.Commit()
				if err != nil {
					log.Printf("%s CreateOrder: failed to commit transaction: %v", orderCtrlTag, err)
				}
			}
		}()
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("%s CreateOrder: invalid request body: %v", orderCtrlTag, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		userId, exists := c.Get("user_id")
		if !exists {
			log.Printf("%s CreateOrder: user_id not found in context", orderCtrlTag)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context", "message": "Unauthorized"})
			return
		}

		log.Printf("%s CreateOrder: fetching market id=%s for userID=%s", orderCtrlTag, req.MarketID, userId.(string))
		market, err := q.GetMarketByID(req.MarketID)
		if err != nil || !market.IsActive {
			log.Printf("%s CreateOrder: invalid or inactive market id=%s: %v", orderCtrlTag, req.MarketID, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid market"})
			return
		}
		var asset string
		if req.Side == "buy" {
			asset = market.QuoteAsset
		} else {
			asset = market.BaseAsset
		}
		log.Printf("%s CreateOrder: locking balance for userID=%s asset=%s quantity=%f side=%s", orderCtrlTag, userId.(string), asset, req.Quantity, req.Side)
		err = q.LockBalance(userId.(string), asset, req.Quantity)
		if err != nil {
			log.Printf("%s CreateOrder: failed to lock balance for userID=%s asset=%s: %v", orderCtrlTag, userId.(string), asset, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Failed to lock balance"})
			return
		}
		log.Printf("%s CreateOrder: creating order for userID=%s marketID=%s type=%s side=%s price=%f quantity=%f", orderCtrlTag, userId.(string), req.MarketID, req.OrderType, req.Side, req.Price, req.Quantity)
		orderId, err := q.CreateOrder(userId.(string), req.MarketID, req.OrderType, req.Side, req.Price, req.Quantity)
		if err != nil {
			log.Printf("%s CreateOrder: failed to create order for userID=%s marketID=%s: %v", orderCtrlTag, userId.(string), req.MarketID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to create order"})
			return
		}
		log.Printf("%s CreateOrder: successfully created order id=%s for userID=%s", orderCtrlTag, orderId, userId.(string))
		c.JSON(http.StatusCreated, gin.H{"id": orderId})
	}
}

func GetOrderBook(q *queries.Queries, redisClient *db.RedisConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		marketId := c.Param("market_id")
		log.Printf("%s GetOrderBook: fetching order book for market id=%s", orderCtrlTag, marketId)
		orderBook, err := redisClient.GetOrderBookFromRedisByMarketId(marketId)
		if err != nil {
			log.Printf("%s GetOrderBook: failed to fetch order book for market id=%s: %v", orderCtrlTag, marketId, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch order book"})
			return
		}
		log.Printf("%s GetOrderBook: successfully fetched order book for market id=%s", orderCtrlTag, marketId)
		c.JSON(http.StatusOK, gin.H{"order_book": orderBook})
	}
}
