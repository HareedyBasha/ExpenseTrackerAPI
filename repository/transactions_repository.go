package repository

import (
	"expense_tracker/auth"
	"expense_tracker/model"
	"time"

	"gorm.io/gorm"
)

func RetrieveTransactions(db *gorm.DB, claims *auth.Claims, page, limit int, from, to *time.Time, category *string, foundTransactions *[]model.Transaction) error {
	if !auth.IsAdmin(claims) {
		var wallet model.Wallet
		result := db.Where("user_id = ?", claims.UserID).First(&wallet)
		if result.Error != nil {
			return result.Error
		}

		db = db.Where("wallet_id = ?", wallet.ID)
	}

	if from != nil {
		db = db.Where("created_at >= ?", from)
	}

	if to != nil {
		db = db.Where("created_at < ?", to)
	}

	if category != nil {
		db = db.Where("category = ?", category)
	}

	if limit <= 0 {
		limit = 1
	}

	offset := (page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	result := db.Offset(offset).Limit(limit).Order("created_at ASC").Find(foundTransactions)
	if result.Error != nil {
		return result.Error
	}

	return nil
}
