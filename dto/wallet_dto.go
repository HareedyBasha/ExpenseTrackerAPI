package dto

type DepositRequest struct {
	Amount   uint   `json:"amount" binding:"required" example:"1000"`
	Category string `json:"category" example:"salary"`
	Note     string `json:"note" example:"Don't spend too much!"`
}

type WithdrawRequest struct {
	Amount   uint   `json:"amount" binding:"required" example:"500"`
	Category string `json:"category" example:"food"`
	Note     string `json:"note" example:"to buy chicken"`
}

type TransferRequest struct {
	ToUser   string `json:"to_user" binding:"required" example:"Adam"`
	Amount   uint   `json:"amount" binding:"required" example:"10000"`
	Category string `json:"category" example:"Tips"`
	Note     string `json:"note" example:"You earned it!"`
}

type Wallet200Response struct {
	Balance uint `json:"balance"  example:"20000"`
	UserID  uint `json:"user_id" example:"3"`
}
