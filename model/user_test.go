package model

import (
	"maps"
	"testing"
)

func validUserInput() UserInput {
	return UserInput{Username: "Hareedy", Password: "12345678"}
}

func TestValidateUserInput(t *testing.T) {

	type Cases struct {
		Name   string
		Modify func(*UserInput)
		Want   map[string]string
	}

	cases := []Cases{
		{
			Name:   "correct input",
			Modify: func(user *UserInput) {},
			Want:   make(map[string]string)},
		{
			Name:   "empty username",
			Modify: func(user *UserInput) { user.Username = "" },
			Want:   map[string]string{"username": "cannot be empty"}},
		{
			Name:   "short password",
			Modify: func(user *UserInput) { user.Password = "123" },
			Want:   map[string]string{"password": "must be at least 8 characters"}},
	}

	for _, testCase := range cases {
		t.Run(testCase.Name, func(t *testing.T) {
			user := validUserInput()
			testCase.Modify(&user)
			errs := user.Validate()
			if !maps.Equal(testCase.Want, errs) {
				t.Errorf("got errs = %v, wanted %v", errs, testCase.Want)
			}
		})
	}
}
