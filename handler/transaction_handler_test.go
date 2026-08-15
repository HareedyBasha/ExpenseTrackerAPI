package handler

import (
	"expense_tracker/auth"
	"expense_tracker/model"
	"net/http"
	"testing"
	"time"
)

func StrPtr(s string) *string        { return &s }
func TimePtr(t time.Time) *time.Time { return &t }
func IntPtr(n int) *int              { return &n }

func TestGetAllTransactions(t *testing.T) {
	userAccount := map[string]string{"username": "Hareedy", "password": "12345678", "role": "user"}

	seedData := []model.Transaction{
		{Amount: 120, Type: "withdraw", Category: "transport", Note: "last month taxi", Model: model.Model{CreatedAt: MustParse(t, "2026-07-15T00:00:00Z")}},
		{Amount: 300, Type: "withdraw", Category: "food", Note: "last month groceries", Model: model.Model{CreatedAt: MustParse(t, "2026-07-28T00:00:00Z")}},
		{Amount: 150, Type: "withdraw", Category: "food", Note: "groceries", Model: model.Model{CreatedAt: MustParse(t, "2026-08-02T00:00:00Z")}},
		{Amount: 250, Type: "withdraw", Category: "food", Note: "restaurant", Model: model.Model{CreatedAt: MustParse(t, "2026-08-05T00:00:00Z")}},
		{Amount: 80, Type: "withdraw", Category: "transport", Note: "uber", Model: model.Model{CreatedAt: MustParse(t, "2026-08-10T00:00:00Z")}},
		{Amount: 500, Type: "withdraw", Category: "entertainment", Note: "concert", Model: model.Model{CreatedAt: MustParse(t, "2026-08-20T00:00:00Z")}},
		{Amount: 5000, Type: "deposit", Category: "salary", Note: "paycheck", Model: model.Model{CreatedAt: MustParse(t, "2026-08-25T00:00:00Z")}},
	}

	type Cases struct {
		Name           string
		Filter         *Filter
		RawParams      string
		WantStatusCode int
		CheckBody      bool
	}
	testCases := []Cases{
		// --- successful retrieval & filtering ---
		{
			Name:           "no params - all transactions returned",
			Filter:         &Filter{},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "category filter - food only",
			Filter:         &Filter{Category: StrPtr("food")},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "category filter - no matches",
			Filter:         &Filter{Category: StrPtr("utilities")},
			WantStatusCode: http.StatusOK,
			CheckBody:      true, // expect empty array, not an error
		},
		{
			Name:           "date range - august only",
			Filter:         &Filter{From: TimePtr(MustParse(t, "2026-08-01T00:00:00Z")), To: TimePtr(MustParse(t, "2026-08-31T00:00:00Z"))},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "date range - excludes everything",
			Filter:         &Filter{From: TimePtr(MustParse(t, "2027-01-01T00:00:00Z")), To: TimePtr(MustParse(t, "2027-01-31T00:00:00Z"))},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "category + date range combined",
			Filter:         &Filter{Category: StrPtr("food"), From: TimePtr(MustParse(t, "2026-08-01T00:00:00Z")), To: TimePtr(MustParse(t, "2026-08-31T00:00:00Z"))},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},

		// --- pagination ---
		{
			Name:           "pagination - page 1 limit 3",
			Filter:         &Filter{Page: IntPtr(1), Limit: IntPtr(3)},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "pagination - page 2 limit 3",
			Filter:         &Filter{Page: IntPtr(2), Limit: IntPtr(3)},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "pagination - last partial page",
			Filter:         &Filter{Page: IntPtr(3), Limit: IntPtr(3)}, // 7 items, page 3 has just 1
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},
		{
			Name:           "pagination - page beyond available data",
			Filter:         &Filter{Page: IntPtr(10), Limit: IntPtr(3)},
			WantStatusCode: http.StatusOK,
			CheckBody:      true, // expect empty array, not an error — this is the case that used to panic
		},
		{
			Name:           "pagination - limit at max boundary (100)",
			Filter:         &Filter{Limit: IntPtr(100)},
			WantStatusCode: http.StatusOK,
			CheckBody:      true,
		},

		// --- malformed / rejected input ---
		{
			Name:           "non-numeric page",
			RawParams:      "?page=abc",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "non-numeric limit",
			RawParams:      "?limit=abc",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "page zero rejected",
			RawParams:      "?page=0",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "negative page rejected",
			RawParams:      "?page=-5",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "limit zero rejected",
			RawParams:      "?limit=0",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "negative limit rejected",
			RawParams:      "?limit=-10",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "limit over max (101) rejected",
			RawParams:      "?limit=101",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "malformed from date",
			RawParams:      "?from=not-a-date",
			WantStatusCode: http.StatusBadRequest,
		},
		{
			Name:           "malformed to date",
			RawParams:      "?to=not-a-date",
			WantStatusCode: http.StatusBadRequest,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			db := SetupTestDB(t)
			if testCase.Filter != nil {
				testCase.RawParams = BuildQueryString(*testCase.Filter)
			}
			c, recorder := SetupTestRequest("/transactions"+testCase.RawParams, "", http.MethodGet, nil)
			claims := SetupAuthUser(t, db, userAccount, c)

			var wallet model.Wallet
			result := db.Where("user_id = ?", claims.UserID).First(&wallet)
			if result.Error != nil {
				t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
			}

			seedIDs := SeedTransactions(t, db, wallet.ID, seedData)

			h := TransactionHandler{DB: db}

			h.GetAllTransactions(c)

			CheckStatusCode(t, testCase.WantStatusCode, recorder.Code, recorder.Body.String())

			if testCase.CheckBody {
				wantIDs := ExpectedIDs(seedIDs, seedData, *testCase.Filter)
				CompareTransactions(t, db, recorder.Body.Bytes(), wantIDs)
			}

		})
	}

	t.Run("missing claims", func(t *testing.T) {
		db := SetupTestDB(t)
		c, recorder := SetupTestRequest("/transactions", "", http.MethodGet, nil)

		h := TransactionHandler{DB: db}
		h.GetAllTransactions(c)

		CheckStatusCode(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	})

	t.Run("invalid claims", func(t *testing.T) {
		db := SetupTestDB(t)
		c, recorder := SetupTestRequest("/transactions", "", http.MethodGet, nil)
		c.Set("claims", 1234)

		h := TransactionHandler{DB: db}
		h.GetAllTransactions(c)

		CheckStatusCode(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	})

	t.Run("regular user only sees their own wallet's transactions", func(t *testing.T) {
		db := SetupTestDB(t)

		userAcc := map[string]string{"username": "Hareedy", "password": "12345678", "role": "user"}
		otherAcc := map[string]string{"username": "SomeoneElse", "password": "12345678", "role": "user"}

		c, recorder := SetupTestRequest("/transactions", "", http.MethodGet, nil)
		claims := SetupAuthUser(t, db, userAcc, c)

		var wallet model.Wallet
		db.Where("user_id = ?", claims.UserID).First(&wallet)
		wantIDs := SeedTransactions(t, db, wallet.ID, seedData)

		otherToken := SetupTestUser(t, db, otherAcc["username"], otherAcc["password"], otherAcc["role"])
		otherClaims, _ := auth.ValidateJWT(otherToken)
		var otherWallet model.Wallet
		db.Where("user_id = ?", otherClaims.UserID).First(&otherWallet)
		SeedTransactions(t, db, otherWallet.ID, []model.Transaction{
			{Amount: 999, Type: "withdraw", Category: "secret", Note: "should never appear",
				Model: model.Model{CreatedAt: MustParse(t, "2026-08-15T00:00:00Z")}},
		})

		h := TransactionHandler{DB: db}
		h.GetAllTransactions(c)

		CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
		CompareTransactions(t, db, recorder.Body.Bytes(), wantIDs)
	})
}
