package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RespondError(c *gin.Context, statusCode int, errorMessage any) {
	if err, ok := errorMessage.(error); ok {
		c.JSON(statusCode, gin.H{"error": err.Error()})
		return
	}
	c.JSON(statusCode, gin.H{"error": errorMessage})
}
func RespondOK(c *gin.Context, message any) {
	c.JSON(http.StatusOK, message)
}

func RespondCreated(c *gin.Context, message any) {
	c.JSON(http.StatusCreated, message)
}
