package auth

import "testing"

func TestComparePasswords(t *testing.T) {
	hash, _ := HashPassword("correct")

	t.Run("correct password", func(t *testing.T) {
		want := true
		err := ComparePasswords(hash, "correct")
		got := err == nil
		if got != want {
			t.Errorf("got %v, wanted %v", err, want)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		want := false
		err := ComparePasswords(hash, "wrong")
		got := err == nil
		if got != want {
			t.Errorf("got err %v, wanted %v", err, want)
		}
	})

}

func TestHashPassword(t *testing.T) {

	t.Run("hashing password", func(t *testing.T) {
		hash, err := HashPassword("password")
		if err != nil {
			t.Fatalf("couldn't hash password, got err = %v", err)
		}

		err = ComparePasswords(hash, "password")
		want := true
		got := err == nil
		if got != want {
			t.Errorf("hashing failed to be compared properly, wanted %v, got err %v", want, err)
		}

	})
}
