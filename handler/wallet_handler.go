package handler

import (
	"errors"
	"expense_tracker/auth"
	"expense_tracker/dto"
	"expense_tracker/model"
	"expense_tracker/response"
	"expense_tracker/service"
	"net/http"
	"strconv"
	"strings"

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

// @Summary Deposit funds to wallet
// @Description Takes inputted amount and adds it to wallet's balance
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body dto.DepositRequest true "Deposit payload"
// @Success 200 {object}  dto.Wallet200Response
// @Failure 400 {object} dto.Error400 "invalid request body"
// @Failure 401 {object} dto.Error401 "invalid authorization header"
// @Failure 500 {object} dto.Error500 "failed to correctly query the database or parse claims"
// @Security BearerAuth
// @Router /wallet/deposit [post]
func (h *WalletHandler) DepositToWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	input := dto.DepositRequest{}
	if err := c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	if input.Amount == 0 {
		response.RespondError(c, http.StatusBadRequest, "amount cannot be zero")
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

	walletResponse := dto.Wallet200Response{Balance: wallet.Balance, UserID: wallet.UserID}
	response.RespondOK(c, walletResponse)
}

// @Summary Withdraws funds from wallet
// @Description Takes inputted amount and subtracts it from wallet's balance
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body dto.WithdrawRequest true "Withdraw payload"
// @Success 200 {object}  dto.Wallet200Response
// @Failure 400 {object} dto.Error400 "invalid request body"
// @Failure 401 {object} dto.Error401 "invalid authorization header"
// @Failure 422 {object} dto.Error422Withdraw "not enough balance to withdraw"
// @Failure 500 {object} dto.Error500 "failed to correctly query the database or parse claims"
// @Security BearerAuth
// @Router /wallet/withdraw [post]
func (h *WalletHandler) WithdrawFromWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	input := dto.WithdrawRequest{}
	if err = c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	if input.Amount == 0 {
		response.RespondError(c, http.StatusBadRequest, "amount cannot be zero")
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

	walletResponse := dto.Wallet200Response{Balance: wallet.Balance, UserID: wallet.UserID}
	response.RespondOK(c, walletResponse)
}

// @Summary Transfers funds between users
// @Description Takes inputted amount from user's wallet and transfers it to another user's wallet
// @Tags wallet
// @Accept json
// @Produce json
// @Param request body dto.TransferRequest true "Transfer payload"
// @Success 200 {object}  dto.Wallet200Response
// @Failure 400 {object} dto.Error400 "invalid request body"
// @Failure 401 {object} dto.Error401 "invalid authorization header"
// @Failure 404 {object} dto.Error404 "inputted username not found in database"
// @Failure 422 {object} dto.Error422Transfer "cannot transfer to self"
// @Failure 500 {object} dto.Error500 "failed to correctly query the database or parse claims"
// @Security BearerAuth
// @Router /wallet/transfer [post]
func (h *WalletHandler) TransferFromWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	input := dto.TransferRequest{}
	if err = c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	if input.Amount == 0 {
		response.RespondError(c, http.StatusBadRequest, "amount cannot be zero")
		return
	}

	if strings.TrimSpace(input.ToUser) == "" {
		response.RespondError(c, http.StatusBadRequest, "to_user cannot be empty")
		return
	}

	var takerUser model.User
	result := h.DB.Where("username = ?", input.ToUser).First(&takerUser)
	if result.Error != nil {
		response.RespondError(c, http.StatusNotFound, result.Error)
		return
	}

	if claims.UserID == takerUser.ID {
		response.RespondError(c, http.StatusUnprocessableEntity, "cannot transfer to your own wallet")
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

	walletResponse := dto.Wallet200Response{Balance: uint(giverWallet.Balance), UserID: giverWallet.UserID}
	response.RespondOK(c, walletResponse)
}

// @Summary Retrieves user's wallet info
// @Description Shows user's wallet's balance and user id or another user's wallet if the issuer is an admin and inputs an id as a query parameter
// @Tags wallet
// @Produce json
// @Param id query int false "User's id if admin wants to query another user's wallet"
// @Success 200 {object}  dto.Wallet200Response
// @Failure 400 {object} dto.Error400 "invalid request body"
// @Failure 401 {object} dto.Error401 "invalid authorization"
// @Failure 403 {object} dto.Error403 "user not authorized to view this wallet"
// @Failure 404 {object} dto.Error404 "inputted username not found in database"
// @Failure 500 {object} dto.Error500 "failed to correctly query the database or parse claims"
// @Security BearerAuth
// @Router /wallet [get]
func (h *WalletHandler) GetUserWallet(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	var id *uint

	idString := c.Query("id")
	if idString != "" {
		idValue, err := strconv.Atoi(idString)
		if err != nil {
			response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse id": err})
			return
		}
		if !auth.IsAdmin(claims) && uint(idValue) != claims.UserID {
			response.RespondError(c, http.StatusForbidden, "user not authorized to view this wallet")
			return
		}
		idValueUint := uint(idValue)
		id = &idValueUint
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

	walletResponse := dto.Wallet200Response{Balance: wallet.Balance, UserID: wallet.UserID}
	response.RespondOK(c, walletResponse)
}
