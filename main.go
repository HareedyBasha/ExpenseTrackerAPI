// @title           ExpenseTracker API
// @version         1.0
// @description     REST API for wallet and transaction management with JWT auth and RBAC.

// @host            localhost:8080

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and your JWT.

package main

import (
	"expense_tracker/auth"
	"expense_tracker/database"
	"expense_tracker/handler"
	"expense_tracker/model"
	"os"

	_ "expense_tracker/docs" // triggers docs init()

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func main() {
	godotenv.Load()
	if os.Getenv("JWT_KEY") == "" {
		panic("JWT_KEY environment variable is not set")
	}

	// Create Router
	r := gin.Default()

	// Create swagger doc endpoint
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Connect to database if it exists, else create the database then connect to it
	db, err := database.ConnectAndMigrateDatabase()
	if err != nil {
		panic(err)
	}

	err = db.AutoMigrate(&model.Wallet{}, &model.User{}, &model.Transaction{})
	if err != nil {
		panic(err.Error())
	}

	userHandler := handler.UserHandler{DB: db}
	walletHandler := handler.WalletHandler{DB: db}
	transactionHandler := handler.TransactionHandler{DB: db}

	// User handling
	r.POST("/signup", userHandler.Signup)
	r.POST("/login", userHandler.Login)

	walletGroup := r.Group("/wallet")
	walletGroup.Use(auth.AuthMiddleware())

	transactionGroup := r.Group("/transactions")
	transactionGroup.Use(auth.AuthMiddleware())

	walletGroup.POST("/deposit", walletHandler.DepositToWallet)
	walletGroup.POST("/withdraw", walletHandler.WithdrawFromWallet)
	walletGroup.POST("/transfer", walletHandler.TransferFromWallet)
	walletGroup.GET("", walletHandler.GetUserWallet)

	transactionGroup.GET("", transactionHandler.GetAllTransactions)
	transactionGroup.GET("/summary", transactionHandler.GetTransactionSummary)

	r.Run("127.0.0.1:8080")
}
