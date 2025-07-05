package auth_test

import (
	"errors"
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/auth"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
)

type passwordTest struct {
	TestName        string
	Password        string
	Hash            string
	ExpectedError   bool
	Errors          []error
	ValidationFlags auth.PasswordCharacterFlag
}

var passwordValidationTests = []passwordTest{
	{
		TestName:        "casing-upper-mismatch",
		Password:        "password",
		ExpectedError:   true,
		Errors:          []error{autherrors.ErrPwdCasingUpper},
		ValidationFlags: auth.ChrFlagUpper,
	},
	{
		TestName:        "casing-lower-mismatch",
		Password:        "PASSWORD",
		ExpectedError:   true,
		Errors:          []error{autherrors.ErrPwdCasingLower},
		ValidationFlags: auth.ChrFlagLower,
	},
	{
		TestName:        "casing-upper|lower-match",
		Password:        "Password",
		ExpectedError:   false,
		Errors:          nil,
		ValidationFlags: auth.ChrFlagLower | auth.ChrFlagUpper,
	},
	{
		TestName:        "digit-mismatch",
		Password:        "password",
		ExpectedError:   true,
		Errors:          []error{autherrors.ErrPwdCasingUpper, autherrors.ErrPwdDigits},
		ValidationFlags: auth.ChrFlagUpper | auth.ChrFlagDigit,
	},
	{
		TestName:        "digit-match",
		Password:        "Password1",
		ExpectedError:   false,
		Errors:          nil,
		ValidationFlags: auth.ChrFlagUpper | auth.ChrFlagDigit,
	},
	{
		TestName:      "spaces-mismatch",
		Password:      "Pass word1",
		ExpectedError: true,
		Errors:        []error{autherrors.ErrPwdSpaces},
	},
	{
		TestName:        "special-mismatch",
		Password:        "Password1",
		ExpectedError:   true,
		Errors:          []error{autherrors.ErrPwdSpecial},
		ValidationFlags: auth.ChrFlagUpper | auth.ChrFlagDigit | auth.ChrFlagSpecial,
	},
	{
		TestName:        "special-match",
		Password:        "Password1!",
		ExpectedError:   false,
		Errors:          nil,
		ValidationFlags: auth.ChrFlagUpper | auth.ChrFlagDigit | auth.ChrFlagSpecial,
	},
	{
		TestName:        "all-match",
		Password:        "Password!123",
		ExpectedError:   false,
		Errors:          nil,
		ValidationFlags: auth.ChrFlagAll,
	},
}

var passwordHashTests = []passwordTest{
	{
		TestName: "match-password",
		Password: "password",
		Hash:     "$2a$10$1/Wxeai7xJZ/RHXqqQo.MOs4r2zTVkqLdICqyQVDeXhmElAoLmLNy",
	},
	{
		TestName:      "mismatch-password",
		Password:      "password",
		Hash:          "$2a$10$H9rdZ17XTgxNgq0hIe/eM.9KHhjn5AqerNCUlIgyyV2ZCdhWjwtve",
		ExpectedError: true,
		Errors:        []error{autherrors.ErrPwdHashMismatch},
	},
	{
		TestName: "match-Password",
		Password: "Password",
		Hash:     "$2a$10$H9rdZ17XTgxNgq0hIe/eM.9KHhjn5AqerNCUlIgyyV2ZCdhWjwtve",
	},
	{
		TestName: "match-Password1",
		Password: "Password1",
		Hash:     "$2a$10$eMXEd5fqGNMJqciAJDS5kegoRqUgf.JbAH3fxghqrffwONuhStm4m",
	},
	{
		TestName: "match-Password!123",
		Password: "Password!123",
		Hash:     "$2a$10$7CMDuopPFz/uiWO5Y/S8J.oJ.G0epnqppdSi4TcjP8Mo3OuQExV5q",
	},
	{
		TestName:      "hash-fail-Password!123",
		Password:      "Password!123",
		Hash:          "notHashedPassword",
		ExpectedError: true,
		Errors:        []error{errExpectedHash},
	},
}

var passwordSetTests = []passwordTest{
	{
		TestName: "set-password-ok",
		Password: "Password!123",
		Hash:     "$2a$10$7CMDuopPFz/uiWO5Y/S8J.oJ.G0epnqppdSi4TcjP8Mo3OuQExV5q",
	},
	{
		TestName: "set-password-fail",
		Password: "Password!123",
		Hash:     "$2a$10$eMXEd5fqGNMJqciAJDS5kegoRqUgf.JbAH3fxghqrffwONuhStm4m",
		Errors:   []error{autherrors.ErrPwdHashMismatch},
	},
}

var errExpectedHash = errors.New("expected hashed password, got invalid hash")

func TestPasswords(t *testing.T) {
	t.Run("Validation", func(t *testing.T) {
		for _, test := range passwordValidationTests {
			test.ExpectedError = test.ExpectedError || len(test.Errors) != 0

			t.Run(test.TestName, func(t *testing.T) {
				var validator = &auth.PasswordCharValidator{
					Flags: test.ValidationFlags,
				}

				var err = validator.Validate(test.Password)
				if test.ExpectedError {
					if err == nil {
						t.Errorf("expected error, got nil")
					}
					if len(test.Errors) != 0 {
						for _, e := range test.Errors {
							if !errors.Is(err, e) {
								t.Errorf("expected error %v, got %v", e, err)
							}
						}
					}
				} else {
					if err != nil {
						t.Errorf("expected no error, got %v", err)
					}
				}
			})
		}
	})

	t.Run("Hashing", func(t *testing.T) {
		for _, test := range passwordHashTests {
			t.Run(test.TestName, func(t *testing.T) {
				test.ExpectedError = test.ExpectedError || len(test.Errors) != 0
				var u = &auth.User{
					Password: auth.Password(test.Hash),
				}

				if !auth.IS_HASHED(string(u.Password)) {
					if test.ExpectedError && !errors.Is(errExpectedHash, test.Errors[0]) {
						t.Errorf("expected hashed password, got invalid hash")
						return
					}
					t.Logf("password is not hashed (nor expected to be), skipping test")
					return
				}

				var err = auth.CheckPassword(u, test.Password)
				if test.ExpectedError {
					if err == nil {
						t.Error("expected error, got nil")
					}
					if len(test.Errors) != 0 {
						for _, e := range test.Errors {
							if !errors.Is(err, e) {
								t.Errorf("expected error %v, got %v", e, err)
							}
						}
					}
				} else {
					if err != nil {
						t.Errorf("expected no error, got %v", err)
					}
				}
			})
		}
	})

	for _, test := range passwordSetTests {
		t.Run(test.TestName, func(t *testing.T) {
			var u = &auth.User{
				Password: auth.Password(test.Hash),
			}

			var err = auth.SetPassword(u, test.Password)
			if test.ExpectedError {
				if err == nil {
					t.Error("expected error, got nil")
				}
				if len(test.Errors) != 0 {
					for _, e := range test.Errors {
						if !errors.Is(err, e) {
							t.Errorf("expected error %v, got %v", e, err)
						}
					}
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}
