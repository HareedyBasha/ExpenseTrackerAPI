// @title           Todo App API
// @version         1.0
// @description     A todo list API with JWT auth and RBAC
// @host            127.0.0.1:8080
// @BasePath        /

package main

import (
	"expense_tracker/auth"
	"expense_tracker/database"
	"expense_tracker/handler"
	"expense_tracker/model"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create Router
	r := gin.Default()

	// Connect to database if it exists, else create the database then connect to it
	db, err := database.ConnectAndMigrateDatabase()
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&model.Wallet{}, &model.User{}, &model.Transaction{})
	if err != nil {
		panic(err.Error())
	}

	walletHandler := handler.WalletHandler{DB: db}
	userHandler := handler.UserHandler{DB: db}

	// User handling
	r.POST("/signup", userHandler.Signup)
	r.POST("/login", userHandler.Login)

	protected := r.Group("/wallet")
	protected.Use(auth.AuthMiddleware())

	protected.POST("/deposit", walletHandler.DepositToWallet)
	protected.POST("/withdraw", walletHandler.WithdrawFromWallet)
	protected.GET("", walletHandler.GetUserWallet)

	r.Run("127.0.0.1:8080")
}
