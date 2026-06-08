package matching

import (
	"context"
	"database/sql"
	"log"
	"time"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/models"
)

type OrderResult struct {
	IncomingOrder models.Order
	UpdatedOrders []models.Order
	Trades        []models.Trade
	BaseAsset     string
	QuoteAsset    string
}
type MatchingEngine struct {
	Rdb *db.RedisConfig
	DB  queries.Queries
}

func (me *MatchingEngine) MatchLimitOrders(order models.Order, baseAsset, quoteAsset string) OrderResult {
	result := OrderResult{IncomingOrder: order, BaseAsset: baseAsset, QuoteAsset: quoteAsset}
	//matching logic here, update result.UpdatedOrders and result.Trades as needed
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for order.FilledQuantity < order.Quantity {
		var bestLevel *models.Order
		var err error
		if order.Side == string(models.BUY) {
			//match with best sell orders
			bestLevel, err = me.Rdb.BestAskFromRedis(ctx, order.MarketID)
			if err != nil {
				break
			}
			//need to hanlde for market orders as well
			if bestLevel == nil || (bestLevel.Price > order.Price) {
				break //no more matches
			}
		} else {
			//match with best buy orders
			bestLevel, err = me.Rdb.BestBidFromRedis(ctx, order.MarketID)
			if err != nil {
				break
			}
			if bestLevel == nil || (bestLevel.Price < order.Price) {
				break //no more matches
			}
		}
		incoming, againstOrder, trade := fillAgainstLevel(order, bestLevel, baseAsset, quoteAsset)
		result.UpdatedOrders = append(result.UpdatedOrders, incoming, againstOrder)
		result.Trades = append(result.Trades, trade)
		order = incoming
		result.IncomingOrder = incoming
	}
	if result.IncomingOrder.FilledQuantity == result.IncomingOrder.Quantity {
		result.IncomingOrder.Status = string(models.FILLED)
	} else if result.IncomingOrder.FilledQuantity > 0 {
		result.IncomingOrder.Status = string(models.PARTIALLY_FILLED)
	} else {
		result.IncomingOrder.Status = string(models.OPEN)
	}

	return result
}
func (me *MatchingEngine) MatchMarketOrders(order models.Order, baseAsset, quoteAsset string) OrderResult {
	result := OrderResult{IncomingOrder: order, BaseAsset: baseAsset, QuoteAsset: quoteAsset}
	//matching logic here, update result.UpdatedOrders and result.Trades as needed
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if order.Side == string(models.SELL) {
		for order.FilledQuantity < order.Quantity {
			bestLevel, err := me.Rdb.BestBidFromRedis(ctx, order.MarketID)
			if err != nil {
				break
			}
			if bestLevel == nil {
				break
			}
			incoming, againstOrder, trade := fillAgainstLevel(order, bestLevel, baseAsset, quoteAsset)
			result.UpdatedOrders = append(result.UpdatedOrders, incoming, againstOrder)
			result.Trades = append(result.Trades, trade)
			order = incoming
			result.IncomingOrder = incoming
		}

		if result.IncomingOrder.FilledQuantity == result.IncomingOrder.Quantity {
			result.IncomingOrder.Status = string(models.FILLED)
		} else {
			result.IncomingOrder.Status = string(models.CANCELLED)
		}
		//remember to unlock the partially filled quantity for market sell order

	} else {
		// currentFilledPrice := 0.0
		//or can we do it like order.Quantity=infinte
		order.Quantity = 1e18 //set to a very large number to represent infinite quantity for market buy order, we will keep matching until we exhaust the price or we fill the quantity
		//this is a buy market order it will not have quantity i has price which is the total amount the buyer is willing to spend, we will keep matching until we fill the quantity or we exhaust the price
		for order.Price > 0 {
			bestLevel, err := me.Rdb.BestAskFromRedis(ctx, order.MarketID)
			if err != nil {
				break
			}
			if bestLevel == nil {
				break
			}
			incoming, againstOrder, trade := fillAgainstLevelForMarketBuy(order, bestLevel, baseAsset, quoteAsset)
			result.UpdatedOrders = append(result.UpdatedOrders, incoming, againstOrder)
			result.Trades = append(result.Trades, trade)
			order = incoming
			result.IncomingOrder = incoming
		}
		if result.IncomingOrder.FilledQuantity == result.IncomingOrder.Quantity {
			result.IncomingOrder.Status = string(models.FILLED)
		} else {
			result.IncomingOrder.Status = string(models.CANCELLED)
		}
	}
	return result
}

func (me *MatchingEngine) ApplyOrderResultToDBTx(ctx context.Context, tx *sql.Tx, result OrderResult) error {
	insertedTrades := make([]models.Trade, 0, len(result.Trades))
	for index, trade := range result.Trades {
		tradeFromDB, err := me.DB.InsertTradeTx(tx, trade)
		if err != nil {
			log.Printf("Index : %v Failed to insert trade into DB: %v", index, err)
			return err
		}
		insertedTrades = append(insertedTrades, tradeFromDB)
	}

	for _, order := range result.UpdatedOrders {
		if order.ID == result.IncomingOrder.ID {
			continue
		}
		if err := me.DB.UpdateOrderStatusAndQuantityTx(tx, order.ID, order.Status, order.FilledQuantity); err != nil {
			log.Printf("Index : %v Failed to update order in DB: %v", order.ID, err)
			return err
		}
	}

	if err := me.DB.UpdateOrderStatusAndQuantityTx(tx, result.IncomingOrder.ID, result.IncomingOrder.Status, result.IncomingOrder.FilledQuantity); err != nil {
		log.Printf("Index : %v Failed to update incoming order in DB: %v", result.IncomingOrder.ID, err)
		return err
	}

	var balanceToCredit, balanceToDebit float64
	for _, trade := range insertedTrades {
		if result.IncomingOrder.Side == string(models.BUY) {
			if err := me.DB.DebitBalanceTx(tx, trade.SellOrder.UserID, trade.BaseAsset, trade.Quantity); err != nil {
				return err
			}
			if err := me.DB.CreditUSDTx(tx, trade.SellOrder.UserID, trade.Price*trade.Quantity); err != nil {
				return err
			}
			balanceToCredit += trade.Quantity
			balanceToDebit += trade.Price * trade.Quantity
		} else {
			if err := me.DB.CreditBalanceTx(tx, trade.BuyOrder.UserID, trade.BaseAsset, trade.Quantity); err != nil {
				return err
			}
			if err := me.DB.DebitUSDTx(tx, trade.BuyOrder.UserID, trade.Price*trade.Quantity); err != nil {
				return err
			}
			balanceToCredit += trade.Price * trade.Quantity
			balanceToDebit += trade.Quantity
		}
	}

	if result.IncomingOrder.Side == string(models.BUY) {
		if err := me.DB.CreditBalanceTx(tx, result.IncomingOrder.UserID, result.BaseAsset, balanceToCredit); err != nil {
			log.Printf("Failed to credit balance for user %s: %v", result.IncomingOrder.UserID, err)
			return err
		}
		if err := me.DB.DebitUSDTx(tx, result.IncomingOrder.UserID, balanceToDebit); err != nil {
			log.Printf("Failed to debit USD for user %s: %v", result.IncomingOrder.UserID, err)
			return err
		}
		if result.IncomingOrder.OrderType == string(models.MARKET) {
			if err := me.DB.UnlockUSDTx(tx, result.IncomingOrder.UserID, result.IncomingOrder.Price); err != nil {
				log.Printf("Failed to unlock USD for user %s: %v", result.IncomingOrder.UserID, err)
				return err
			}
		}
	} else {
		if err := me.DB.CreditUSDTx(tx, result.IncomingOrder.UserID, balanceToCredit); err != nil {
			log.Printf("Failed to credit USD for user %s: %v", result.IncomingOrder.UserID, err)
			return err
		}
		if err := me.DB.DebitBalanceTx(tx, result.IncomingOrder.UserID, result.BaseAsset, balanceToDebit); err != nil {
			log.Printf("Failed to debit balance for user %s: %v", result.IncomingOrder.UserID, err)
			return err
		}
		if result.IncomingOrder.OrderType == string(models.MARKET) {
			if err := me.DB.UnlockBalanceTx(tx, result.IncomingOrder.UserID, result.BaseAsset, result.IncomingOrder.Quantity-result.IncomingOrder.FilledQuantity); err != nil {
				log.Printf("Failed to unlock balance for user %s: %v", result.IncomingOrder.UserID, err)
				return err
			}
		}
	}

	_ = ctx
	return nil
}

func (me *MatchingEngine) ApplyOrderResultToRedis(ctx context.Context, result OrderResult) error {
	for _, order := range result.UpdatedOrders {
		if order.ID == result.IncomingOrder.ID {
			continue
		}
		if err := me.Rdb.UpdateOrderInRedis(ctx, order); err != nil {
			return err
		}
	}
	if result.IncomingOrder.Status != string(models.FILLED) && result.IncomingOrder.OrderType == string(models.LIMIT) {
		if err := me.Rdb.InserOrderToRedis(ctx, result.IncomingOrder); err != nil {
			log.Printf("Failed to insert incoming order into Redis: %v", err)
			return err
		}
	}
	return nil
}

func fillAgainstLevel(incoming models.Order, level *models.Order, baseAsset, quoteAsset string) (models.Order, models.Order, models.Trade) {
	//calculate fill quantity and price
	fillQuantity := min(incoming.Quantity-incoming.FilledQuantity, level.Quantity-level.FilledQuantity)
	log.Printf("Calculated fill quantity: %f for incoming order id=%s and level order id=%s", fillQuantity, incoming.ID, level.ID)
	fillPrice := level.Price
	incoming.FilledQuantity += fillQuantity
	level.FilledQuantity += fillQuantity
	if level.FilledQuantity == level.Quantity {
		level.Status = string(models.FILLED)
	} else {
		level.Status = string(models.PARTIALLY_FILLED)
	}
	//how can we determine which will be the buy order and which will be the sell order in the trade here?
	var buyOrderID, sellOrderID string
	if incoming.Side == string(models.BUY) {
		//incoming is buy, level is sell
		buyOrderID = incoming.ID
		sellOrderID = level.ID
	} else {
		//incoming is sell, level is buy
		buyOrderID = level.ID
		sellOrderID = incoming.ID
	}
	trade := models.Trade{
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		Price:       fillPrice,
		Quantity:    fillQuantity,
		TradeTime:   time.Now(),
		MarketID:    incoming.MarketID,
		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
	}
	return incoming, *level, trade
}
func fillAgainstLevelForMarketBuy(incoming models.Order, level *models.Order, baseAsset, quoteAsset string) (models.Order, models.Order, models.Trade) {
	//for market buy order the incoming order will not have quantity but it will have price which is the total amount the buyer is willing to spend, we will calculate the fill quantity based on the price and the level price
	fillPrice := level.Price
	fillQuantity := min(incoming.Price/fillPrice, level.Quantity-level.FilledQuantity)
	log.Printf("Calculated fill quantity: %f for incoming market buy order id=%s and level order id=%s", fillQuantity, incoming.ID, level.ID)
	incoming.FilledQuantity += fillQuantity
	level.FilledQuantity += fillQuantity
	if level.FilledQuantity == level.Quantity {
		level.Status = string(models.FILLED)
	} else {
		level.Status = string(models.PARTIALLY_FILLED)
	}
	var buyOrderID, sellOrderID string
	//for market buy order the incoming order is always the buy order and the level is always the sell order
	buyOrderID = incoming.ID
	sellOrderID = level.ID
	incoming.Price -= fillPrice * fillQuantity

	trade := models.Trade{
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		Price:       fillPrice,
		Quantity:    fillQuantity,
		TradeTime:   time.Now(),
		MarketID:    incoming.MarketID,
		BaseAsset:   baseAsset,
		QuoteAsset:  quoteAsset,
	}
	return incoming, *level, trade

}
