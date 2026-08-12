package auth

import (
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims struct {
	UserID uint
	Role   string
	jwt.RegisteredClaims
}

func GenerateJWT(userID uint, role string) (string, error) {
	// godotenv.Load() already runs once in main when using loadConfig from database.go
	secret := os.Getenv("JWT_KEY")
	claims := Claims{UserID: userID, Role: role}
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(24 * time.Hour))
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return token, err
}

func hmacKeyFunc(token *jwt.Token) (any, error) {
	if token.Method != jwt.SigningMethodHS256 {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}
	// godotenv.Load() already runs once in main when using loadConfig from database.go
	return []byte(os.Getenv("JWT_KEY")), nil
}

func ValidateJWT(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, hmacKeyFunc)
	if err != nil || !token.Valid {
		return nil, err
	}

	return claims, nil
}
