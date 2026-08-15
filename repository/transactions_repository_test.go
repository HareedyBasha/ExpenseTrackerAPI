package repository_test

import (
	"expense_tracker/auth"
	"expense_tracker/handler"
	"expense_tracker/model"
	"expense_tracker/repository"
	"fmt"
	"testing"
	"time"
)

func StrPtr(s string) *string        { return &s }
func TimePtr(t time.Time) *time.Time { return &t }
func IntPtr(n int) *int              { return &n }

func compareTransactionIDs(t *testing.T, got []model.Transaction, wantIDs []uint) {
	t.Helper()

	if len(got) != len(wantIDs) {
		t.Fatalf("got %v transaction(s), wanted %v — got ids = %v, want ids = %v",
			len(got), len(wantIDs), idsOf(got), wantIDs)
	}

	for i, transaction := range got {
		if transaction.ID != wantIDs[i] {
			t.Errorf("position %v: got id = %v, wanted %v", i, transaction.ID, wantIDs[i])
		}
	}
}

func idsOf(transactions []model.Transaction) []uint {
	ids := make([]uint, len(transactions))
	for i, transaction := range transactions {
		ids[i] = transaction.ID
	}
	return ids
}

func TestRetrieveTransactions(t *testing.T) {
	userAccount := map[string]string{"username": "Hareedy", "password": "12345678", "role": "user"}
	seedData := []model.Transaction{
		{Amount: 120, Type: "withdraw", Category: "transport", Note: "last month taxi", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-07-15T00:00:00Z")}},
		{Amount: 300, Type: "withdraw", Category: "food", Note: "last month groceries", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-07-28T00:00:00Z")}},
		{Amount: 150, Type: "withdraw", Category: "food", Note: "groceries", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-02T00:00:00Z")}},
		{Amount: 250, Type: "withdraw", Category: "food", Note: "restaurant", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-05T00:00:00Z")}},
		{Amount: 80, Type: "withdraw", Category: "transport", Note: "uber", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-10T00:00:00Z")}},
		{Amount: 500, Type: "withdraw", Category: "entertainment", Note: "concert", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-20T00:00:00Z")}},
		{Amount: 5000, Type: "deposit", Category: "salary", Note: "paycheck", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-25T00:00:00Z")}},
	}

	type Cases struct {
		Name   string
		Filter handler.Filter
	}

	testCases := []Cases{
		{Name: "no filters - all transactions returned", Filter: handler.Filter{}},
		{Name: "category filter - food only", Filter: handler.Filter{Category: StrPtr("food")}},
		{Name: "category filter - no matches", Filter: handler.Filter{Category: StrPtr("utilities")}},
		{
			Name: "date range - august only",
			Filter: handler.Filter{
				From: TimePtr(handler.MustParse(t, "2026-08-01T00:00:00Z")),
				To:   TimePtr(handler.MustParse(t, "2026-08-31T00:00:00Z")),
			},
		},
		{
			Name: "date range - excludes everything",
			Filter: handler.Filter{
				From: TimePtr(handler.MustParse(t, "2027-01-01T00:00:00Z")),
				To:   TimePtr(handler.MustParse(t, "2027-01-31T00:00:00Z")),
			},
		},
		{
			Name: "category + date range combined",
			Filter: handler.Filter{
				Category: StrPtr("food"),
				From:     TimePtr(handler.MustParse(t, "2026-08-01T00:00:00Z")),
				To:       TimePtr(handler.MustParse(t, "2026-08-31T00:00:00Z")),
			},
		},
		{Name: "pagination - page 1 limit 3", Filter: handler.Filter{Page: IntPtr(1), Limit: IntPtr(3)}},
		{Name: "pagination - page 2 limit 3", Filter: handler.Filter{Page: IntPtr(2), Limit: IntPtr(3)}},
		{Name: "pagination - last partial page", Filter: handler.Filter{Page: IntPtr(3), Limit: IntPtr(3)}},
		{Name: "pagination - page beyond available data", Filter: handler.Filter{Page: IntPtr(10), Limit: IntPtr(3)}},
		{Name: "pagination - limit at max boundary (100)", Filter: handler.Filter{Limit: IntPtr(100)}},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			db := handler.SetupTestDB(t)

			token := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
			claims, err := auth.ValidateJWT(token)
			if err != nil {
				t.Fatalf("failed to validate token: got err = %v", err)
			}

			var wallet model.Wallet
			result := db.Where("user_id = ?", claims.UserID).First(&wallet)
			if result.Error != nil {
				t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
			}

			seedIDs := handler.SeedTransactions(t, db, wallet.ID, seedData)

			page := 1
			if testCase.Filter.Page != nil {
				page = *testCase.Filter.Page
			}
			limit := 20
			if testCase.Filter.Limit != nil {
				limit = *testCase.Filter.Limit
			}

			var got []model.Transaction
			err = repository.RetrieveTransactions(db, claims, page, limit, testCase.Filter.From, testCase.Filter.To, testCase.Filter.Category, &got)
			if err != nil {
				t.Fatalf("RetrieveTransactions returned unexpected error: %v", err)
			}

			wantIDs := handler.ExpectedIDs(seedIDs, seedData, testCase.Filter)
			compareTransactionIDs(t, got, wantIDs)
		})
	}

	t.Run("limit zero or negative defaults to one", func(t *testing.T) {
		db := handler.SetupTestDB(t)

		token := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			t.Fatalf("failed to validate token: got err = %v", err)
		}

		var wallet model.Wallet
		if result := db.Where("user_id = ?", claims.UserID).First(&wallet); result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		seedData := []model.Transaction{
			{Amount: 100, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-01T00:00:00Z")}},
			{Amount: 200, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-02T00:00:00Z")}},
			{Amount: 300, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-03T00:00:00Z")}},
		}
		seedIDs := handler.SeedTransactions(t, db, wallet.ID, seedData)

		for _, limit := range []int{0, -5} {
			t.Run(fmt.Sprintf("limit=%v", limit), func(t *testing.T) {
				var got []model.Transaction
				err := repository.RetrieveTransactions(db, claims, 1, limit, nil, nil, nil, &got)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				compareTransactionIDs(t, got, []uint{seedIDs[0]})
			})
		}
	})

	t.Run("non-positive page clamps offset to zero", func(t *testing.T) {
		db := handler.SetupTestDB(t)

		token := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			t.Fatalf("failed to validate token: got err = %v", err)
		}

		var wallet model.Wallet
		if result := db.Where("user_id = ?", claims.UserID).First(&wallet); result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		seedData := []model.Transaction{
			{Amount: 100, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-01T00:00:00Z")}},
			{Amount: 200, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-02T00:00:00Z")}},
		}
		seedIDs := handler.SeedTransactions(t, db, wallet.ID, seedData)

		for _, page := range []int{0, -5} {
			t.Run(fmt.Sprintf("page=%v", page), func(t *testing.T) {
				var got []model.Transaction
				err := repository.RetrieveTransactions(db, claims, page, 20, nil, nil, nil, &got)
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				compareTransactionIDs(t, got, seedIDs)
			})
		}
	})

	t.Run("admin sees all wallets", func(t *testing.T) {
		db := handler.SetupTestDB(t)

		userToken := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		userClaims, err := auth.ValidateJWT(userToken)
		if err != nil {
			t.Fatalf("failed to validate user token: got err = %v", err)
		}

		otherToken := handler.SetupTestUser(t, db, "SomeoneElse", userAccount["password"], userAccount["role"])
		otherClaims, err := auth.ValidateJWT(otherToken)
		if err != nil {
			t.Fatalf("failed to validate other user's token: got err = %v", err)
		}

		adminToken := handler.SetupTestUser(t, db, "AdminUser", userAccount["password"], "admin")
		adminClaims, err := auth.ValidateJWT(adminToken)
		if err != nil {
			t.Fatalf("failed to validate admin token: got err = %v", err)
		}

		var wallet, otherWallet model.Wallet
		db.Where("user_id = ?", userClaims.UserID).First(&wallet)
		db.Where("user_id = ?", otherClaims.UserID).First(&otherWallet)

		userIDs := handler.SeedTransactions(t, db, wallet.ID, []model.Transaction{
			{Amount: 100, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-01T00:00:00Z")}},
		})
		otherIDs := handler.SeedTransactions(t, db, otherWallet.ID, []model.Transaction{
			{Amount: 200, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-02T00:00:00Z")}},
		})

		wantIDs := append(append([]uint{}, userIDs...), otherIDs...)

		var got []model.Transaction
		err = repository.RetrieveTransactions(db, adminClaims, 1, 20, nil, nil, nil, &got)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		compareTransactionIDs(t, got, wantIDs)
	})

	t.Run("regular user sees only own wallet", func(t *testing.T) {
		db := handler.SetupTestDB(t)

		userToken := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		userClaims, err := auth.ValidateJWT(userToken)
		if err != nil {
			t.Fatalf("failed to validate user token: got err = %v", err)
		}

		otherToken := handler.SetupTestUser(t, db, "SomeoneElse", userAccount["password"], userAccount["role"])
		otherClaims, err := auth.ValidateJWT(otherToken)
		if err != nil {
			t.Fatalf("failed to validate other user's token: got err = %v", err)
		}

		var wallet, otherWallet model.Wallet
		db.Where("user_id = ?", userClaims.UserID).First(&wallet)
		db.Where("user_id = ?", otherClaims.UserID).First(&otherWallet)

		wantIDs := handler.SeedTransactions(t, db, wallet.ID, []model.Transaction{
			{Amount: 100, Type: "withdraw", Category: "food", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-01T00:00:00Z")}},
		})
		handler.SeedTransactions(t, db, otherWallet.ID, []model.Transaction{
			{Amount: 999, Type: "withdraw", Category: "secret", Model: model.Model{CreatedAt: handler.MustParse(t, "2026-08-02T00:00:00Z")}},
		})

		var got []model.Transaction
		err = repository.RetrieveTransactions(db, userClaims, 1, 20, nil, nil, nil, &got)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		compareTransactionIDs(t, got, wantIDs)
	})

	t.Run("missing wallet returns error", func(t *testing.T) {
		db := handler.SetupTestDB(t)

		token := handler.SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		claims, err := auth.ValidateJWT(token)
		if err != nil {
			t.Fatalf("failed to validate token: got err = %v", err)
		}

		claims.UserID = 999999

		var got []model.Transaction
		err = repository.RetrieveTransactions(db, claims, 1, 20, nil, nil, nil, &got)
		if err == nil {
			t.Fatal("expected an error when the caller's wallet can't be found, got nil")
		}
	})
}
