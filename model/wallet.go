package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint           `json:"id" gorm:"primarykey" example:"4"`
	CreatedAt time.Time      `json:"created_at" gorm:"timestamptz" example:"2026-08-13T22:08:50.997585+03:00"`
	UpdatedAt time.Time      `json:"updated_at" gorm:"timestamptz" example:"2026-08-13T22:08:50.997585+03:00"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`
}

type Wallet struct {
	Model
	UserID  uint `json:"user_id" gorm:"foreignkey"`
	Balance uint `json:"balance"`
}

func (wallet *Wallet) Deposit(amount uint) {
	wallet.Balance += amount
}

func (wallet *Wallet) Withdraw(amount uint) error {
	if wallet.Balance >= amount {
		wallet.Balance -= amount
		return nil
	}

	return errors.New("insufficient balance")
}
