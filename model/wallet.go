package model

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Model struct {
	ID        uint           `json:"id" gorm:"primarykey"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"deleted_at" gorm:"index"`
}

type Wallet struct {
	Model
	UserID  uint `json:"user_id" gorm:"foreignkey"`
	Balance int  `json:"balance"`
}

func (wallet *Wallet) Deposit(amount uint) {
	wallet.Balance += int(amount)
}

func (wallet *Wallet) Withdraw(amount uint) error {
	if wallet.Balance >= int(amount) {
		wallet.Balance -= int(amount)
		return nil
	}

	return errors.New("insufficient balance")
}
