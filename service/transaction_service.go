package service

import (
	"expense_tracker/model"
	"expense_tracker/repository"

	"gorm.io/gorm"
)

func NewTransaction(db *gorm.DB, walletId, amount uint, transactionType, category, note string) (model.Transaction, error) {
	wallet := model.Wallet{}
	result := repository.RetrieveByID(db, &wallet, int(walletId))
	if result.Error != nil {
		return model.Transaction{}, result.Error
	}

	transaction := model.Transaction{WalletID: walletId, RelatedWalledID: nil, Amount: amount, Type: transactionType, Category: category, Note: note}

	err := transaction.Validate()
	if err != nil {
		return model.Transaction{}, err
	}

	return transaction, nil
}

func NewTransferTransaction(db *gorm.DB, walletId, relatedWalledID, amount uint, transactionType, category, note string) (model.Transaction, error) {
	wallet := model.Wallet{}
	result := repository.RetrieveByID(db, &wallet, int(walletId))
	if result.Error != nil {
		return model.Transaction{}, result.Error
	}

	relatedWallet := model.Wallet{}
	result = repository.RetrieveByID(db, &relatedWallet, int(relatedWalledID))
	if result.Error != nil {
		return model.Transaction{}, result.Error
	}

	transaction := model.Transaction{WalletID: walletId, RelatedWalledID: &relatedWalledID, Amount: amount, Type: transactionType, Category: category, Note: note}

	err := transaction.Validate()
	if err != nil {
		return model.Transaction{}, err
	}

	return transaction, nil
}
