package handler

import (
	"net/http"
	"testing"
)

func TestSignup(t *testing.T) {
	t.Run("successful signup", func(t *testing.T) {
		db := SetupTestDB(t)

		h := UserHandler{DB: db}
		userInput := `{"username": "Hareedy", "password":"12345678"}`
		c, recorder := SetupTestRequest("/signup", userInput, http.MethodPost, nil)

		h.Signup(c)

		want := http.StatusCreated

		if recorder.Code != want {
			t.Errorf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
		}

	})

	t.Run("short password", func(t *testing.T) {
		db := SetupTestDB(t)

		h := UserHandler{DB: db}
		userInput := `{"username": "Hareedy", "password":"123"}`
		c, recorder := SetupTestRequest("/signup", userInput, http.MethodPost, nil)

		h.Signup(c)

		want := http.StatusBadRequest

		if recorder.Code != want {
			t.Errorf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
		}

	})

	t.Run("duplicate username", func(t *testing.T) {
		db := SetupTestDB(t)

		h := UserHandler{DB: db}
		userInput1 := `{"username": "Hareedy", "password":"12345678"}`
		c, recorder := SetupTestRequest("/signup", userInput1, http.MethodPost, nil)
		h.Signup(c)

		userInput2 := `{"username": "Hareedy", "password":"password"}`
		c, recorder = SetupTestRequest("/signup", userInput2, http.MethodPost, nil)
		h.Signup(c)

		want := http.StatusConflict

		if recorder.Code != want {
			t.Errorf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
		}

	})
}

func TestLogin(t *testing.T) {
	t.Run("successful Login", func(t *testing.T) {
		db := SetupTestDB(t)

		h := UserHandler{DB: db}
		userAccount := `{"username": "Hareedy", "password":"12345678"}`
		c, recorder := SetupTestRequest("/signup", userAccount, http.MethodPost, nil)
		h.Signup(c)

		userInput := `{"username": "Hareedy", "password":"12345678"}`
		c, recorder = SetupTestRequest("/login", userInput, http.MethodPost, nil)
		h.Login(c)

		want := http.StatusOK

		if recorder.Code != want {
			t.Errorf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
		}

	})

	t.Run("wrong password", func(t *testing.T) {
		db := SetupTestDB(t)

		h := UserHandler{DB: db}
		userAccount := `{"username": "Hareedy", "password":"12345678"}`
		c, recorder := SetupTestRequest("/signup", userAccount, http.MethodPost, nil)
		h.Signup(c)

		userInput := `{"username": "Hareedy", "password":"password"}`
		c, recorder = SetupTestRequest("/login", userInput, http.MethodPost, nil)
		h.Login(c)

		want := http.StatusUnauthorized

		if recorder.Code != want {
			t.Errorf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
		}

	})
	t.Run("non-existent username", func(t *testing.T) {
		db := SetupTestDB(t)

		h := UserHandler{DB: db}
		userAccount := `{"username": "Hareedy", "password":"12345678"}`
		c, recorder := SetupTestRequest("/signup", userAccount, http.MethodPost, nil)
		h.Signup(c)

		userInput := `{"username": "Adam", "password":"password"}`
		c, recorder = SetupTestRequest("/login", userInput, http.MethodPost, nil)
		h.Login(c)

		want := http.StatusUnauthorized

		if recorder.Code != want {
			t.Errorf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
		}

	})
}
