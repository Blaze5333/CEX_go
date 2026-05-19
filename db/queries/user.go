package queries

import (
	"database/sql"
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

func (q *Queries) GetUserByID(userID string) (string, error) {
	log.Printf("%s GetUserByID: looking up user with userID=%s", userTag, userID)
	var email string
	err := q.db.QueryRow(`
		SELECT email 
		FROM users 
		WHERE id = $1
	`, userID).Scan(&email)
	if err != nil {
		log.Printf("%s GetUserByID: user not found for userID=%s: %v", userTag, userID, err)
		return "", err
	}
	log.Printf("%s GetUserByID: found email=%s for userID=%s", userTag, email, userID)
	return email, nil
}
