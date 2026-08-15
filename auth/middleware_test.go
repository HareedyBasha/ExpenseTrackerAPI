package auth_test

import (
	"expense_tracker/auth"
	"expense_tracker/handler"
	"net/http"
	"testing"
)

func TestAuthMiddleware(t *testing.T) {
	userAccount := map[string]string{"username": "Hareedy", "password": "12345678", "role": "user"}
	t.Run("valid authentication header", func(t *testing.T) {
		db := handler.SetupTestDB(t)
		c, recorder := handler.SetupTestRequest("/protected", "", http.MethodGet, nil)
		token := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])

		c.Request.Header.Set("Authorization", "Bearer "+token)

		auth.AuthMiddleware()(c)
		if c.IsAborted() {
			t.Errorf("expected request to proceed, got aborted — body: %s", recorder.Body.String())
		}
	})

	t.Run("missing authentication header", func(t *testing.T) {
		c, _ := handler.SetupTestRequest("/protected", "", http.MethodGet, nil)

		auth.AuthMiddleware()(c)
		if !c.IsAborted() {
			t.Error("expected request to abort, got proceed")
		}
	})

	t.Run("forged token", func(t *testing.T) {
		db := handler.SetupTestDB(t)
		c, _ := handler.SetupTestRequest("/protected", "", http.MethodGet, nil)
		token := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		last := string(token[len(token)-1])
		if last == "x" {
			last = "y"
		} else {
			last = "x"
		}
		forgedToken := token[:len(token)-1] + last
		c.Request.Header.Set("Authorization", "Bearer "+forgedToken)

		auth.AuthMiddleware()(c)
		if !c.IsAborted() {
			t.Error("expected request to abort, got proceed")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		db := handler.SetupTestDB(t)
		c, _ := handler.SetupTestRequest("/protected", "", http.MethodGet, nil)
		claims := handler.SetupAuthUser(t, db, userAccount, c)
		token := auth.GenerateExpiredJWT(claims)

		last := string(token[len(token)-1])
		if last == "x" {
			last = "y"
		} else {
			last = "x"
		}
		forgedToken := token[:len(token)-1] + last
		c.Request.Header.Set("Authorization", "Bearer "+forgedToken)

		auth.AuthMiddleware()(c)
		if !c.IsAborted() {
			t.Error("expected request to abort, got proceed")
		}
	})
}
