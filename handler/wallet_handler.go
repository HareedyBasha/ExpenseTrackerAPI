package handler

import (
	"errors"
	"expense_tracker/auth"
	"expense_tracker/model"
	"expense_tracker/response"
	"expense_tracker/service"
	"net/http"
	"strconv"

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
		return
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

		transaction, err = service.NewTransaction(tx, wallet.ID, input.Amount, "deposit", input.Category, input.Note)
		if err != nil {
			return err
		}

		result = tx.Create(&transaction)
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
		return
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
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		err = wallet.Withdraw(input.Amount)
		if err != nil {
			statusCode = http.StatusUnprocessableEntity
			return err
		}

		result = tx.Save(&wallet)
		if result.Error != nil {
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		transaction, err = service.NewTransaction(tx, wallet.ID, input.Amount, "withdraw", input.Category, input.Note)
		if err != nil {
			statusCode = http.StatusInternalServerError
			return err
		}

		result = tx.Create(&transaction)
		if result.Error != nil {
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		return nil
	})

	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	response.RespondOK(c, gin.H{"user_id": wallet.UserID, "balance": wallet.Balance})
}

func (h *WalletHandler) TransferFromWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	input := model.InputTransaction{}
	if err = c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var takerUser model.User
	result := h.DB.Where("username = ?", input.ToUser).First(&takerUser)
	if result.Error != nil {
		response.RespondError(c, http.StatusNotFound, result.Error)
		return
	}

	if claims.UserID == takerUser.ID {
		response.RespondError(c, http.StatusUnprocessableEntity, gin.H{"cannot transfer to your own wallet": result.Error})
		return
	}
	var giverWallet model.Wallet
	var takerWallet model.Wallet
	var takerTransaction model.Transaction
	var giverTransaction model.Transaction

	err = h.DB.Transaction(func(tx *gorm.DB) error {

		if claims.UserID < takerUser.ID {
			result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", claims.UserID).First(&giverWallet)
			if result.Error != nil {
				statusCode = http.StatusInternalServerError
				return result.Error
			}

			result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", takerUser.ID).First(&takerWallet)
			if result.Error != nil {
				statusCode = http.StatusInternalServerError
				return result.Error
			}
		} else {
			result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", takerUser.ID).First(&takerWallet)
			if result.Error != nil {
				statusCode = http.StatusInternalServerError
				return result.Error
			}

			result = tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", claims.UserID).First(&giverWallet)
			if result.Error != nil {
				statusCode = http.StatusInternalServerError
				return result.Error
			}
		}

		err = giverWallet.Withdraw(input.Amount)
		if err != nil {
			statusCode = http.StatusUnprocessableEntity
			return err
		}

		takerWallet.Deposit(input.Amount)

		result = tx.Save(&giverWallet)
		if result.Error != nil {
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		result = tx.Save(&takerWallet)
		if result.Error != nil {
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		takerTransaction, giverTransaction, err = service.NewTransferTransaction(tx, giverWallet.ID, takerWallet.ID, input.Amount, input.Category, input.Note)
		if err != nil {
			statusCode = http.StatusInternalServerError
			return err
		}

		result = tx.Create(&giverTransaction)
		if result.Error != nil {
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		result = tx.Create(&takerTransaction)
		if result.Error != nil {
			statusCode = http.StatusInternalServerError
			return result.Error
		}

		return nil
	})

	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	response.RespondOK(c, gin.H{"user_id": giverWallet.UserID, "balance": giverWallet.Balance})
}

func (h *WalletHandler) GetUserWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	var id *uint

	if auth.IsAdmin(claims) {
		idString := c.Query("id")
		if idString != "" {
			idValue, err := strconv.Atoi(idString)
			if err != nil {
				response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse id": err})
				return
			}
			idValueUint := (uint(idValue))
			id = &idValueUint
		}
	}

	if id == nil {
		id = &claims.UserID
	}

	wallet := model.Wallet{}
	result := h.DB.Where("user_id = ?", *id).First(&wallet)
	if result.Error != nil {
		response.RespondError(c, http.StatusNotFound, result.Error)
		return
	}

	response.RespondOK(c, gin.H{"user_id": wallet.UserID, "balance": wallet.Balance})
}
