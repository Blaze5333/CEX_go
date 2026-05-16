package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	// Implement password hashing logic here (e.g., using bcrypt)
	return password, nil // Placeholder: return the password as-is for now
}

func CheckPasswordHash(password, hash string) bool {
	// Implement password hash comparison logic here (e.g., using bcrypt)
	return password == hash // Placeholder: compare the password directly for now
}
func GenerateJWT(userID, email string) (string, error) {
	// Implement JWT generation logic here (e.g., using github.com/dgrijalva/jwt-go)
	//implment real logic here
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * 24 * time.Hour)), // Token expires in 10 days
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(os.Getenv("JWT_SECRET")))
}
func ValidateJWT(tokenString string) (*Claims, error) {
	// Implement JWT validation logic here (e.g., using github.com/dgrijalva/jwt-go)
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}
	return nil, fmt.Errorf("invalid token")
}
