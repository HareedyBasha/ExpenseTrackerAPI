package model

import "strings"

type UserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type User struct {
	Model
	Username string `json:"username" gorm:"uniqueIndex"`
	Password string `json:"-"`
	Role     string `json:"role"`
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
