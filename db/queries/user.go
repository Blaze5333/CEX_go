package queries

import (
	"database/sql"
	"github.com/Blaze5333/cex/internal/models"
	"log"
)

const userTag = "[queries/user]"

type Queries struct {
	db *sql.DB
}

func New(db *sql.DB) *Queries {
	return &Queries{db: db}
}

func (q *Queries) CreateUser(email, passwordHash string) (string, error) {
	log.Printf("%s CreateUser: creating user with email=%s", userTag, email)
	var id string
	err := q.db.QueryRow(`
		INSERT INTO users (email, password_hash) 
		VALUES ($1, $2) 
		RETURNING id
	`, email, passwordHash).Scan(&id)
	if err != nil {
		log.Printf("%s CreateUser: failed to create user with email=%s: %v", userTag, email, err)
		return "", err
	}
	log.Printf("%s CreateUser: successfully created user id=%s email=%s", userTag, id, email)
	return id, nil
}

func (q *Queries) GetUserByEmail(email string) (string, string, error) {
	log.Printf("%s GetUserByEmail: looking up user with email=%s", userTag, email)
	var id, passwordHash string
	err := q.db.QueryRow(`
		SELECT id, password_hash 
		FROM users 
		WHERE email = $1
	`, email).Scan(&id, &passwordHash)
	if err != nil {
		log.Printf("%s GetUserByEmail: user not found for email=%s: %v", userTag, email, err)
		return "", "", err
	}
	log.Printf("%s GetUserByEmail: found user id=%s for email=%s", userTag, id, email)
	return id, passwordHash, nil
}

func (q *Queries) GetUserByID(userID string) (*models.User, error) {
	log.Printf("%s GetUserByID: looking up user with userID=%s", userTag, userID)
	var email, role string
	err := q.db.QueryRow(`
		SELECT email, role
		FROM users 
		WHERE id = $1
	`, userID).Scan(&email, &role)
	if err != nil {
		log.Printf("%s GetUserByID: user not found for userID=%s: %v", userTag, userID, err)
		return nil, err
	}
	log.Printf("%s GetUserByID: found email=%s for userID=%s", userTag, email, userID)
	return &models.User{
		ID:    userID,
		Email: email,
		Role:  role,
	}, nil
}
func (q *Queries) UnlockUSD(userId string, amount float64) error {
	//need to lock the row while updatin
	log.Printf("%s UnlockUSD: attempting to unlock USD %.2f for userID=%s", userTag, amount, userId)
	tx, err := q.db.Begin()
	if err != nil {
		log.Printf("%s UnlockUSD: failed to begin transaction for userID=%s: %v", userTag, userId, err)
		return err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			log.Printf("%s UnlockUSD: rolling back transaction for userID=%s due to error: %v", userTag, userId, err)
			tx.Rollback()
		} else {
			err = tx.Commit()
			if err != nil {
				log.Printf("%s UnlockUSD: failed to commit transaction for userID=%s: %v", userTag, userId, err)
			}
		}
	}()

	var currentLockedBalance float64
	err = tx.QueryRow(`
	SELECT locked_balance
	FROM users
	WHERE id = $1
	FOR UPDATE
	`, userId).Scan(&currentLockedBalance)
	if err != nil {
		log.Printf("%s UnlockUSD: failed to fetch locked balance for userID=%s: %v", userTag, userId, err)
		return err
	}
	if currentLockedBalance < amount {
		log.Printf("%s UnlockUSD: insufficient locked balance to unlock USD %.2f for userID=%s: current locked balance is %.2f", userTag, amount, userId, currentLockedBalance)
		return sql.ErrNoRows // or a custom error indicating insufficient locked balance
	}
	log.Printf("%s UnlockUSD: unlocking USD %.2f for userID=%s", userTag, amount, userId)
	_, err = tx.Exec(`
	UPDATE users
	SET locked_balance = locked_balance - $1,USD_balance = USD_balance + $1
	WHERE id = $2 AND locked_balance >= $1
	`, amount, userId)
	if err != nil {
		log.Printf("%s UnlockUSD: failed to unlock USD for userID=%s: %v", userTag, userId, err)
		return err
	}
	log.Printf("%s UnlockUSD: successfully unlocked USD %.2f for userID=%s", userTag, amount, userId)
	return nil
}
