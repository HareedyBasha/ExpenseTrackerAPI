package model

import (
	"strings"

	"gorm.io/gorm"
)

type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	Model
	Username string `json:"username" gorm:"uniqueIndex"`
	Password string `json:"-"`
	Role     string `json:"role"`
	Wallet   Wallet
}

func (user UserInput) Validate() map[string]string {
	errs := make(map[string]string)

	if strings.TrimSpace(user.Username) == "" {
		errs["username"] = "cannot be empty"
	}

	if len(user.Password) < 8 {
		errs["password"] = "must be at least 8 characters"
	}

	return errs
}

func (user *User) AfterCreate(tx *gorm.DB) (err error) {
	// Create a new wallet associated with this user's ID
	wallet := Wallet{
		UserID:  user.ID,
		Balance: 0,
	}

	// Use the transaction (tx) provided by GORM to create the wallet
	if err := tx.Create(&wallet).Error; err != nil {
		return err
	}

	return nil
}
