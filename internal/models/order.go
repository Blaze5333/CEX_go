package models

import "time"

type Order struct {
	ID             string    `json:"id"`
	UserID         string    `json:"user_id"`
	MarketID       string    `json:"market_id"`
	OrderType      string    `json:"order_type"` // "limit" or "market"
	Side           string    `json:"side"`       // "buy" or "sell"
	Price          float64   `json:"price"`
	Quantity       float64   `json:"quantity"`
	Status         string    `json:"status"` // "open", "filled", "partially_filled", "cancelled"
	CreatedAt      time.Time `json:"created_at"`
	FilledQuantity float64   `json:"filled_quantity"` // New field to track filled quantity
}
type OrderRef struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}
type CreateOrderRequest struct {
	MarketID  string  `json:"market_id" binding:"required"`
	OrderType string  `json:"order_type" binding:"required,oneof=limit market"`
	Side      string  `json:"side" binding:"required,oneof=buy sell"`
	Price     float64 `json:"price" binding:"required_if=OrderType limit,gt=0"`
	Quantity  float64 `json:"quantity" binding:"required,gt=0"`
}

type Trade struct {
	ID          string    `json:"id"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	MarketID    string    `json:"market_id"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	TradeTime   time.Time `json:"trade_time"`
	BuyOrder    Order     `json:"buy_order"`
	SellOrder   Order     `json:"sell_order"`
	BaseAsset   string    `json:"base_asset"`
	QuoteAsset  string    `json:"quote_asset"`
}

type OrderSide string

const (
	BUY  OrderSide = "buy"
	SELL OrderSide = "sell"
)

type OrderStatus string

const (
	OPEN             OrderStatus = "open"
	FILLED           OrderStatus = "filled"
	PARTIALLY_FILLED OrderStatus = "partially_filled"
	CANCELLED        OrderStatus = "cancelled"
)
