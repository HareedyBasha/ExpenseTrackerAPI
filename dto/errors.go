package dto

type Error400 struct {
	Error string `json:"error" example:"Invalid request body"`
}

type Error401 struct {
	Error string `json:"error" example:"missing claims"`
}

type Error403 struct {
	Error string `json:"error" example:"user not authorized to view this wallet"`
}

type Error404 struct {
	Error string `json:"error" example:"username not found in records"`
}

type Error422Transfer struct {
	Error string `json:"error" example:"cannot transfer to self"`
}

type Error422Withdraw struct {
	Error string `json:"error" example:"insufficient funds"`
}

type Error500 struct {
	Error string `json:"error" example:"failed to query database"`
}