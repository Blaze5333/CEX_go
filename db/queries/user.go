package queries

import "database/sql"

type Queries struct {
	db *sql.DB
}

func New(db *sql.DB) *Queries {
	return &Queries{db: db}
}

func (q *Queries) CreateUser(email, passwordHash string) (string, error) {
	var id string
	err := q.db.QueryRow(`
		INSERT INTO users (email, password_hash) 
		VALUES ($1, $2) 
		RETURNING id
	`, email, passwordHash).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}
func (q *Queries) GetUserByEmail(email string) (string, string, error) {
	var id, passwordHash string
	err := q.db.QueryRow(`
		SELECT id, password_hash 
		FROM users 
		WHERE email = $1
	`, email).Scan(&id, &passwordHash)
	if err != nil {
		return "", "", err
	}
	return id, passwordHash, nil
}
