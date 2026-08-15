package handler

import (
	"context"
	"encoding/json"
	"expense_tracker/auth"
	"expense_tracker/model"
	"expense_tracker/repository"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	gormpostgres "gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

func SetupTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test db, got err = %v", err)
	}
	err = db.AutoMigrate(&model.User{}, &model.Wallet{}, &model.Transaction{})
	if err != nil {
		t.Fatalf("failed to migrate user to test db, got err = %v", err)
	}

	return db
}

func SetupTestPostgresDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"docker.io/postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies())

	if err != nil {
		t.Fatalf("failed to start up postgres container: got err = %v", err)
	}

	t.Cleanup(func() {
		err := pgContainer.Terminate(ctx)
		if err != nil {
			t.Fatalf("failed to terminated postgres container: got err = %v", err)
		}
	})

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: got err = %v", err)
	}

	db, err := gorm.Open(gormpostgres.Open(connStr), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test postgres db: got err = %v", err)
	}

	err = db.AutoMigrate(&model.User{}, &model.Wallet{}, &model.Transaction{})
	if err != nil {
		t.Fatalf("failed to migrate tables: got err = %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("failed to retrieve sqlDB: got err = %v", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(25)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

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
	t.Helper()
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

func SetupAuthUser(t *testing.T, db *gorm.DB, userAccount map[string]string, c *gin.Context) (claims *auth.Claims) {
	t.Helper()
	token := SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
	claims, _ = auth.ValidateJWT(token)
	c.Set("claims", claims)
	return
}

func SetClaims(c *gin.Context, token string) {
	authHeader := fmt.Sprintf("Bearer %v", token)
	c.Request.Header.Set("Authorization", authHeader)
	authFunc := auth.AuthMiddleware()
	authFunc(c)
}

func CompareWalletsBalance(t *testing.T, gotWalletBytes []byte, wantBalance int) {
	t.Helper()
	var gotWallet model.Wallet

	err := json.Unmarshal(gotWalletBytes, &gotWallet)
	if err != nil {
		t.Fatalf("couldn't parse returned wallet - gor err = %v", err)
	}

	if gotWallet.Balance != wantBalance {
		t.Errorf("got balance = %v, wanted %v", gotWallet.Balance, wantBalance)
	}

}

func CompareWalletsUserID(t *testing.T, gotWalletBytes []byte, wantUserID int) {
	t.Helper()
	type Response struct {
		UserID  uint `json:"user_id"`
		Balance uint `json:"balance"`
	}
	var gotRespone Response

	err := json.Unmarshal(gotWalletBytes, &gotRespone)
	if err != nil {
		t.Fatalf("couldn't parse returned wallet - gor err = %v", err)
	}

	if int(gotRespone.UserID) != wantUserID {
		t.Errorf("got user_id = %v, wanted %v", gotRespone.UserID, wantUserID)
	}

}

func mustParse(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("failed to parse time %q: got err = %v", value, err)
	}
	return parsed
}

func TransactionsCheck(t *testing.T, db *gorm.DB, wantTransaction model.Transaction, gotTransactionID int) {
	t.Helper()
	var gotTransaction model.Transaction

	result := repository.RetrieveByID(db, &gotTransaction, gotTransactionID)
	if result.Error != nil {
		t.Fatalf("couldn't retreive transaction of id = %v, got err = %v", gotTransactionID, result.Error)
	}

	if wantTransaction.Amount != gotTransaction.Amount {
		t.Errorf("got amount = %v, wanted %v", gotTransaction.Amount, wantTransaction.Amount)
	}

	if wantTransaction.Type != gotTransaction.Type {
		t.Errorf("got type = %v, wanted %v", gotTransaction.Type, wantTransaction.Type)
	}

	if wantTransaction.Note != gotTransaction.Note {
		t.Errorf("got note = %v, wanted %v", gotTransaction.Note, wantTransaction.Note)
	}

	if wantTransaction.Category != gotTransaction.Category {
		t.Errorf("got category = %v, wanted %v", gotTransaction.Category, wantTransaction.Category)
	}

	if wantTransaction.WalletID != gotTransaction.WalletID {
		t.Errorf("got wallet_id = %v, wanted %v", gotTransaction.WalletID, wantTransaction.WalletID)
	}

	if wantTransaction.RelatedWalletID != nil && gotTransaction.RelatedWalletID != nil {
		if *wantTransaction.RelatedWalletID != *gotTransaction.RelatedWalletID {
			t.Errorf("got related_wallet_id = %v, wanted %v", gotTransaction.RelatedWalletID, wantTransaction.RelatedWalletID)
		}
	}
}

func SeedTransactions(t *testing.T, db *gorm.DB, walletID uint, transactions []model.Transaction) (ids []uint) {
	t.Helper()

	for _, transaction := range transactions {
		transaction.WalletID = walletID
		wantCreatedAt := transaction.CreatedAt
		result := db.Create(&transaction)
		if result.Error != nil {
			t.Fatalf("failed to create transaction: got err = %v", result.Error)
		}

		result = db.Model(&transaction).Update("created_at", wantCreatedAt)
		if result.Error != nil {
			t.Fatalf("failed to create transaction: got err = %v", result.Error)
		}

		ids = append(ids, transaction.ID)
	}

	return
}

func CompareTransactions(t *testing.T, db *gorm.DB, gotTransactionsBytes []byte, wantIDs []uint) {
	t.Helper()

	var gotTransactions []model.Transaction

	err := json.Unmarshal(gotTransactionsBytes, &gotTransactions)
	if err != nil {
		t.Fatalf("failed to unmarshal transactions body: got err = %v", err)
	}

	if len(gotTransactions) != len(wantIDs) {
		t.Fatalf("got %v transaction(s), wanted %v", len(gotTransactions), len(wantIDs))
	}
	for i, gotTransaction := range gotTransactions {
		if gotTransaction.ID != wantIDs[i] {
			t.Errorf("position %v: got id = %v, wanted %v", i, gotTransaction.ID, wantIDs[i])
			TransactionsCheck(t, db, gotTransaction, int(wantIDs[i]))
		}
	}
}

func CheckStatusCode(t *testing.T, wantCode, gotCode int, gotBody string) {
	if gotCode != wantCode {
		t.Errorf("got code = %v, wanted %v: err = %v", gotCode, wantCode, gotBody)
	}
}

type Filter struct {
	Category *string
	From     *time.Time
	To       *time.Time
	Page     *int
	Limit    *int
}

func ExpectedIDs(seedIDs []uint, seedData []model.Transaction, filter Filter) []uint {

	var matches []uint

	for i, tx := range seedData {
		if filter.Category != nil && tx.Category != *filter.Category {
			continue
		}
		if filter.From != nil && tx.CreatedAt.Before(*filter.From) {
			continue
		}
		if filter.To != nil && tx.CreatedAt.After(*filter.To) {
			continue
		}
		matches = append(matches, seedIDs[i])
	}

	if filter.Page == nil {
		page := 1
		filter.Page = &page
	}

	if filter.Limit == nil {
		limit := 20
		filter.Limit = &limit
	}

	low := (*(filter.Page) - 1) * *(filter.Limit)
	high := low + *(filter.Limit)

	if low > len(matches) {
		low = len(matches)
	}
	if high > len(matches) {
		high = len(matches)
	}

	matches = matches[low:high]

	return matches
}

func BuildQueryString(filter Filter) string {
	values := url.Values{}
	if filter.Page != nil {
		values.Set("page", strconv.Itoa(*filter.Page))
	}
	if filter.Limit != nil {
		values.Set("limit", strconv.Itoa(*filter.Limit))
	}
	if filter.Category != nil {
		values.Set("category", *filter.Category)
	}
	if filter.From != nil {
		values.Set("from", filter.From.Format(time.RFC3339))
	}
	if filter.To != nil {
		values.Set("to", filter.To.Format(time.RFC3339))
	}
	if len(values) == 0 {
		return ""
	}
	return "?" + values.Encode()
}
