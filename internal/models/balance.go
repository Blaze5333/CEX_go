package models

type Balance struct {
	Asset     string  `json:"asset"`
	Available float64 `json:"available"`
	Locked    float64 `json:"locked"`
}

type Portfolio struct {
	Balances     []Balance `json:"balances"`
	TotalUSD     float64   `json:"total_usd"`
	OpenExposure float64   `json:"open_exposure"`
}

type DepositRequest struct {
	Asset  string  `json:"asset" binding:"required"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

type CreateAssetRequest struct {
	Symbol  string `json:"symbol" binding:"required"`
	Name    string `json:"name" binding:"required"`
	IconURL string `json:"icon_url" binding:"omitempty,url"`
}
