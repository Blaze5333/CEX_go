package queries

import (
	"log"

	"github.com/Blaze5333/cex/internal/models"
)

const marketTag = "[queries/market]"

func (q *Queries) GetAllMarkets() ([]models.Market, error) {
	log.Printf("%s GetAllMarkets: fetching all active markets", marketTag)
	rows, err := q.db.Query(`
		SELECT id, name, base_asset, quote_asset, created_at, is_active,
		       min_order_size, max_order_size, taker_fee, maker_fee, current_price
		FROM markets
		WHERE is_active = true
	`)
	if err != nil {
		log.Printf("%s GetAllMarkets: query failed: %v", marketTag, err)
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
			log.Printf("%s GetAllMarkets: failed to scan row: %v", marketTag, err)
			return nil, err
		}
		markets = append(markets, m)
	}
	log.Printf("%s GetAllMarkets: returned %d market(s)", marketTag, len(markets))
	return markets, nil
}

func (q *Queries) GetMarketByID(id string) (*models.Market, error) {
	log.Printf("%s GetMarketByID: marketID=%s", marketTag, id)
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
		log.Printf("%s GetMarketByID: market not found for id=%s: %v", marketTag, id, err)
		return nil, err
	}
	log.Printf("%s GetMarketByID: found market name=%s id=%s", marketTag, m.Name, m.ID)
	return &m, nil
}

func (q *Queries) CreateMarket(name, baseAsset, quoteAsset string, minOrderSize, maxOrderSize, takerFee, makerFee float64) (string, error) {
	log.Printf("%s CreateMarket: name=%s baseAsset=%s quoteAsset=%s", marketTag, name, baseAsset, quoteAsset)
	var id string
	err := q.db.QueryRow(`
		INSERT INTO markets (name, base_asset, quote_asset, min_order_size, max_order_size, taker_fee, maker_fee)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
	`, name, baseAsset, quoteAsset, minOrderSize, maxOrderSize, takerFee, makerFee).Scan(&id)
	if err != nil {
		log.Printf("%s CreateMarket: failed to create market name=%s: %v", marketTag, name, err)
		return "", err
	}
	log.Printf("%s CreateMarket: successfully created market id=%s name=%s", marketTag, id, name)
	return id, nil
}
