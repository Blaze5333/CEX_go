package models

import "time"

type Market struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	BaseAsset    string    `json:"base_asset"`
	QuoteAsset   string    `json:"quote_asset"`
	CreatedAt    time.Time `json:"created_at"`
	IsActive     bool      `json:"is_active"`
	MinOrderSize float64   `json:"min_order_size"`
	MaxOrderSize float64   `json:"max_order_size"`
	TakerFee     float64   `json:"taker_fee"`
	MakerFee     float64   `json:"maker_fee"`
	CurrentPrice float64   `json:"current_price"`
}

type CreateMarketRequest struct {
	Name         string  `json:"name" binding:"required"`
	BaseAsset    string  `json:"base_asset" binding:"required"`
	QuoteAsset   string  `json:"quote_asset" binding:"required"`
	MinOrderSize float64 `json:"min_order_size" binding:"required,gt=0"`
	MaxOrderSize float64 `json:"max_order_size" binding:"required,gt=0"`
	TakerFee     float64 `json:"taker_fee" binding:"gte=0"`
	MakerFee     float64 `json:"maker_fee" binding:"gte=0"`
}
