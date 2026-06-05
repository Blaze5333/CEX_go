package matching

import (
	"context"
	"log"
	"time"

	"sync"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/models"
	"golang.org/x/sync/errgroup"
)

type OrderResult struct {
	IncomingOrder models.Order
	UpdatedOrders []models.Order
	Trades        []models.Trade
}
type MathcingEngine struct {
	rdb *db.RedisConfig
	db  queries.Queries
}

func (me *MathcingEngine) MatchOrders(order models.Order) OrderResult {
	result := OrderResult{IncomingOrder: order}
	//matching logic here, update result.UpdatedOrders and result.Trades as needed
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for order.FilledQuantity < order.Quantity {
		var bestLevel *models.Order
		var err error
		if order.Side == string(models.BUY) {
			//match with best sell orders
			bestLevel, err = me.rdb.BestAskFromRedis(ctx, order.MarketID)
			if err != nil {
				break
			}
			if bestLevel == nil || bestLevel.Price > order.Price {
				break //no more matches
			}
		} else {
			//match with best buy orders
			bestLevel, err = me.rdb.BestBidFromRedis(ctx, order.MarketID)
			if err != nil {
				break
			}
			if bestLevel == nil || bestLevel.Price < order.Price {
				break //no more matches
			}
		}
		incoming, againstOrder, trade := fillAgainstLevel(order, bestLevel)
		result.UpdatedOrders = append(result.UpdatedOrders, incoming, againstOrder)
		result.Trades = append(result.Trades, trade)
		order = incoming
		result.IncomingOrder = incoming
		//also need to update the order book in Redis and do the final call to DB to persist the trade and updated orders after the loop ends
		err = me.rdb.UpdateOrderInRedis(ctx, againstOrder)
		if err != nil {
			break
		}
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

func (me *MathcingEngine) ApplyOrderResultToDB(ctx context.Context, result OrderResult) error {
	//so the operations here are
	//1. insert all trades in result.Trades
	//2. update all orders in result.UpdatedOrders
	//3. update the incoming order in result.IncomingOrder
	//4.update balance of the users involved in the trades
	//we will use a wait group to wait for all operations to complete before returning
	var wg sync.WaitGroup
	var err error
	wg.Add(4)
	var trades []models.Trade
	tradesDone := make(chan models.Trade, len(result.Trades))
	var errorGroup errgroup.Group
	errorGroup.Go(func() error {
		defer close(tradesDone)
		defer wg.Done()
		for index, trade := range result.Trades {
			tradefromDb, err := me.db.InsertTrade(trade)
			if err != nil {
				log.Printf("Index : %v Failed to insert trade into DB: %v", index, err)
				return err
			}
			trades = append(trades, tradefromDb)
			tradesDone <- tradefromDb
		}
		return nil
	})
	errorGroup.Go(func() error {
		defer wg.Done()
		for _, order := range result.UpdatedOrders {
			err = me.db.UpdateOrderStatusAndQuantity(order.ID, order.Status, order.FilledQuantity)
			if err != nil {
				log.Printf("Index : %v Failed to update order in DB: %v", order.ID, err)
				return err
			}
		}
		return nil
	})
	errorGroup.Go(func() error {
		defer wg.Done()
		err = me.db.UpdateOrderStatusAndQuantity(result.IncomingOrder.ID, result.IncomingOrder.Status, result.IncomingOrder.FilledQuantity)
		if err != nil {
			log.Printf("Index : %v Failed to update incoming order in DB: %v", result.IncomingOrder.ID, err)
			return err
		}
		return nil
	})

	errorGroup.Go(func() error {
		defer wg.Done()
		var mu sync.Mutex
		var innerGroup errgroup.Group

		var balanceToCredit, balanceToDebit float64
		for trade := range tradesDone {
			innerGroup.Go(func() error {
				if result.IncomingOrder.Side == string(models.BUY) {

					if err := me.db.DebitBalance(trade.SellOrder.UserID, trade.QuoteAsset, trade.Quantity); err != nil {
						return err
					}
					if err := me.db.CreditUSD(trade.SellOrder.UserID, trade.Price*trade.Quantity); err != nil {
						return err
					}
					mu.Lock()
					balanceToCredit += trade.Quantity
					balanceToDebit += trade.Price * trade.Quantity
					mu.Unlock()

				} else {
					if err := me.db.CreditBalance(trade.BuyOrder.UserID, trade.QuoteAsset, trade.Quantity); err != nil {
						return err
					}
					if err := me.db.DebitUSD(trade.BuyOrder.UserID, trade.Price*trade.Quantity); err != nil {
						return err
					}
					mu.Lock()
					balanceToCredit += trade.Price * trade.Quantity
					balanceToDebit += trade.Quantity
					mu.Unlock()
				}
				return nil
			})

		}
		if err := innerGroup.Wait(); err != nil {
			log.Printf("Failed to update balances for trades: %v", err)
			return err
		}
		if result.IncomingOrder.Side == string(models.BUY) {
			if err := me.db.CreditBalance(result.IncomingOrder.UserID, result.IncomingOrder.MarketID, balanceToCredit); err != nil {
				log.Printf("Failed to credit balance for user %s: %v", result.IncomingOrder.UserID, err)
				return err
			}
			if err := me.db.DebitUSD(result.IncomingOrder.UserID, balanceToDebit); err != nil {
				log.Printf("Failed to debit USD for user %s: %v", result.IncomingOrder.UserID, err)
				return err
			}
		} else {
			if err := me.db.CreditUSD(result.IncomingOrder.UserID, balanceToCredit); err != nil {
				log.Printf("Failed to credit USD for user %s: %v", result.IncomingOrder.UserID, err)
				return err
			}
			if err := me.db.DebitBalance(result.IncomingOrder.UserID, result.IncomingOrder.MarketID, balanceToDebit); err != nil {
				log.Printf("Failed to debit balance for user %s: %v", result.IncomingOrder.UserID, err)
				return err
			}
		}
		return nil
	})
	wg.Wait()
	//now we neeed to indert the incoming order in redis if it is not fully filled
	if result.IncomingOrder.Status != string(models.FILLED) {
		err = me.rdb.InserOrderToRedis(ctx, result.IncomingOrder)
		if err != nil {
			log.Printf("Failed to insert incoming order into Redis: %v", err)
			return err
		}
	}
	if err := errorGroup.Wait(); err != nil {
		log.Printf("Error occurred while applying order result to DB: %v", err)
		return err
	}
	return err
}

func fillAgainstLevel(incoming models.Order, level *models.Order) (models.Order, models.Order, models.Trade) {
	//calculate fill quantity and price
	fillQuantity := min(incoming.Quantity-incoming.FilledQuantity, level.Quantity-level.FilledQuantity)
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
	}
	return incoming, *level, trade
}
