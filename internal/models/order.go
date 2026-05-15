package models

import "time"

type Order struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	MarketID  string    `json:"market_id"`
	OrderType string    `json:"order_type"` // "limit" or "market"
	Side      string    `json:"side"`       // "buy" or "sell"
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Status    string    `json:"status"` // "open", "filled", "partially_filled", "cancelled"
	CreatedAt time.Time `json:"created_at"`
}
