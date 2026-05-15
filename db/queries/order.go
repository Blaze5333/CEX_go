package queries

import (
	"database/sql"

	"github.com/Blaze5333/cex/internal/models"
)

func (q *Queries) CreateOrder(userId, marketId, orderType, side string, price, quantity float64) (string, error) {
	var id string
	err := q.db.QueryRow(`
		INSERT INTO orders (user_id, market_id, order_type, side, price, quantity, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, userId, marketId, orderType, side, price, quantity, "open").Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (q *Queries) GetOrderByID(id string) (*models.Order, error) {
	var o models.Order
	err := q.db.QueryRow(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at
		FROM orders
		WHERE id = $1
	`, id).Scan(
		&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
		&o.Price, &o.Quantity, &o.Status, &o.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &o, nil
}
func (q *Queries) GetOrdersByUserID(userId string) ([]models.Order, error) {
	rows, err := q.db.Query(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at
		FROM orders
		WHERE user_id = $1
	`, userId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

func (q *Queries) GetOrdersByMarketID(marketId string) ([]models.Order, error) {
	rows, err := q.db.Query(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at
		FROM orders
		WHERE market_id = $1
	`, marketId)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}
func (q *Queries) UpdateOrderStatus(id, status string) error {
	_, err := q.db.Exec(`
		UPDATE orders
		SET status = $1
		WHERE id = $2
	`, status, id)
	return err
}
func (q *Queries) DeleteOrder(id string) error {
	_, err := q.db.Exec(`
		DELETE FROM orders
		WHERE id = $1
	`, id)
	return err
}

func (q *Queries) GetOpenOrdersByMarket(marketID, side string) ([]models.Order, error) {
	sortDir := "ASC"
	if side == "buy" {
		sortDir = "DESC"
	}
	rows, err := q.db.Query(`
		SELECT id, user_id, market_id, order_type, side, price, quantity, status, created_at
		FROM orders
		WHERE market_id = $1 AND side = $2 AND status = 'open'
		ORDER BY price `+sortDir+`, created_at ASC
	`, marketID, side)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orders []models.Order
	for rows.Next() {
		var o models.Order
		if err := rows.Scan(
			&o.ID, &o.UserID, &o.MarketID, &o.OrderType, &o.Side,
			&o.Price, &o.Quantity, &o.Status, &o.CreatedAt,
		); err != nil {
			return nil, err
		}
		orders = append(orders, o)
	}
	return orders, nil
}

// UpdateOrderStatusAndQuantity updates both status and remaining quantity after a partial or full fill.
func (q *Queries) UpdateOrderStatusAndQuantity(id, status string, remainingQty float64) error {
	_, err := q.db.Exec(`
		UPDATE orders
		SET status = $1, quantity = $2
		WHERE id = $3
	`, status, remainingQty, id)
	return err
}

// CancelOrder marks an order as cancelled. The AND user_id check ensures a user can only cancel their own orders.
func (q *Queries) CancelOrder(id, userID string) error {
	result, err := q.db.Exec(`
		UPDATE orders
		SET status = 'cancelled'
		WHERE id = $1 AND user_id = $2 AND status = 'open'
	`, id, userID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows // order not found, not owned by user, or already closed
	}
	return nil
}
