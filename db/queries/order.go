package queries

import (
	"database/sql"
	"log"
	"time"

	"github.com/Blaze5333/cex/internal/models"
)

const orderTag = "[queries/order]"

func (q *Queries) CreateOrder(userId, marketId, orderType, side string, price, quantity float64) (*models.Order, error) {
	log.Printf("%s CreateOrder: userID=%s marketID=%s type=%s side=%s price=%f quantity=%f", orderTag, userId, marketId, orderType, side, price, quantity)
	var id string
	err := q.db.QueryRow(`
		INSERT INTO orders (user_id, market_id, order_type, side, price, quantity, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userId, marketId, orderType, side, price, quantity, "pending").Scan(&id)
	if err != nil {
		log.Printf("%s CreateOrder: failed for userID=%s marketID=%s: %v", orderTag, userId, marketId, err)
		return nil, err
	}
	log.Printf("%s CreateOrder: created order id=%s", orderTag, id)
	return &models.Order{ID: id, UserID: userId, MarketID: marketId, OrderType: orderType, Side: side, Price: price, Quantity: quantity, Status: "pending", FilledQuantity: 0}, nil
}

func (q *Queries) CreateOrderTx(tx *sql.Tx, userId, marketId, orderType, side string, price, quantity float64) (*models.Order, error) {
	log.Printf("%s CreateOrderTx: userID=%s marketID=%s type=%s side=%s price=%f quantity=%f", orderTag, userId, marketId, orderType, side, price, quantity)
	var order models.Order
	err := tx.QueryRow(`
		INSERT INTO orders (user_id, market_id, order_type, side, price, quantity, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, market_id, order_type, side, price, quantity, status, created_at, filled_quantity
	`, userId, marketId, orderType, side, price, quantity, "pending").Scan(
		&order.ID,
		&order.UserID,
		&order.MarketID,
		&order.OrderType,
		&order.Side,
		&order.Price,
		&order.Quantity,
		&order.Status,
		&order.CreatedAt,
		&order.FilledQuantity,
	)
	if err != nil {
		log.Printf("%s CreateOrderTx: failed for userID=%s marketID=%s: %v", orderTag, userId, marketId, err)
		return nil, err
	}
	if order.CreatedAt.IsZero() {
		order.CreatedAt = time.Now()
	}
	log.Printf("%s CreateOrderTx: created order id=%s", orderTag, order.ID)
	return &order, nil
}

func (q *Queries) GetOrderByID(id string) (*models.Order, error) {
	log.Printf("%s GetOrderByID: orderID=%s", orderTag, id)
	var o models.Order
	err := q.db.QueryRow(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at, filled_quantity
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
		&o.Price, &o.Quantity, &o.Status, &o.CreatedAt, &o.FilledQuantity,
	)
	if err != nil {
		log.Printf("%s GetOrderByID: order not found for id=%s: %v", orderTag, id, err)
		return nil, err
	}
	log.Printf("%s GetOrderByID: found order id=%s status=%s", orderTag, o.ID, o.Status)
	return &o, nil
}

func (q *Queries) GetActiveOrdersByUserID(userId string) ([]models.Order, error) {
	//join market_id with markets table to get base and quote asset for each order
	log.Printf("%s GetActiveOrdersByUserID: userID=%s", orderTag, userId)
	rows, err := q.db.Query(`
		SELECT orders.id, orders.user_id, orders.market_id, orders.order_type, orders.side, orders.price, orders.quantity, orders.status, orders.created_at, orders.filled_quantity,markets.base_asset, markets.quote_asset
		FROM orders INNER JOIN markets ON orders.market_id = markets.id
		WHERE orders.user_id = $1 AND (orders.status='open' OR orders.status='partially_filled')
		ORDER BY orders.created_at DESC
	`, userId)
	if err != nil {
		log.Printf("%s GetActiveOrdersByUserID: query failed for userID=%s: %v", orderTag, userId, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt, &o.FilledQuantity, &o.BaseAsset, &o.QuoteAsset,
		); err != nil {
			log.Printf("%s GetActiveOrdersByUserID: failed to scan row for userID=%s: %v", orderTag, userId, err)
			return nil, err
		}
		orders = append(orders, o)
	}
	log.Printf("%s GetActiveOrdersByUserID: returned %d order(s) for userID=%s", orderTag, len(orders), userId)
	return orders, nil
}

func (q *Queries) GetOrdersByMarketID(marketId string) ([]models.Order, error) {
	log.Printf("%s GetOrdersByMarketID: marketID=%s", orderTag, marketId)
	rows, err := q.db.Query(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at, filled_quantity
		FROM orders
		WHERE market_id = $1
	`, marketId)
	if err != nil {
		log.Printf("%s GetOrdersByMarketID: query failed for marketID=%s: %v", orderTag, marketId, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt, &o.FilledQuantity,
		); err != nil {
			log.Printf("%s GetOrdersByMarketID: failed to scan row for marketID=%s: %v", orderTag, marketId, err)
			return nil, err
		}
		orders = append(orders, o)
	}
	log.Printf("%s GetOrdersByMarketID: returned %d order(s) for marketID=%s", orderTag, len(orders), marketId)
	return orders, nil
}
func (q *Queries) GetAllActiveOrders() ([]models.Order, error) {
	log.Printf("%s GetAllActiveOrders", orderTag)
	rows, err := q.db.Query(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at, filled_quantity
		FROM orders
		WHERE status = 'open'
		ORDER BY created_at ASC
	`)
	if err != nil {
		log.Printf("%s GetAllActiveOrders: query failed: %v", orderTag, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt, &o.FilledQuantity,
		); err != nil {
			log.Printf("%s GetAllActiveOrders: failed to scan row: %v", orderTag, err)
			return nil, err
		}
		orders = append(orders, o)
	}
	log.Printf("%s GetAllActiveOrders: returned %d active order(s)", orderTag, len(orders))
	return orders, nil
}
func (q *Queries) UpdateOrderStatus(id, status string) error {
	log.Printf("%s UpdateOrderStatus: orderID=%s status=%s", orderTag, id, status)
	_, err := q.db.Exec(`
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`, status, id)
	if err != nil {
		log.Printf("%s UpdateOrderStatus: failed for orderID=%s status=%s: %v", orderTag, id, status, err)
		return err
	}
	log.Printf("%s UpdateOrderStatus: successfully updated orderID=%s to status=%s", orderTag, id, status)
	return nil
}
func (q *Queries) UpdateOrderStatusTx(tx *sql.Tx, id, status string) (*models.Order, error) {
	var order models.Order
	log.Printf("%s UpdateOrderStatusTx: orderID=%s status=%s", orderTag, id, status)

	err := tx.QueryRow(`
		UPDATE orders
		SET status = $1
		FROM markets
		WHERE orders.id = $2 AND orders.market_id = markets.id
		RETURNING orders.id, orders.user_id, orders.market_id, orders.order_type, orders.side, orders.price, orders.quantity, orders.status, orders.created_at, orders.filled_quantity,markets.base_asset, markets.quote_asset

	`, status, id).Scan(
		&order.ID,
		&order.UserID,
		&order.MarketID,
		&order.OrderType,
		&order.Side,
		&order.Price,
		&order.Quantity,
		&order.Status,
		&order.CreatedAt,
		&order.FilledQuantity,
		&order.BaseAsset,
		&order.QuoteAsset,
	)
	if err != nil {
		log.Printf("%s UpdateOrderStatusTx: failed to update and return order for orderID=%s status=%s: %v", orderTag, id, status, err)
		return nil, err
	}
	log.Printf("%s UpdateOrderStatusTx: successfully updated orderID=%s to status=%s", orderTag, id, status)
	return &order, nil

}

func (q *Queries) DeleteOrder(id string) error {
	log.Printf("%s DeleteOrder: orderID=%s", orderTag, id)
	_, err := q.db.Exec(`
		DELETE FROM orders
		WHERE id = $1
	`, id)
	if err != nil {
		log.Printf("%s DeleteOrder: failed for orderID=%s: %v", orderTag, id, err)
		return err
	}
	log.Printf("%s DeleteOrder: successfully deleted orderID=%s", orderTag, id)
	return nil
}

func (q *Queries) GetOpenOrdersByMarket(marketID, side string) ([]models.Order, error) {
	log.Printf("%s GetOpenOrdersByMarket: marketID=%s side=%s", orderTag, marketID, side)
	sortDir := "ASC"
	if side == "buy" {
		sortDir = "DESC"
	}
	rows, err := q.db.Query(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at, filled_quantity
		FROM orders
		WHERE market_id = $1 AND side = $2 AND status = 'open'
		ORDER BY price `+sortDir+`, created_at ASC
	`, marketID, side)
	if err != nil {
		log.Printf("%s GetOpenOrdersByMarket: query failed for marketID=%s side=%s: %v", orderTag, marketID, side, err)
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt, &o.FilledQuantity,
		); err != nil {
			log.Printf("%s GetOpenOrdersByMarket: failed to scan row for marketID=%s side=%s: %v", orderTag, marketID, side, err)
			return nil, err
		}
		orders = append(orders, o)
	}
	log.Printf("%s GetOpenOrdersByMarket: returned %d order(s) for marketID=%s side=%s", orderTag, len(orders), marketID, side)
	return orders, nil
}

// UpdateOrderStatusAndQuantity updates both status and remaining quantity after a partial or full fill.
func (q *Queries) UpdateOrderStatusAndQuantity(id, status string, filledQty float64) error {
	log.Printf("%s UpdateOrderStatusAndQuantity: orderID=%s status=%s filledQty=%f", orderTag, id, status, filledQty)
	_, err := q.db.Exec(`
		UPDATE orders
		SET status = $1, filled_quantity = $2
		WHERE id = $3
	`, status, filledQty, id)
	if err != nil {
		log.Printf("%s UpdateOrderStatusAndQuantity: failed for orderID=%s: %v", orderTag, id, err)
		return err
	}
	log.Printf("%s UpdateOrderStatusAndQuantity: successfully updated orderID=%s", orderTag, id)
	return nil
}

func (q *Queries) UpdateOrderStatusAndQuantityTx(tx *sql.Tx, id, status string, filledQty float64) error {
	log.Printf("%s UpdateOrderStatusAndQuantityTx: orderID=%s status=%s filledQty=%f", orderTag, id, status, filledQty)
	_, err := tx.Exec(`
		UPDATE orders
		SET status = $1, filled_quantity = $2
		WHERE id = $3
	`, status, filledQty, id)
	if err != nil {
		log.Printf("%s UpdateOrderStatusAndQuantityTx: failed for orderID=%s: %v", orderTag, id, err)
		return err
	}
	return nil
}

// CancelOrder marks an order as cancelled. The AND user_id check ensures a user can only cancel their own orders.
func (q *Queries) CancelOrder(id, userID string) error {
	log.Printf("%s CancelOrder: orderID=%s userID=%s", orderTag, id, userID)
	result, err := q.db.Exec(`
		UPDATE orders
		SET status = 'cancelled'
		WHERE id = $1 AND user_id = $2 AND status = 'open'
	`, id, userID)
	if err != nil {
		log.Printf("%s CancelOrder: exec failed for orderID=%s userID=%s: %v", orderTag, id, userID, err)
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		log.Printf("%s CancelOrder: failed to get rows affected for orderID=%s: %v", orderTag, id, err)
		return err
	}
	if rows == 0 {
		log.Printf("%s CancelOrder: no order cancelled (not found, wrong user, or not open) orderID=%s userID=%s", orderTag, id, userID)
		return sql.ErrNoRows
	}
	log.Printf("%s CancelOrder: successfully cancelled orderID=%s", orderTag, id)
	return nil
}
