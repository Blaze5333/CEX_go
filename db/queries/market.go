package queries

import "github.com/Blaze5333/cex/internal/models"

func (q *Queries) GetAllMarkets() ([]models.Market, error) {
	rows, err := q.db.Query(`
		SELECT id, name, base_asset, quote_asset, created_at, is_active,
		       min_order_size, max_order_size, taker_fee, maker_fee, current_price
		FROM markets
		WHERE is_active = true
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var markets []models.Market
	for rows.Next() {
		var m models.Market
		if err := rows.Scan(
			&m.ID, &m.Name, &m.BaseAsset, &m.QuoteAsset, &m.CreatedAt, &m.IsActive,
			&m.MinOrderSize, &m.MaxOrderSize, &m.TakerFee, &m.MakerFee, &m.CurrentPrice,
		); err != nil {
			return nil, err
		}
		markets = append(markets, m)
	}
	return markets, nil
}

func (q *Queries) GetMarketByID(id string) (*models.Market, error) {
	var m models.Market
	err := q.db.QueryRow(`
		SELECT id, name, base_asset, quote_asset, created_at, is_active,
		       min_order_size, max_order_size, taker_fee, maker_fee, current_price
		FROM markets
		WHERE id = $1 AND is_active = true
	`, id).Scan(
		&m.ID, &m.Name, &m.BaseAsset, &m.QuoteAsset, &m.CreatedAt, &m.IsActive,
		&m.MinOrderSize, &m.MaxOrderSize, &m.TakerFee, &m.MakerFee, &m.CurrentPrice,
	)
	if err != nil {
		return nil, err
	}
	return &m, nil
}

func (q *Queries) CreateMarket(name, baseAsset, quoteAsset string, minOrderSize, maxOrderSize, takerFee, makerFee float64) (string, error) {
	var id string
	err := q.db.QueryRow(`
		INSERT INTO markets (name, base_asset, quote_asset, min_order_size, max_order_size, taker_fee, maker_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, name, baseAsset, quoteAsset, minOrderSize, maxOrderSize, takerFee, makerFee).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}
