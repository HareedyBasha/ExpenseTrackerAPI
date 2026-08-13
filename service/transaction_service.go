package service

import (
	"expense_tracker/model"
	"expense_tracker/repository"
	"strings"

	"gorm.io/gorm"
)

func NewTransaction(db *gorm.DB, walletId, amount uint, transactionType, category, note string) (model.Transaction, error) {
	wallet := model.Wallet{}
	result := repository.RetrieveByID(db, &wallet, int(walletId))
	if result.Error != nil {
		return model.Transaction{}, result.Error
	}

	transaction := model.Transaction{WalletID: walletId, RelatedWalledID: nil, Amount: amount, Type: transactionType, Category: strings.ToLower(category), Note: note}

	err := transaction.Validate()
	if err != nil {
		return model.Transaction{}, err
	}

	return transaction, nil
}

func NewTransferTransaction(db *gorm.DB, giverWalletId, takerWalletId, amount uint, category, note string) (giverTransaction model.Transaction, takerTransaction model.Transaction, err error) {
	wallet := model.Wallet{}
	result := repository.RetrieveByID(db, &wallet, int(takerWalletId))
	if result.Error != nil {
		err = result.Error
		return
	}

	relatedWallet := model.Wallet{}
	result = repository.RetrieveByID(db, &relatedWallet, int(giverWalletId))
	if result.Error != nil {
		err = result.Error
		return
	}

	giverTransaction = model.Transaction{WalletID: giverWalletId, RelatedWalledID: &takerWalletId, Amount: amount, Type: "transfer_out", Category: strings.ToLower(category), Note: note}
	err = giverTransaction.Validate()
	if err != nil {
		return
	}

	takerTransaction = model.Transaction{WalletID: takerWalletId, RelatedWalledID: &giverWalletId, Amount: amount, Type: "transfer_in", Category: strings.ToLower(category), Note: note}
	err = takerTransaction.Validate()
	if err != nil {
		return
	}

	return
}
