package auth

import (
	"testing"
)

func TestIsAdmin(t *testing.T) {
	adminClaim := Claims{Role: "admin"}
	userClaim := Claims{Role: "user"}

	t.Run("user is an admin", func(t *testing.T) {
		got := IsAdmin(&adminClaim)
		want := true
		if got != want {
			t.Errorf("got %v, wanted %v", got, want)
		}
	})
	t.Run("user is not an admin", func(t *testing.T) {
		got := IsAdmin(&userClaim)
		want := false
		if got != want {
			t.Errorf("got %v, wanted %v", got, want)
		}
	})
}
