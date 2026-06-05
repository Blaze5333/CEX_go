package auth

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const authTag = "[auth]"

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
	log.Printf("%s GenerateJWT: generating token for userID=%s email=%s", authTag, userID, email)
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(10 * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		log.Printf("%s GenerateJWT: failed to sign token for userID=%s: %v", authTag, userID, err)
		return "", err
	}
	log.Printf("%s GenerateJWT: token generated successfully for userID=%s", authTag, userID)
	return signed, nil
}

func ValidateJWT(tokenString string) (*Claims, error) {
	log.Printf("%s ValidateJWT: validating token", authTag)
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		log.Printf("%s ValidateJWT: token validation failed: %v", authTag, err)
		return nil, err
	}
	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		log.Printf("%s ValidateJWT: token valid for userID=%s", authTag, claims.UserID)

		return claims, nil
	}
	log.Printf("%s ValidateJWT: token is invalid", authTag)
	return nil, fmt.Errorf("invalid token")
}
