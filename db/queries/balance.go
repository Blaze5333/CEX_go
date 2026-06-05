package queries

import (
	"database/sql"
	"log"

	"github.com/Blaze5333/cex/internal/models"
)

const balanceTag = "[queries/balance]"

func (q *Queries) GetDB() *sql.DB {
	return q.db
}

func (q *Queries) LockBalance(userID, asset string, amount float64) error {
	log.Printf("%s LockBalance: userID=%s asset=%s amount=%f", balanceTag, userID, asset, amount)
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("%s LockBalance: failed to begin transaction: %v", balanceTag, err)
		return err
	}
	defer tx.Rollback()

	var available float64
	err = tx.QueryRow(`
	   SELECT available
	   FROM balances
	   WHERE user_id = $1 AND asset = $2
	   FOR UPDATE
	`, userID, asset).Scan(&available)
	if err != nil {
		log.Printf("%s LockBalance: failed to query available balance for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	if available < amount {
		log.Printf("%s LockBalance: insufficient balance for userID=%s asset=%s available=%f requested=%f", balanceTag, userID, asset, available, amount)
		return sql.ErrNoRows
	}

	_, err = tx.Exec(`
	   UPDATE balances
	   SET available = available - $1, locked = locked + $1
	   WHERE user_id = $2 AND asset = $3
	`, amount, userID, asset)
	if err != nil {
		log.Printf("%s LockBalance: failed to update balance for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Printf("%s LockBalance: failed to commit transaction for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	log.Printf("%s LockBalance: successfully locked %f of %s for userID=%s", balanceTag, amount, asset, userID)
	return nil
}

func (q *Queries) UnlockBalance(userID, asset string, amount float64) error {
	log.Printf("%s UnlockBalance: userID=%s asset=%s amount=%f", balanceTag, userID, asset, amount)
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("%s UnlockBalance: failed to begin transaction: %v", balanceTag, err)
		return err
	}
	defer tx.Rollback()

	var locked float64
	err = tx.QueryRow("SELECT locked FROM balances WHERE user_id=$1 AND asset=$2 FOR UPDATE", userID, asset).Scan(&locked)
	if err != nil {
		log.Printf("%s UnlockBalance: failed to query locked balance for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	if locked < amount {
		log.Printf("%s UnlockBalance: insufficient locked balance for userID=%s asset=%s locked=%f requested=%f", balanceTag, userID, asset, locked, amount)
		return sql.ErrNoRows
	}

	_, err = tx.Exec(`
		UPDATE balances
		SET locked = locked - $1, available = available + $1
		WHERE user_id = $2 AND asset = $3
	`, amount, userID, asset)
	if err != nil {
		log.Printf("%s UnlockBalance: failed to update balance for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Printf("%s UnlockBalance: failed to commit transaction for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	log.Printf("%s UnlockBalance: successfully unlocked %f of %s for userID=%s", balanceTag, amount, asset, userID)
	return nil
}

func (q *Queries) TransferFromLocked(fromUserID, toUserID, asset string, amount float64) error {
	log.Printf("%s TransferFromLocked: fromUserID=%s toUserID=%s asset=%s amount=%f", balanceTag, fromUserID, toUserID, asset, amount)
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("%s TransferFromLocked: failed to begin transaction: %v", balanceTag, err)
		return err
	}
	defer tx.Rollback()

	var locked float64
	err = tx.QueryRow(`
		SELECT locked
		FROM balances
		WHERE user_id = $1 AND asset = $2
		FOR UPDATE
	`, fromUserID, asset).Scan(&locked)
	if err != nil {
		log.Printf("%s TransferFromLocked: failed to query locked balance for fromUserID=%s asset=%s: %v", balanceTag, fromUserID, asset, err)
		return err
	}
	if locked < amount {
		log.Printf("%s TransferFromLocked: insufficient locked balance for fromUserID=%s asset=%s locked=%f requested=%f", balanceTag, fromUserID, asset, locked, amount)
		return sql.ErrNoRows
	}

	_, err = tx.Exec(`
		UPDATE balances
		SET locked = locked - $1
		WHERE user_id = $2 AND asset = $3
	`, amount, fromUserID, asset)
	if err != nil {
		log.Printf("%s TransferFromLocked: failed to deduct locked balance for fromUserID=%s asset=%s: %v", balanceTag, fromUserID, asset, err)
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (user_id, asset)
		DO UPDATE SET available = balances.available + $3
	`, toUserID, asset, amount)
	if err != nil {
		log.Printf("%s TransferFromLocked: failed to credit balance for toUserID=%s asset=%s: %v", balanceTag, toUserID, asset, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Printf("%s TransferFromLocked: failed to commit transaction: %v", balanceTag, err)
		return err
	}
	log.Printf("%s TransferFromLocked: successfully transferred %f of %s from userID=%s to userID=%s", balanceTag, amount, asset, fromUserID, toUserID)
	return nil
}

func (q *Queries) CreditBalance(userID, asset string, amount float64) error {
	log.Printf("%s CreditBalance: userID=%s asset=%s amount=%f", balanceTag, userID, asset, amount)
	_, err := q.db.Exec(`
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, $2, $3, 0)
		ON CONFLICT (user_id, asset)
		DO UPDATE SET available = balances.available + $3
	`, userID, asset, amount)
	if err != nil {
		log.Printf("%s CreditBalance: failed for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	log.Printf("%s CreditBalance: successfully credited %f of %s for userID=%s", balanceTag, amount, asset, userID)
	return nil
}
func (q *Queries) CreditUSD(userID string, amount float64) error {
	log.Printf("%s CreditUSD: userID=%s amount=%f", balanceTag, userID, amount)
	//user table already has usd balance so we will just update that
	_, err := q.db.Exec(`
		UPDATE users
		SET USD_balance = USD_balance + $1
		WHERE id = $2
	`, amount, userID)
	if err != nil {
		log.Printf("%s CreditUSD: failed for userID=%s amount=%f: %v", balanceTag, userID, amount, err)
		return err
	}
	log.Printf("%s CreditUSD: successfully credited %f USD for userID=%s", balanceTag, amount, userID)
	return nil
}
func (q *Queries) DebitUSD(userID string, amount float64) error {
	log.Printf("%s DebitUSD: userID=%s amount=%f", balanceTag, userID, amount)
	//here also we need to lock the row when updating
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("%s DebitUSD: failed to begin transaction: %v", balanceTag, err)
		return err
	}
	defer tx.Rollback()
	var usdBalance float64
	err = tx.QueryRow(`
		SELECT locked_balance
		FROM users
		WHERE id = $1
		FOR UPDATE
	`, userID).Scan(&usdBalance)
	if err != nil {
		log.Printf("%s DebitUSD: failed to query USD balance for userID=%s: %v", balanceTag, userID, err)
		return err
	}
	if usdBalance < amount {
		log.Printf("%s DebitUSD: insufficient USD balance for userID=%s available=%f requested=%f", balanceTag, userID, usdBalance, amount)
		return sql.ErrNoRows
	}
	_, err = tx.Exec(`
		UPDATE users
		SET locked_balance = locked_balance - $1
		WHERE id = $2
	`, amount, userID)
	if err != nil {
		log.Printf("%s DebitUSD: failed to update USD balance for userID=%s: %v", balanceTag, userID, err)
		return err
	}
	if err = tx.Commit(); err != nil {
		log.Printf("%s DebitUSD: failed to commit transaction for userID=%s: %v", balanceTag, userID, err)
		return err
	}
	log.Printf("%s DebitUSD: successfully debited %f USD for userID=%s", balanceTag, amount, userID)
	return nil
}
func (q *Queries) DebitBalance(userID, asset string, amount float64) error {
	log.Printf("%s DebitBalance: userID=%s asset=%s amount=%f", balanceTag, userID, asset, amount)
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("%s DebitBalance: failed to begin transaction: %v", balanceTag, err)
		return err
	}
	defer tx.Rollback()
	//balance will always be debited from locked balance so we will check locked balance instead of available balance
	var locked float64
	err = tx.QueryRow(`
		SELECT locked
		FROM balances
		WHERE user_id = $1 AND asset = $2
		FOR UPDATE
	`, userID, asset).Scan(&locked)
	if err != nil {
		log.Printf("%s DebitBalance: failed to query locked balance for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	if locked < amount {
		log.Printf("%s DebitBalance: insufficient locked balance for userID=%s asset=%s locked=%f requested=%f", balanceTag, userID, asset, locked, amount)
		return sql.ErrNoRows
	}

	_, err = tx.Exec(`
		UPDATE balances
		SET locked = locked - $1
		WHERE user_id = $2 AND asset = $3
	`, amount, userID, asset)
	if err != nil {
		log.Printf("%s DebitBalance: failed to update balance for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}

	if err = tx.Commit(); err != nil {
		log.Printf("%s DebitBalance: failed to commit transaction for userID=%s asset=%s: %v", balanceTag, userID, asset, err)
		return err
	}
	log.Printf("%s DebitBalance: successfully debited %f of %s for userID=%s", balanceTag, amount, asset, userID)
	return nil
}

func (q *Queries) GetBalances(userID string) ([]models.Balance, error) {
	log.Printf("%s GetBalances: fetching balances for userID=%s", balanceTag, userID)
	rows, err := q.db.Query(`
		SELECT asset, available, locked
		FROM balances
		WHERE user_id = $1
	`, userID)
	if err != nil {
		log.Printf("%s GetBalances: query failed for userID=%s: %v", balanceTag, userID, err)
		return nil, err
	}
	defer rows.Close()

	var balances []models.Balance
	for rows.Next() {
		var b models.Balance
		if err := rows.Scan(&b.Asset, &b.Available, &b.Locked); err != nil {
			log.Printf("%s GetBalances: failed to scan row for userID=%s: %v", balanceTag, userID, err)
			return nil, err
		}
		balances = append(balances, b)
	}
	log.Printf("%s GetBalances: returned %d balance(s) for userID=%s", balanceTag, len(balances), userID)
	return balances, nil
}

func (q *Queries) InitializeBalance(userID string) error {
	log.Printf("%s InitializeBalance: initializing USD balance for userID=%s", balanceTag, userID)
	_, err := q.db.Exec(`
		INSERT INTO balances (user_id, asset, available, locked)
		VALUES ($1, 'USD', 500, 0)
	`, userID)
	if err != nil {
		log.Printf("%s InitializeBalance: failed for userID=%s: %v", balanceTag, userID, err)
		return err
	}
	log.Printf("%s InitializeBalance: successfully initialized balance for userID=%s", balanceTag, userID)
	return nil
}

// when user buys BTC or deposits usd so we need smart contract to update the balance of the user.\
func (q *Queries) CreateAsset(asset models.CreateAssetRequest) error {
	log.Printf("%s CreateAsset: creating asset with symbol=%s name=%s", balanceTag, asset.Symbol, asset.Name)
	_, err := q.db.Exec(`
		INSERT INTO assets (symbol, name, icon_url,is_active)
		VALUES ($1, $2, $3, TRUE)
	`, asset.Symbol, asset.Name, asset.IconURL)
	if err != nil {
		log.Printf("%s CreateAsset: failed to create asset with symbol=%s: %v", balanceTag, asset.Symbol, err)
		return err
	}
	log.Printf("%s CreateAsset: successfully created asset with symbol=%s", balanceTag, asset.Symbol)
	return nil
}
func (q *Queries) InactivateAsset(symbol string) error {
	log.Printf("%s InactivateAsset: inactivating asset with symbol=%s", balanceTag, symbol)
	_, err := q.db.Exec(`
		UPDATE assets
		SET is_active = FALSE
		WHERE symbol = $1
	`, symbol)
	if err != nil {
		log.Printf("%s InactivateAsset: failed to inactivate asset with symbol=%s: %v", balanceTag, symbol, err)
		return err
	}
	log.Printf("%s InactivateAsset: successfully inactivated asset with symbol=%s", balanceTag, symbol)
	return nil
}
func (q *Queries) GetActiveAssets() ([]string, error) {
	log.Printf("%s GetActiveAssets: fetching active assets", balanceTag)
	rows, err := q.db.Query(`
		SELECT symbol
		FROM assets
		WHERE is_active = TRUE
	`)
	if err != nil {
		log.Printf("%s GetActiveAssets: query failed: %v", balanceTag, err)
		return nil, err
	}
	defer rows.Close()

	var assets []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			log.Printf("%s GetActiveAssets: failed to scan row: %v", balanceTag, err)
			return nil, err
		}
		assets = append(assets, symbol)
	}
	log.Printf("%s GetActiveAssets: returned %d active asset(s)", balanceTag, len(assets))
	return assets, nil

}
