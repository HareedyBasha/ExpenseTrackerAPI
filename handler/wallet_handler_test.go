package handler

import (
	"expense_tracker/auth"
	"expense_tracker/model"
	"expense_tracker/repository"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
)

func TestDepositToWallet(t *testing.T) {
	userAccount := map[string]string{"username": "Hareedy", "password": "12345678", "role": "admin"}

	t.Run("successful desposit", func(t *testing.T) {
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: 1000, Note: "Don't spend it all in one place!", Category: "Salary"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		SetupAuthUser(t, db, userAccount, c)

		h := WalletHandler{DB: db}
		h.DepositToWallet(c)

		CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())

		wantBalance := 1000
		CompareWalletsBalance(t, recorder.Body.Bytes(), wantBalance)

		wantTransaction := model.Transaction{Amount: input.Amount, Note: input.Note, Category: strings.ToLower(input.Category), Type: "deposit", WalletID: 1}
		TransactionsCheck(t, db, wantTransaction, 1)
	})

	t.Run("missing claims", func(t *testing.T) {
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: 1000, Note: "Don't spend it all in one place!", Category: "Salary"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)

		h := WalletHandler{DB: db}
		h.DepositToWallet(c)

		CheckStatusCode(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	})

	t.Run("invalid claims", func(t *testing.T) {
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: 1000, Note: "Don't spend it all in one place!", Category: "Salary"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		c.Set("claims", 1234)

		h := WalletHandler{DB: db}
		h.DepositToWallet(c)

		CheckStatusCode(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	})

	t.Run("bad json body", func(t *testing.T) {
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: 1000, Note: "Don't spend it all in one place!", Category: "Salary"}
		body := fmt.Sprintf(`{"amount"=%v, "note"=%q, "category"=%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		SetupAuthUser(t, db, userAccount, c)

		h := WalletHandler{DB: db}
		h.DepositToWallet(c)

		CheckStatusCode(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	})

	t.Run("multiple concurrent deposits (race condition)", func(t *testing.T) {
		db := SetupTestPostgresDB(t)

		requestsNum := 500
		depositAmount := 1000

		input := model.InputTransaction{Amount: uint(depositAmount), Note: "Don't spend it all in one place!", Category: "Salary"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		var wg sync.WaitGroup
		wg.Add(requestsNum)
		start := make(chan struct{})

		// no *gin.Context exists yet at this point — each goroutine creates its own below,
		// so SetupAuthUser (which needs a context to attach claims to) can't be used here
		token := SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		claims, _ := auth.ValidateJWT(token)

		for i := 0; i < requestsNum; i++ {
			go func() {
				defer wg.Done()

				c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
				c.Set("claims", claims)
				h := WalletHandler{DB: db}

				<-start
				h.DepositToWallet(c)

				CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
			}()
		}

		close(start)
		wg.Wait()

		var wallet model.Wallet
		result := repository.RetrieveByID(db, &wallet, 1)
		if result.Error != nil {
			t.Fatalf("failed to retrieve wallet: got err = %v", result.Error)
		}

		wantBalance := requestsNum * depositAmount
		if wallet.Balance != wantBalance {
			t.Errorf("got balance = %v, wanted %v", wallet.Balance, wantBalance)
		}

		var count int64
		db.Model(&model.Transaction{}).Where("wallet_id = ?", wallet.ID).Count(&count)

		if count != int64(requestsNum) {
			t.Errorf("got %v transactions, wanted %v", count, requestsNum)
		}
	})
}

func TestWithdrawFromWallet(t *testing.T) {
	userAccount := map[string]string{"username": "Hareedy", "password": "12345678", "role": "admin"}

	t.Run("successful withdrawal", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount, c)

		result := db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.WithdrawFromWallet(c)

		CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())

		wantBalance := 0
		CompareWalletsBalance(t, recorder.Body.Bytes(), wantBalance)

		wantTransaction := model.Transaction{Amount: input.Amount, Note: input.Note, Category: strings.ToLower(input.Category), Type: "withdraw", WalletID: 1}
		TransactionsCheck(t, db, wantTransaction, 1)
	})

	t.Run("insuffcient funds", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount, c)

		result := db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", (amount / 2))
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.WithdrawFromWallet(c)

		CheckStatusCode(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	})

	t.Run("missing claims", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)

		h := WalletHandler{DB: db}
		h.WithdrawFromWallet(c)

		CheckStatusCode(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	})

	t.Run("invalid claims", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		c.Set("claims", 1234)

		h := WalletHandler{DB: db}
		h.WithdrawFromWallet(c)

		CheckStatusCode(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	})

	t.Run("bad json input", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"amount"=%v, "note"=%q, "category"=%q}`, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		SetupAuthUser(t, db, userAccount, c)

		h := WalletHandler{DB: db}
		h.WithdrawFromWallet(c)

		CheckStatusCode(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	})

	t.Run("multiple concurrent withdraws (race condition)", func(t *testing.T) {
		db := SetupTestPostgresDB(t)

		requestsNum := 500
		depositAmount := 1000

		input := model.InputTransaction{Amount: uint(depositAmount), Note: "Don't spend it all in one place!", Category: "Salary"}
		body := fmt.Sprintf(`{"amount":%v, "note":%q, "category":%q}`, input.Amount, input.Note, input.Category)

		// same reason as the deposit race test — no single context to attach claims to
		token := SetupTestUser(t, db, userAccount["username"], userAccount["password"], userAccount["role"])
		claims, _ := auth.ValidateJWT(token)

		result := db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", (requestsNum * depositAmount))
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		var wg sync.WaitGroup
		wg.Add(requestsNum)
		start := make(chan struct{})

		for i := 0; i < requestsNum; i++ {
			go func() {
				defer wg.Done()

				c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
				c.Set("claims", claims)
				h := WalletHandler{DB: db}

				<-start
				h.WithdrawFromWallet(c)

				CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
			}()
		}

		close(start)
		wg.Wait()

		var wallet model.Wallet
		result = repository.RetrieveByID(db, &wallet, 1)
		if result.Error != nil {
			t.Fatalf("failed to retrieve wallet: got err = %v", result.Error)
		}

		wantBalance := 0
		if wallet.Balance != wantBalance {
			t.Errorf("got balance = %v, wanted %v", wallet.Balance, wantBalance)
		}

		var count int64
		db.Model(&model.Transaction{}).Where("wallet_id = ?", wallet.ID).Count(&count)

		if count != int64(requestsNum) {
			t.Errorf("got %v transactions, wanted %v", count, requestsNum)
		}
	})
}

func TestTransferFromWallet(t *testing.T) {
	userAccount1 := map[string]string{"username": "Hareedy", "password": "12345678", "role": "user"}
	userAccount2 := map[string]string{"username": "Ahmed", "password": "verystrongpassword", "role": "user"}
	recieverUser := model.User{Username: "Adam", Password: "password", Role: "user"}

	t.Run("successful transfer", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, input.ToUser, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount1, c)

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())

		wantBalance := 0
		CompareWalletsBalance(t, recorder.Body.Bytes(), wantBalance)

		var recieverWallet model.Wallet
		result = db.Where("user_id = ?", recieverUser.ID).First(&recieverWallet)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve reciever wallet: got err = %v", result.Error)
		}

		wantBalance = amount
		if recieverWallet.Balance != wantBalance {
			t.Errorf("reciever got %v, wanted %v", recieverWallet.Balance, wantBalance)
		}

		var giverWallet model.Wallet
		db.Where("user_id = ?", claims.UserID).First(&giverWallet)

		var takerWallet model.Wallet
		db.Where("user_id = ?", recieverUser.ID).First(&takerWallet)

		giverWalletID := giverWallet.ID
		takerWalletID := takerWallet.ID

		wantTransaction := model.Transaction{Amount: input.Amount, Note: input.Note, Category: strings.ToLower(input.Category), Type: "transfer_out", WalletID: giverWalletID, RelatedWalletID: &takerWalletID}
		TransactionsCheck(t, db, wantTransaction, 2)

		wantTransaction = model.Transaction{Amount: input.Amount, Note: input.Note, Category: strings.ToLower(input.Category), Type: "transfer_in", WalletID: takerWalletID, RelatedWalletID: &giverWalletID}
		TransactionsCheck(t, db, wantTransaction, 1)
	})

	t.Run("insufficient funds", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, input.ToUser, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount1, c)

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", (amount / 2))
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	})

	t.Run("missing claims", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, input.ToUser, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)

		// need claims.UserID to seed the balance below, but must NOT call c.Set("claims", ...)
		// since this test is specifically checking the missing-claims path — so get claims
		// directly via ValidateJWT instead of SetupAuthUser (which would set them on c)
		token := SetupTestUser(t, db, userAccount1["username"], userAccount1["password"], userAccount1["role"])
		claims, _ := auth.ValidateJWT(token)

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	})

	t.Run("invalid claims", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, input.ToUser, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount1, c)
		c.Set("claims", 1234) // overwrite the valid claims SetupAuthUser just set — that's the point of this test

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	})

	t.Run("bad json input transfer", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user"=%q,"amount"=%v, "note"=%q, "category"=%q}`, input.ToUser, input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount1, c)

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	})

	t.Run("user does not exist", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, "Bob", input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount1, c)

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
	})

	t.Run("self-transfer", func(t *testing.T) {
		amount := 1000
		db := SetupTestDB(t)

		input := model.InputTransaction{ToUser: recieverUser.Username, Amount: uint(amount), Note: "I spent it all in one place...", Category: "Food"}
		body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, userAccount1["username"], input.Amount, input.Note, input.Category)

		c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
		claims := SetupAuthUser(t, db, userAccount1, c)

		result := db.Create(&recieverUser)
		if result.Error != nil {
			t.Fatalf("couldn't create reciever user: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		h := WalletHandler{DB: db}
		h.TransferFromWallet(c)

		CheckStatusCode(t, http.StatusUnprocessableEntity, recorder.Code, recorder.Body.String())
	})

	t.Run("A transfers to B while B transfers to A (race condition)", func(t *testing.T) {
		requestNum := 100
		depositNum := 100
		amount := requestNum * depositNum

		db := SetupTestPostgresDB(t)

		// two separate users, claims reused across many goroutines — no single context
		// to hand to SetupAuthUser, same reasoning as the other race tests
		token1 := SetupTestUser(t, db, userAccount1["username"], userAccount1["password"], userAccount1["role"])
		claims1, _ := auth.ValidateJWT(token1)

		token2 := SetupTestUser(t, db, userAccount2["username"], userAccount2["password"], userAccount2["role"])
		claims2, _ := auth.ValidateJWT(token2)

		result := db.Model(&model.Wallet{}).Where("user_id = ?", claims2.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		result = db.Model(&model.Wallet{}).Where("user_id = ?", claims1.UserID).Update("balance", amount)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve wallet: got err = %v", result.Error)
		}

		input1 := model.InputTransaction{ToUser: userAccount2["username"], Amount: uint(depositNum), Note: "don't spend it all in one place!", Category: "Salary"}
		input2 := model.InputTransaction{ToUser: userAccount1["username"], Amount: uint(depositNum), Note: "I don't want your money!", Category: "Return"}

		h := WalletHandler{DB: db}

		var wg sync.WaitGroup
		wg.Add(requestNum)
		start := make(chan struct{})

		for i := 0; i < requestNum; i++ {
			go func(index int) {
				if index%2 == 0 {
					defer wg.Done()

					body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, input1.ToUser, input1.Amount, input1.Note, input1.Category)

					c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
					c.Set("claims", claims1)
					<-start
					h.TransferFromWallet(c)

					CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
				} else {
					defer wg.Done()

					body := fmt.Sprintf(`{"to_user":%q,"amount":%v, "note":%q, "category":%q}`, input2.ToUser, input2.Amount, input2.Note, input2.Category)

					c, recorder := SetupTestRequest("/wallet", body, http.MethodPost, nil)
					c.Set("claims", claims2)
					<-start
					h.TransferFromWallet(c)

					CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
				}
			}(i)
		}

		close(start)
		wg.Wait()

		var recieverWallet model.Wallet
		result = db.Where("user_id = ?", claims2.UserID).First(&recieverWallet)
		if result.Error != nil {
			t.Fatalf("couldn't retrieve reciever wallet: got err = %v", result.Error)
		}

		wantBalance := amount
		if recieverWallet.Balance != wantBalance {
			t.Errorf("reciever got %v, wanted %v", recieverWallet.Balance, wantBalance)
		}

		var giverWallet model.Wallet
		db.Where("user_id = ?", claims1.UserID).First(&giverWallet)

		var takerWallet model.Wallet
		db.Where("user_id = ?", claims2.UserID).First(&takerWallet)

		giverWalletID := giverWallet.ID
		takerWalletID := takerWallet.ID

		var giverTransactionCount int64
		var takerTransactionCount int64

		result = db.Model(&model.Transaction{}).Where("wallet_id = ?", giverWalletID).Where("type = ?", "transfer_out").Count(&giverTransactionCount)
		if result.Error != nil {
			t.Fatalf("couldn't count transactions for giver wallet: got err = %v", result.Error)
		}
		if giverTransactionCount != int64(requestNum/2) {
			t.Errorf("got %v transaction(s), wanted %v", giverTransactionCount, requestNum)
		}

		result = db.Model(&model.Transaction{}).Where("wallet_id = ?", takerWalletID).Where("type = ?", "transfer_in").Count(&takerTransactionCount)
		if result.Error != nil {
			t.Fatalf("couldn't count transactions for taker wallet: got err = %v", result.Error)
		}
		if takerTransactionCount != int64(requestNum/2) {
			t.Errorf("got %v transaction(s), wanted %v", takerTransactionCount, requestNum)
		}
	})
}

func TestGetUserWallet(t *testing.T) {
	userAccount1 := map[string]string{"username": "Hareedy", "password": "12345678", "role": "admin"}
	userAccount2 := map[string]string{"username": "Adam", "password": "12345678", "role": "user"}

	t.Run("successful retrieval", func(t *testing.T) {
		db := SetupTestDB(t)

		c, recorder := SetupTestRequest("/wallet", "", http.MethodGet, nil)
		SetupAuthUser(t, db, userAccount1, c)

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
	})

	t.Run("missing claims", func(t *testing.T) {
		db := SetupTestDB(t)

		c, recorder := SetupTestRequest("/wallet", "", http.MethodGet, nil)

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusUnauthorized, recorder.Code, recorder.Body.String())
	})

	t.Run("invalid claims", func(t *testing.T) {
		db := SetupTestDB(t)

		c, recorder := SetupTestRequest("/wallet", "", http.MethodGet, nil)
		c.Set("claims", 1234)

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusInternalServerError, recorder.Code, recorder.Body.String())
	})

	t.Run("successful admin retrieval for another id", func(t *testing.T) {
		db := SetupTestDB(t)
		id := 2

		c, recorder := SetupTestRequest(fmt.Sprintf("/wallet?id=%v", id), "", http.MethodGet, nil)
		SetupAuthUser(t, db, userAccount1, c)
		_ = SetupTestUser(t, db, userAccount2["username"], userAccount2["password"], userAccount2["role"])

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusOK, recorder.Code, recorder.Body.String())
		CompareWalletsUserID(t, recorder.Body.Bytes(), id)
	})

	t.Run("wrong id format for admin retrieval", func(t *testing.T) {
		db := SetupTestDB(t)

		c, recorder := SetupTestRequest(fmt.Sprintf("/wallet?id=%v", "two"), "", http.MethodGet, nil)
		SetupAuthUser(t, db, userAccount1, c)
		_ = SetupTestUser(t, db, userAccount2["username"], userAccount2["password"], userAccount2["role"])

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusBadRequest, recorder.Code, recorder.Body.String())
	})

	t.Run("id doesn't exist for admin retrieval", func(t *testing.T) {
		db := SetupTestDB(t)

		c, recorder := SetupTestRequest(fmt.Sprintf("/wallet?id=%v", 999), "", http.MethodGet, nil)
		SetupAuthUser(t, db, userAccount1, c)
		_ = SetupTestUser(t, db, userAccount2["username"], userAccount2["password"], userAccount2["role"])

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusNotFound, recorder.Code, recorder.Body.String())
	})

	t.Run("user trying to get other user's wallets", func(t *testing.T) {
		db := SetupTestDB(t)

		c, recorder := SetupTestRequest("/wallet?id=6", "", http.MethodGet, nil)
		SetupAuthUser(t, db, userAccount2, c)

		h := WalletHandler{DB: db}
		h.GetUserWallet(c)

		CheckStatusCode(t, http.StatusForbidden, recorder.Code, recorder.Body.String())
	})
}
