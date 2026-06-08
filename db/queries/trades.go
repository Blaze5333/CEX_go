package queries

import (
	"database/sql"

	"github.com/Blaze5333/cex/internal/models"
)

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

func (q *Queries) InsertTradeTx(tx *sql.Tx, trade models.Trade) (models.Trade, error) {
	var insertedTrade models.Trade

	err := tx.QueryRow(`
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

	err = tx.QueryRow(`
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

func (q *Queries) GetTradesByUserID(userID string) ([]models.UserTrade, error) {
	rows, err := q.db.Query(`
		SELECT
			t.id,
			bo.market_id,
			CASE WHEN bo.user_id = $1 THEN 'buy' ELSE 'sell' END AS side,
			t.price,
			t.quantity,
			t.base_asset,
			t.quote_asset,
			t.created_at,
			t.buy_order_id,
			t.sell_order_id
		FROM trades t
		JOIN orders bo ON bo.id = t.buy_order_id
		JOIN orders so ON so.id = t.sell_order_id
		WHERE bo.user_id = $1 OR so.user_id = $1
		ORDER BY t.created_at DESC
		LIMIT 100
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	trades := []models.UserTrade{}
	for rows.Next() {
		var trade models.UserTrade
		if err := rows.Scan(
			&trade.ID,
			&trade.MarketID,
			&trade.Side,
			&trade.Price,
			&trade.Quantity,
			&trade.BaseAsset,
			&trade.QuoteAsset,
			&trade.CreatedAt,
			&trade.BuyOrderID,
			&trade.SellOrderID,
		); err != nil {
			return nil, err
		}
		trades = append(trades, trade)
	}
	return trades, rows.Err()
}
