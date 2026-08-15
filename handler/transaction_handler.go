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

	var page, limit int

	if c.Query("page") == "" {
		page = 1
	}

	if c.Query("limit") == "" {
		limit = 20
	}

	page, err = strconv.Atoi(c.Query("page"))
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse page": err})
		return
	}

	limit, err = strconv.Atoi(c.Query("limit"))
	if err != nil {
		response.RespondError(c, http.StatusBadRequest, gin.H{"couldn't parse limit": err})
		return
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

func (h *TransactionHandler) GetTransactionSummary(c *gin.Context) {
	claims, err, statusCode := claimsChecks(c)
	if err != nil {
		response.RespondError(c, statusCode, err)
		return
	}

	type CategorySummary struct {
		Category string
		Total    int
		Count    int
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

	type SummaryResult struct {
		Period     string            `json:"period"`
		Total      int               `json:"total"`
		Categories []CategorySummary `json:"categories"`
	}

	var total int
	for _, summary := range summaries {
		total += summary.Total
	}

	summaryResult := SummaryResult{Period: now.Format("2006-01"), Total: total, Categories: summaries}

	response.RespondOK(c, summaryResult)

}
