package handler

import (
	"expense_tracker/auth"
	"expense_tracker/model"
	"expense_tracker/repository"
	"expense_tracker/response"
	"expense_tracker/service"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB *gorm.DB
}

func (h *UserHandler) Signup(c *gin.Context) {
	var input model.UserInput

	if err := c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	errs := input.Validate()
	if len(errs) > 0 {
		response.RespondError(c, http.StatusBadRequest, errs)
		return
	}

	taken, err := service.IsUsernameTaken(h.DB, input.Username)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	if taken {
		response.RespondError(c, http.StatusConflict, "username is already taken")
		return
	}

	hashedPassword, err := auth.HashPassword(input.Password)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	newUser := service.NewUser(input.Username, hashedPassword)

	result := h.DB.Create(&newUser)
	if result.Error != nil {
		response.RespondError(c, http.StatusInternalServerError, result.Error)
		return
	}

	response.RespondCreated(c, newUser)

}

func (h *UserHandler) Login(c *gin.Context) {
	var input model.UserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.RespondError(c, http.StatusBadRequest, err)
		return
	}

	var foundUser model.User
	result := repository.RetrieveBy(h.DB, &foundUser, "username", input.Username)
	if result.Error != nil {
		response.RespondError(c, http.StatusInternalServerError, result.Error)
		return
	}

	if foundUser.Username == "" {
		// no such user — still run bcrypt against a dummy hash to burn equivalent time,
		// so this path takes as long as a real "wrong password" path
		auth.ComparePasswords("$2a$10$wrTUM4XPmaQ7rsGInQoJau4HZ.ckO2TLM/9kRMaRhFmuVGBiZw94X", input.Password)

		response.RespondError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}

	err := auth.ComparePasswords(foundUser.Password, input.Password)
	if err != nil {
		response.RespondError(c, http.StatusUnauthorized, "invalid username or password")
		return
	}

	token, err := auth.GenerateJWT(foundUser.ID, foundUser.Role)
	if err != nil {
		response.RespondError(c, http.StatusInternalServerError, err)
		return
	}

	response.RespondOK(c, gin.H{"token": token})
}
