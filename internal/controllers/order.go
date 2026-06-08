package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/matching"
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

func CreateOrder(q *queries.Queries, redisClient *db.RedisConfig, matchingConfig *matching.MatchingEngine) gin.HandlerFunc {
	return func(c *gin.Context) {
		log.Printf("%s CreateOrder: handling create order request", orderCtrlTag)
		var req models.CreateOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			log.Printf("%s CreateOrder: invalid request body: %v", orderCtrlTag, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Invalid request"})
			return
		}
		if req.Amount != 0 {
			req.Price = req.Amount
		}
		if req.OrderType == string(models.MARKET) && req.Side == "buy" {
			req.Quantity = 100000
			req.Price = req.Amount
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
		var quantityToLock float64
		if req.Side == "buy" {
			asset = market.QuoteAsset
			if req.OrderType == string(models.MARKET) {
				quantityToLock = req.Amount
			} else {
				quantityToLock = req.Price * req.Quantity
			}
			log.Printf("%s CreateOrder: calculated quote asset quantity to lock for buy order: price=%f quantity=%f total=%f", orderCtrlTag, req.Price, req.Quantity, quantityToLock)
			if quantityToLock <= 0 {
				log.Printf("%s CreateOrder: invalid total cost to lock for buy order: price=%f quantity=%f total=%f", orderCtrlTag, req.Price, req.Quantity, quantityToLock)
				c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid total cost to lock", "message": "Price and quantity must be greater than zero"})
				return
			}
		} else {
			asset = market.BaseAsset
			quantityToLock = req.Quantity
			log.Printf("%s CreateOrder: calculated base asset quantity to lock for sell order: quantity=%f", orderCtrlTag, quantityToLock)
		}

		dbTxn, err := q.GetDB().BeginTx(c.Request.Context(), nil)
		if err != nil {
			log.Printf("%s CreateOrder: failed to begin database transaction: %v", orderCtrlTag, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to start database transaction"})
			return
		}
		committed := false
		defer func() {
			if !committed {
				if rollbackErr := dbTxn.Rollback(); rollbackErr != nil {
					log.Printf("%s CreateOrder: rollback failed: %v", orderCtrlTag, rollbackErr)
				}
			}
		}()

		log.Printf("%s CreateOrder: locking balance for userID=%s asset=%s quantity=%f side=%s", orderCtrlTag, userId.(string), asset, quantityToLock, req.Side)
		err = q.LockBalanceTx(dbTxn, userId.(string), asset, quantityToLock)
		if err != nil {
			log.Printf("%s CreateOrder: failed to lock balance for userID=%s asset=%s: %v", orderCtrlTag, userId.(string), asset, err)
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "message": "Failed to lock balance"})
			return
		}
		log.Printf("%s CreateOrder: creating order for userID=%s marketID=%s type=%s side=%s price=%f quantity=%f", orderCtrlTag, userId.(string), req.MarketID, req.OrderType, req.Side, req.Price, req.Quantity)
		order, err := q.CreateOrderTx(dbTxn, userId.(string), req.MarketID, req.OrderType, req.Side, req.Price, req.Quantity)
		if err != nil {
			log.Printf("%s CreateOrder: failed to create order for userID=%s marketID=%s: %v", orderCtrlTag, userId.(string), req.MarketID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to create order"})
			return
		}
		log.Printf("%s CreateOrder: successfully created order id=%s for userID=%s", orderCtrlTag, order.ID, userId.(string))
		var matchResult matching.OrderResult
		if req.OrderType == string(models.MARKET) {
			log.Printf("%s CreateOrder: matching market order id=%s for userID=%s", orderCtrlTag, order.ID, userId.(string))
			matchResult = matchingConfig.MatchMarketOrders(*order, req.BaseAsset, req.QuoteAsset)
		} else {
			log.Printf("%s CreateOrder: matching limit order id=%s for userID=%s", orderCtrlTag, order.ID, userId.(string))
			matchResult = matchingConfig.MatchLimitOrders(*order, req.BaseAsset, req.QuoteAsset)
		}
		if err := matchingConfig.ApplyOrderResultToDBTx(c.Request.Context(), dbTxn, matchResult); err != nil {
			log.Printf("%s CreateOrder: failed to apply order result to DB for order id=%s: %v", orderCtrlTag, order.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to process order"})
			return
		}
		if err := dbTxn.Commit(); err != nil {
			log.Printf("%s CreateOrder: failed to commit transaction for order id=%s: %v", orderCtrlTag, order.ID, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to commit order"})
			return
		}
		committed = true

		redisCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := matchingConfig.ApplyOrderResultToRedis(redisCtx, matchResult); err != nil {
			log.Printf("%s CreateOrder: order id=%s committed but redis sync failed: %v", orderCtrlTag, order.ID, err)
		}
		log.Printf("%s CreateOrder: successfully processed order id=%s", orderCtrlTag, order.ID)
		c.JSON(http.StatusCreated, gin.H{"id": order.ID})
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

func GetUserTrades(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Printf("%s GetUserTrades: user_id not found in context", orderCtrlTag)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context", "message": "Unauthorized"})
			return
		}
		trades, err := q.GetTradesByUserID(userID.(string))
		if err != nil {
			log.Printf("%s GetUserTrades: failed to fetch trades for userID=%s: %v", orderCtrlTag, userID.(string), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch trades"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"trades": trades})
	}
}

func GetActiveUserOrders(q *queries.Queries) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			log.Printf("%s GetActiveUserOrders: user_id not found in context", orderCtrlTag)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID not found in context", "message": "Unauthorized"})
			return
		}
		log.Printf("%s GetActiveUserOrders: fetching active orders for userID=%s", orderCtrlTag, userID.(string))
		orders, err := q.GetActiveOrdersByUserID(userID.(string))
		if err != nil {
			log.Printf("%s GetActiveUserOrders: failed to fetch active orders for userID=%s: %v", orderCtrlTag, userID.(string), err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to fetch active orders"})
			return
		}
		log.Printf("%s GetActiveUserOrders: returning %d active order(s) for userID=%s", orderCtrlTag, len(orders), userID.(string))
		c.JSON(http.StatusOK, gin.H{"orders": orders})
	}
}

func CancelOrder(q *queries.Queries, redisClient *db.RedisConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderId := c.Param("id")
		txn, err := q.GetDB().BeginTx(c.Request.Context(), nil)
		if err != nil {
			log.Printf("%s CancelOrder: failed to begin transaction for order id=%s: %v", orderCtrlTag, orderId, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to start transaction"})
			return
		}
		defer func() {
			if r := recover(); r != nil {
				if rollbackErr := txn.Rollback(); rollbackErr != nil {
					log.Printf("%s CancelOrder: panic occurred and rollback failed for order id=%s: %v", orderCtrlTag, orderId, rollbackErr)
				} else {
					log.Printf("%s CancelOrder: panic occurred but transaction rolled back successfully for order id=%s", orderCtrlTag, orderId)
				}
				panic(r) // re-throw panic after handling
			}
		}()
		//step to cancel order:
		//unlock balances
		//update order status to cancelled
		//remove from order book in redis
		log.Printf("%s CancelOrder: attempting to cancel order id=%s", orderCtrlTag, orderId)
		order, err := q.UpdateOrderStatusTx(txn, orderId, string(models.CANCELLED))
		if err != nil || order == nil {
			log.Printf("%s CancelOrder: failed to cancel order id=%s", orderCtrlTag, orderId)
			if rollbackErr := txn.Rollback(); rollbackErr != nil {
				log.Printf("%s CancelOrder: rollback failed for order id=%s: %v", orderCtrlTag, orderId, rollbackErr)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to cancel order"})
			return
		}
		if order.Side == "sell" {
			err = q.UnlockBalanceTx(txn, order.UserID, order.BaseAsset, order.Quantity-order.FilledQuantity)
			if err != nil {
				log.Printf("%s CancelOrder: failed to unlock balance for order id=%s: %v", orderCtrlTag, orderId, err)
				if rollbackErr := txn.Rollback(); rollbackErr != nil {
					log.Printf("%s CancelOrder: rollback failed for order id=%s after unlock failure: %v", orderCtrlTag, orderId, rollbackErr)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to unlock balance"})
				return
			}
		} else {
			err = q.UnlockUSDTx(txn, order.UserID, order.Price*(order.Quantity-order.FilledQuantity))
			if err != nil {
				log.Printf("%s CancelOrder: failed to unlock balance for order id=%s: %v", orderCtrlTag, orderId, err)
				if rollbackErr := txn.Rollback(); rollbackErr != nil {
					log.Printf("%s CancelOrder: rollback failed for order id=%s after unlock failure: %v", orderCtrlTag, orderId, rollbackErr)
				}
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to unlock balance"})
				return
			}
		}
		//now remove from redis as well
		redisCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		order.Status = string(models.CANCELLED)
		if err = redisClient.UpdateOrderInRedis(redisCtx, *order); err != nil {
			log.Printf("%s CancelOrder: failed to update order in redis for order id=%s: %v", orderCtrlTag, orderId, err)
			if rollbackErr := txn.Rollback(); rollbackErr != nil {
				log.Printf("%s CancelOrder: rollback failed for order id=%s after redis update failure: %v", orderCtrlTag, orderId, rollbackErr)
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to update order in cache"})
			return
		}
		log.Printf("%s CancelOrder: successfully cancelled order id=%s", orderCtrlTag, orderId)

		if err := txn.Commit(); err != nil {
			log.Printf("%s CancelOrder: failed to commit transaction for order id=%s: %v", orderCtrlTag, orderId, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "message": "Failed to commit transaction"})
			return
		}
		log.Printf("%s CancelOrder: successfully cancelled order id=%s", orderCtrlTag, orderId)
		c.JSON(http.StatusOK, gin.H{"message": "Order cancelled successfully"})
	}
}
