package service

import (
	"expense_tracker/model"
	"expense_tracker/repository"

	"gorm.io/gorm"
)

func NewUser(username, hashedPassword string) model.User {
	user := model.User{Username: username, Password: hashedPassword, Role: "user"}
	return user
}

func IsUsernameTaken(db *gorm.DB, username string) (bool, error) {
	var foundUser model.User
	result := repository.RetrieveBy(db, &foundUser, "username", username)
	if result.Error != nil {
		return false, result.Error
	}

	if foundUser.Username != "" {
		return true, nil
	}

	return false, nil
}
