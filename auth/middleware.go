package auth

import (
	"expense_tracker/response"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			response.RespondError(c, http.StatusUnauthorized, "invalid authorization header")
			c.Abort()
			return
		}
		authToken := authHeader[7:]
		claims, err := ValidateJWT(authToken)
		if err != nil {
			response.RespondError(c, http.StatusUnauthorized, err)
			c.Abort()
			return
		}

		c.Set("claims", claims)
		c.Next()

	}
}
