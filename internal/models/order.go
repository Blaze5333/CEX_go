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

type CreateOrderRequest struct {
	MarketID  string  `json:"market_id" binding:"required"`
	OrderType string  `json:"order_type" binding:"required,oneof=limit market"`
	Side      string  `json:"side" binding:"required,oneof=buy sell"`
	Price     float64 `json:"price" binding:"required_if=OrderType limit,gt=0"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
}
