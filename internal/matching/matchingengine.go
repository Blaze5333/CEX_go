package matching

import (
	"context"
	"time"

	"github.com/Blaze5333/cex/db/queries"
	"github.com/Blaze5333/cex/internal/db"
	"github.com/Blaze5333/cex/internal/models"
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
		if order.Side == string(models.BUY) {
			//match with best sell orders
			bestSellOrder, error := me.rdb.BestAskFromRedis(ctx, order.MarketID)
			if error != nil {
				break
			}
			if bestSellOrder == nil || bestSellOrder.Price > order.Price {
				break //no more matches
			}
		} else {
			//match with best buy orders
			bestBuyOrder, error := me.rdb.BestBidFromRedis(ctx, order.MarketID)
			if error != nil {
				break
			}
			if bestBuyOrder == nil || bestBuyOrder.Price < order.Price {
				break //no more matches
			}
		}

	}
	return result
}
