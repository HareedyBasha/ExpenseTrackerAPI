package handler

import (
	"errors"
	"expense_tracker/auth"
	"expense_tracker/model"
	"expense_tracker/repository"
	"expense_tracker/response"
	"expense_tracker/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletHandler struct {
	DB *gorm.DB
}

func claimsChecks(c *gin.Context) (claims *auth.Claims, err error, statusCode int) {
	claimsValue, exists := c.Get("claims")
	if !exists {
		err = errors.New("missing claims")
		statusCode = http.StatusUnauthorized
		return
	}

	claims, ok := claimsValue.(*auth.Claims)
	if !ok {
		err = errors.New("couldn't parse claims")
		statusCode = http.StatusInternalServerError
		return
	}

	return
}

func (h *WalletHandler) DepositToWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
	}

	input := model.InputTransaction{}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var wallet model.Wallet
	var transaction model.Transaction

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", claims.UserID).First(&wallet)
		if result.Error != nil {
			return result.Error
		}

		wallet.Deposit(input.Amount)

		result = tx.Save(&wallet)
		if result.Error != nil {
			return result.Error
		}

		transaction, err = service.NewTransaction(h.DB, wallet.ID, input.Amount, "deposit", input.Category, input.Note)
		if err != nil {
			return result.Error
		}

		result = h.DB.Create(&transaction)
		if result.Error != nil {
			return result.Error
		}

		return nil
	})

	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondOK(c, gin.H{"user_id": wallet.UserID, "balance": wallet.Balance})
}

func (h *WalletHandler) WithdrawFromWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
	}

	input := model.InputTransaction{}
	if err = c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var wallet model.Wallet
	var transaction model.Transaction

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", claims.UserID).First(&wallet)
		if result.Error != nil {
			return result.Error
		}

		err = wallet.Withdraw(input.Amount)
		if err != nil {
			return err
		}

		result = tx.Save(&wallet)
		if result.Error != nil {
			return result.Error
		}

		transaction, err = service.NewTransaction(h.DB, wallet.ID, input.Amount, "withdraw", input.Category, input.Note)
		if err != nil {
			return result.Error
		}

		result = h.DB.Create(&transaction)
		if result.Error != nil {
			return result.Error
		}

		return nil
	})

	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondOK(c, gin.H{"user_id": wallet.UserID, "balance": wallet.Balance})
}

func (h *WalletHandler) GetUserWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
	}

	wallet := model.Wallet{}
	result := repository.RetrieveBy(h.DB, &wallet, "user_id", int(claims.UserID))
	if result.Error != nil {
		response.RespondError(c, http.StatusInternalServerError, result.Error)
	}

	response.RespondOK(c, gin.H{"user_id": wallet.UserID, "balance": wallet.Balance})
}
