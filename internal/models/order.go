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
	BaseAsset      string    `json:"base_asset"`
	QuoteAsset     string    `json:"quote_asset"`
}
type OrderRef struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

// quanity field is required if order type is limit or side is sell, price field is required if order type is limit or side is buy
type CreateOrderRequest struct {
	MarketID   string  `json:"market_id" binding:"required"`
	OrderType  string  `json:"order_type" binding:"required,oneof=limit market"`
	Side       string  `json:"side" binding:"required,oneof=buy sell"`
	Price      float64 `json:"price" binding:"required_if=OrderType limit"`
	Quantity   float64 `json:"quantity" binding:"required_if=OrderType limit"`
	BaseAsset  string  `json:"base_asset" binding:"required"`
	QuoteAsset string  `json:"quote_asset" binding:"required"`
	Amount     float64 `json:"amount" ` // New field for market orders to specify total amount to spend (for buy) or receive (for sell)
}

type CancelOrderRequest struct {
	OrderID string `json:"order_id" binding:"required"`
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

type UserTrade struct {
	ID          string    `json:"id"`
	MarketID    string    `json:"market_id"`
	Side        string    `json:"side"`
	Price       float64   `json:"price"`
	Quantity    float64   `json:"quantity"`
	BaseAsset   string    `json:"base_asset"`
	QuoteAsset  string    `json:"quote_asset"`
	CreatedAt   time.Time `json:"created_at"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
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
	CLOSED           OrderStatus = "closed" // For market orders that cannot be filled
)

type OrderType string

const (
	LIMIT  OrderType = "limit"
	MARKET OrderType = "market"
)
