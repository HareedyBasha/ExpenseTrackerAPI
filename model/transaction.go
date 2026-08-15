package model

import (
	"errors"
	"slices"
	"strings"
)

type Transaction struct {
	Model
	WalletID        uint   `json:"wallet_id"`
	RelatedWalletID *uint  `json:"related_wallet_id"`
	Amount          uint   `json:"amount"`
	Type            string `json:"type"`
	Category        string `json:"category"`
	Note            string `json:"note"`
}

type InputTransaction struct {
	Amount   uint   `json:"amount"`
	ToUser   string `json:"to_user"`
	Note     string `json:"note"`
	Category string `json:"category"`
}

func (transaction *Transaction) Validate() (err error) {

	if !slices.Contains([]string{"deposit", "withdraw", "transfer_in", "transfer_out"}, strings.ToLower(transaction.Type)) {
		err = errors.New("must be one of deposit, withdraw, transfer_in, or transfer_out")
	}

	return err
}
