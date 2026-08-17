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

// @Summary Retrieves user's transactions
// @Description Shows user's transactions with optional filters like category and from and to dates, where the displayed amount is determined by page and limit
// @Tags transaction
// @Produce json
// @Param page query int false "Which page of the transactions list to show"
// @Param limit query int false "How many transactions to show in each page"
// @Param category query string false "Which category of transactions to show (case-insensitive)"
// @Param from query string false "Which date to start showing transactions after in RFC3339"
// @Param to query string false "Which date to start showing transactions before in RFC3339"
// @Success 200 {array}  model.Transaction
// @Failure 400 {object} dto.Error400 "failed to parse a query param"
// @Failure 401 {object} dto.Error401 "invalid authorization"
// @Failure 500 {object} dto.Error500 "failed to correctly query the database or parse claims"
// @Security BearerAuth
// @Router /transactions [get]
func (h *TransactionHandler) GetAllTransactions(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	var page, limit int

	if c.Query("page") == "" {
		page = 1
	} else {
		page, err = strconv.Atoi(c.Query("page"))
		if err != nil {
			response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse page": err})
			return
		}

	}

	if c.Query("limit") == "" {
		limit = 20
	} else {
		limit, err = strconv.Atoi(c.Query("limit"))
		if err != nil {
			response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse limit": err})
			return
		}

	}

	if page < 1 {
		response.RespondError(c, http.StatusBadRequest, gin.H{"page": "must be >= 1"})
		return
	}

	if limit < 1 || limit > 100 { // pick a sane max
		response.RespondError(c, http.StatusBadRequest, gin.H{"limit": "must be between 1 and 100"})
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

type CategorySummary struct {
	Category string `json:"category"`
	Total    int    `json:"total"`
	Count    int    `json:"count"`
}

type SummaryResult struct {
	Period     string            `json:"period" example:"2026-08"`
	Total      int               `json:"total" example:"20000"`
	Categories []CategorySummary `json:"categories"`
}

// @Summary Retrieves a summary of user's transactions
// @Description Shows user's transactions grouped by category and formatted to be concise and nice to read
// @Tags transaction
// @Produce json
// @Success 200 {object}  SummaryResult
// @Failure 400 {object} dto.Error400 "failed to parse a query param"
// @Failure 401 {object} dto.Error401 "invalid authorization"
// @Failure 500 {object} dto.Error500 "failed to correctly query the database or parse claims"
// @Security BearerAuth
// @Router /transactions/summary [get]
func (h *TransactionHandler) GetTransactionSummary(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	endOfMonth := startOfMonth.AddDate(0, 1, 0)
	var wallet model.Wallet

	result := h.DB.Where("user_id = ?", claims.UserID).First(&wallet)
	if result.Error != nil {
		response.RespondError(c, http.StatusNotFound, result.Error)
		return
	}

	var summaries []CategorySummary
	result = h.DB.Model(&model.Transaction{}).Select("category, SUM(amount) AS total, COUNT(*) AS count").Where("wallet_id = ? AND created_at < ? AND created_at >= ?", wallet.ID, endOfMonth, startOfMonth).Group("category").Order("total DESC").Scan(&summaries)
	if result.Error != nil {
		response.RespondError(c, http.StatusInternalServerError, result.Error)
		return
	}

	var total int
	for _, summary := range summaries {
		total += summary.Total
	}

	summaryResult := SummaryResult{Period: now.Format("2006-01"), Total: total, Categories: summaries}

	response.RespondOK(c, summaryResult)

}
