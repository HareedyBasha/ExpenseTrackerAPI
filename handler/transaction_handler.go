package handler

import (
	"expense_tracker/model"
	"expense_tracker/repository"
	"expense_tracker/response"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TransactionHandler struct {
	DB *gorm.DB
}

func (h *TransactionHandler) GetAllTransactions(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	page, err := strconv.Atoi(c.Query("page"))
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse \"page\"": err})
		return
	}

	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse \"limit\"": err})
		return
	}

	var from, to *time.Time
	var category *string

	if strings.TrimSpace(c.Query("from")) != "" {
		fromValue, err := time.Parse(time.RFC3339, c.Query("from"))
		if err != nil {
			response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse \"from\"": err})
			return
		}
		from = &fromValue
	}

	if strings.TrimSpace(c.Query("to")) != "" {
		toValue, err := time.Parse(time.RFC3339, c.Query("to"))
		if err != nil {
			response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse \"to\"": err})
			return
		}
		to = &toValue
	}

	if c.Query("category") != "" {
		categoryValue := strings.ToLower(c.Query("category"))
		category = &categoryValue
	}

	var foundTransactions []model.Transaction
	err = repository.RetrieveTransactions(h.DB, claims, page, limit, from, to, category, &foundTransactions)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondOK(c, foundTransactions)
}
