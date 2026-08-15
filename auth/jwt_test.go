package auth

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
)

func TestGenerateJWT(t *testing.T) {
	t.Run("generate a JWT", func(t *testing.T) {
		token, err := GenerateJWT(1, "admin", "Hareedy")

		if err != nil {
			t.Fatalf("got err = %v while generating token", err)
		}

		if token == "" {
			t.Error("token was empty")
		}

		if len(strings.Split(token, ".")) != 3 {
			t.Errorf("wrong string format, got %v sections", len(strings.Split(token, ".")))
		}
	})

}

func GenerateExpiredJWT(claims *Claims) string {
	godotenv.Load()
	secret := os.Getenv("JWT_KEY")
	claims.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-24 * time.Hour))
	token, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	return token

}

func TestValidateJWT(t *testing.T) {
	validClaims := Claims{UserID: 1, Role: "user", Username: "Adam"}
	validToken, _ := GenerateJWT(validClaims.UserID, validClaims.Role, validClaims.Username)

	t.Run("valid token", func(t *testing.T) {
		claims, err := ValidateJWT(validToken)
		if err != nil {
			t.Fatalf("got err = %v while validating token", err)
		}
		if claims.UserID != (&validClaims).UserID {
			t.Errorf("got userID = %v, wanted %v", claims.UserID, validClaims.UserID)
		}
		if claims.Role != (&validClaims).Role {
			t.Errorf("got role = %v, wanted %v", claims.Role, validClaims.Role)
		}
	})

	t.Run("tampered token", func(t *testing.T) {
		tamperedClaims := Claims{UserID: 2, Role: "admin"}

		tamperedToken, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, tamperedClaims).SignedString([]byte("secret"))
		splitValidToken := (strings.Split(validToken, "."))
		splitTamperedToken := strings.Split(tamperedToken, ".")

		forgedToken := splitTamperedToken[0] + "." + splitTamperedToken[1] + "." + splitValidToken[2]

		_, err := ValidateJWT(forgedToken)
		if err == nil {
			t.Errorf("validation was succesful on a forged token")
		}

	})

	t.Run("expired token", func(t *testing.T) {
		expiredClaims := validClaims
		_, err := ValidateJWT(GenerateExpiredJWT(&expiredClaims))
		if err == nil {
			t.Error("validation was successful on an expired token")
		}
	})
}
