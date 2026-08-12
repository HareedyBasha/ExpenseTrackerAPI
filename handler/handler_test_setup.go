package handler

import (
	"encoding/json"
	"expense_tracker/auth"
	"expense_tracker/model"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test db, got err = %v", err)
	}
	err = db.AutoMigrate(&model.User{}, &model.Wallet{})
	if err != nil {
		t.Fatalf("failed to migrate user to test db, got err = %v", err)
	}

	return db
}

func SetupTestRequest(path, body, requestMethod string, params gin.Params) (c *gin.Context, recorder *httptest.ResponseRecorder) {
	recorder = httptest.NewRecorder()
	c, _ = gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(requestMethod, path, strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	c.Params = params
	return c, recorder
}

func SetupTestUser(t *testing.T, db *gorm.DB, username, password, role string) (token string) {
	h := UserHandler{DB: db}
	body := fmt.Sprintf(`{"username": "%v", "password":"%v"}`, username, password)
	c, recorder := SetupTestRequest("/signup", body, http.MethodPost, nil)
	h.Signup(c)
	want := http.StatusCreated

	if recorder.Code != want {
		t.Fatalf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
	}

	result := db.Model(&model.User{}).Where("username = ?", username).Update("role", role)
	if result.Error != nil {
		t.Fatalf("got err = %v while setting user's role", result.Error)
	}

	want = http.StatusOK
	c, recorder = SetupTestRequest("/login", body, http.MethodPost, nil)
	h.Login(c)

	if recorder.Code != want {
		t.Fatalf("got status %v, wanted %v — body: %s", recorder.Code, want, recorder.Body.String())
	}

	responseBody := make(map[string]string)
	err := json.Unmarshal(recorder.Body.Bytes(), &responseBody)
	if err != nil {
		t.Fatalf("wasn't able to unmarshel response body into a valid map[string]string - got err = %v", err.Error())
	}
	return responseBody["token"]
}

func SetClaims(c *gin.Context, token string) {
	authHeader := fmt.Sprintf("Bearer %v", token)
	c.Request.Header.Set("Authorization", authHeader)
	authFunc := auth.AuthMiddleware()
	authFunc(c)
}
