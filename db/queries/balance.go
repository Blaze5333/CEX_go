package queries

import (
	"database/sql"

	"github.com/Blaze5333/cex/internal/models"
)

func (q *Queries) LockBalance(userID, asset string, amount float64) error {
	result, err := q.db.Exec(`
		UPDATE balances
		SET available = available - $1, locked = locked + $1
		WHERE user_id = $2 AND asset = $3 AND available >= $1
	`, amount, userID, asset)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows // insufficient balance
	}
	return nil
}

func (q *Queries) UnlockBalance(userID, asset string, amount float64) error {
	_, err := q.db.Exec(`
		UPDATE balances
		SET locked = locked - $1, available = available + $1
		WHERE user_id = $2 AND asset = $3
	`, amount, userID, asset)
	return err
}

func (q *Queries) TransferFromLocked(fromUserID, toUserID, asset string, amount float64) error {
	tx, err := q.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`
		UPDATE balances
		SET locked = locked - $1
		WHERE user_id = $2 AND asset = $3
	`, amount, fromUserID, asset)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (user_id, asset)
		DO UPDATE SET available = balances.available + $3
	`, toUserID, asset, amount)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (q *Queries) CreditBalance(userID, asset string, amount float64) error {
	_, err := q.db.Exec(`
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (user_id, asset)
		DO UPDATE SET available = balances.available + $3
	`, userID, asset, amount)
	return err
}

func (q *Queries) GetBalances(userID string) ([]models.Balance, error) {
	rows, err := q.db.Query(`
		SELECT asset, available, locked
		FROM balances
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var balances []models.Balance
	for rows.Next() {
		var b models.Balance
		if err := rows.Scan(&b.Asset, &b.Available, &b.Locked); err != nil {
			return nil, err
		}
		balances = append(balances, b)
	}
	return balances, nil
}
func (q *Queries) InitializeBalance(userID string) error {
	_, err := q.db.Exec(`
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, 'USD', 500, 0)
	`,
		userID,
	)
	return err
}

//when user buys BTC or deposits usd so we need smart contract to update the balance of the user.\
