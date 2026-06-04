package queries

import "github.com/Blaze5333/cex/internal/models"

func (q *Queries) InsertTrade(trade models.Trade) (models.Trade, error) {
	var insertedTrade models.Trade

	err := q.db.QueryRow(`
        INSERT INTO trades (buy_order_id, sell_order_id, price, quantity, quote_asset, base_asset)
        VALUES ($1, $2, $3, $4, $5, $6)
        RETURNING id, buy_order_id, sell_order_id, price, quantity, created_at, quote_asset, base_asset
    `, trade.BuyOrderID, trade.SellOrderID, trade.Price, trade.Quantity, trade.QuoteAsset, trade.BaseAsset).Scan(
		&insertedTrade.ID,
		&insertedTrade.BuyOrderID,
		&insertedTrade.SellOrderID,
		&insertedTrade.Price,
		&insertedTrade.Quantity,
		&insertedTrade.TradeTime,
		&insertedTrade.QuoteAsset,
		&insertedTrade.BaseAsset,
	)
	if err != nil {
		return insertedTrade, err
	}

	// Only fetch user IDs from both orders in a single query
	err = q.db.QueryRow(`
        SELECT
            bo.id, bo.user_id,
            so.id, so.user_id
        FROM orders bo, orders so
        WHERE bo.id = $1 AND so.id = $2
    `, insertedTrade.BuyOrderID, insertedTrade.SellOrderID).Scan(
		&insertedTrade.BuyOrder.ID,
		&insertedTrade.BuyOrder.UserID,
		&insertedTrade.SellOrder.ID,
		&insertedTrade.SellOrder.UserID,
	)

	return insertedTrade, err
}
