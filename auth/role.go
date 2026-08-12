package auth

func IsAdmin(claims *Claims) bool {
	return claims.Role == "admin"
}
