package service

import "expense_tracker/model"

func NewWallet(user model.User) model.Wallet {
	return model.Wallet{UserID: user.ID, Balance: 0}
}
