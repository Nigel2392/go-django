package auth_test

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/Nigel2392/django/contrib/auth"
	auth_models "github.com/Nigel2392/django/contrib/auth/auth-models"
	autherrors "github.com/Nigel2392/django/contrib/auth/auth_errors"
	"github.com/Nigel2392/django/core/errs"
	"github.com/Nigel2392/django/forms/fields"
	"github.com/Nigel2392/django/models"
)

var backend models.Backend[auth_models.Querier]

func init() {
	var (
		DB_FLAVOR = "sqlite3"
		DB_SOURCE = ":memory:"
	)

	var db, err = sql.Open(DB_FLAVOR, DB_SOURCE)
	if err != nil {
		panic(err)
	}

	backend, err = auth_models.BackendForDB(db.Driver())
	if err != nil {
		panic(err)
	}

	if err := backend.CreateTable(db); err != nil {
		panic(err)
	}

	auth.Auth.Queries, err = auth_models.NewQueries(db)
	if err != nil {
		panic(err)
	}
}

type testUser struct {
	ID              uint64
	Email           string
	Username        string
	Password        string
	PasswordConfirm string
	FirstName       string
	LastName        string
	IsAdministrator bool
	IsActive        bool
	IsLoggedIn      bool
}

type formsTest struct {
	user          testUser
	formConfig    auth.RegisterFormConfig
	shouldError   bool
	expectedError error
}

var formsTests = []formsTest{
	{
		user: testUser{
			Email:           "test1@localhost",
			Username:        "test1",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			FirstName:       "Test",
			LastName:        "User",
			IsAdministrator: false,
			IsActive:        true,
			IsLoggedIn:      false,
		},
		formConfig: auth.RegisterFormConfig{
			AskForNames: true,
			IsInactive:  false,
			AutoLogin:   false,
		},
	},
	{
		user: testUser{
			Email:           "test2@localhost",
			Username:        "test2",
			Password:        "Test!123",
			PasswordConfirm: "Test!1234",
			FirstName:       "Test",
			LastName:        "User",
			IsAdministrator: false,
			IsActive:        true,
			IsLoggedIn:      false,
		},
		formConfig: auth.RegisterFormConfig{
			AskForNames: true,
			IsInactive:  false,
			AutoLogin:   false,
		},
		shouldError:   true,
		expectedError: autherrors.ErrPwdNoMatch,
	},
	{
		user: testUser{
			Email:           "test3@localhost",
			Username:        "test3",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			FirstName:       "Test",
			LastName:        "User",
			IsAdministrator: false,
			IsActive:        false,
			IsLoggedIn:      false,
		},
		formConfig: auth.RegisterFormConfig{
			AskForNames: true,
			IsInactive:  true,
			AutoLogin:   false,
		},
	},
	{
		user: testUser{
			Email:           "test4@localhost",
			Username:        "test4",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			IsAdministrator: false,
			IsActive:        true,
			IsLoggedIn:      false,
		},
		formConfig: auth.RegisterFormConfig{
			AskForNames: false,
			IsInactive:  false,
			AutoLogin:   false,
		},
	},
	{
		user: testUser{
			Email:           "test5@localhost",
			Username:        "test5",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			FirstName:       "Test",
			LastName:        "User",
			IsAdministrator: false,
			IsActive:        true,
			IsLoggedIn:      true,
		},
		formConfig: auth.RegisterFormConfig{
			AskForNames: true,
			IsInactive:  false,
			AutoLogin:   true,
		},
	},
	{
		user: testUser{
			Email:           "test6",
			Username:        "test6",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			IsActive:        true,
		},
		expectedError: errs.ErrInvalidSyntax,
	},
	{
		user: testUser{
			Email:           "test6@localhost",
			Username:        "te",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			IsActive:        true,
		},
		expectedError: errs.ErrLengthMin,
	},
	{
		user: testUser{
			Email:           "test6@localhost",
			Username:        "te123A!@#",
			Password:        "Test123!",
			PasswordConfirm: "Test123!",
			IsActive:        true,
		},
		expectedError: fields.ErrRegexInvalid,
	},
	{
		user: testUser{
			Email:           "test7@localhost",
			Username:        "test7",
			Password:        "test123!",
			PasswordConfirm: "test123!",
			IsActive:        true,
		},
		expectedError: autherrors.ErrPwdCasingUpper,
	},
	{
		user: testUser{
			Email:           "test7@localhost",
			Username:        "test7",
			Password:        "TEST1234!",
			PasswordConfirm: "TEST1234!",
			IsActive:        true,
		},
		expectedError: autherrors.ErrPwdCasingLower,
	},
	{
		user: testUser{
			Email:           "test7@localhost",
			Username:        "test7",
			Password:        "Testtttt!",
			PasswordConfirm: "Testtttt!",
			IsActive:        true,
		},
		expectedError: autherrors.ErrPwdDigits,
	},
	{
		user: testUser{
			Email:           "test7@localhost",
			Username:        "test7",
			Password:        "Test 123!",
			PasswordConfirm: "Test 123!",
			IsActive:        true,
		},
		expectedError: autherrors.ErrPwdSpaces,
	},
	{
		user: testUser{
			Email:           "test7@localhost",
			Username:        "test7",
			Password:        "Test1234",
			PasswordConfirm: "Test1234",
			IsActive:        true,
		},
		expectedError: autherrors.ErrPwdSpecial,
	},
}

func TestRegisterForm(t *testing.T) {

	for _, test := range formsTests {
		test.formConfig.AlwaysAllLoginFields = true
		test.shouldError = test.shouldError || test.expectedError != nil

		var formData = make(url.Values)
		formData.Set("email", test.user.Email)
		formData.Set("username", test.user.Username)
		formData.Set("password", test.user.Password)
		formData.Set("passwordConfirm", test.user.PasswordConfirm)
		formData.Set("firstName", test.user.FirstName)
		formData.Set("lastName", test.user.LastName)
		formData.Set("isAdministrator", "false")
		formData.Set("isActive", "true")

		t.Run(fmt.Sprintf("registerUser-%s", test.user.Username), func(t *testing.T) {
			var req, _ = http.NewRequest("POST", "/register", nil)
			var form = auth.UserRegisterForm(
				req, test.formConfig,
			)

			form.WithData(formData, nil, req)

			if !form.IsValid() {
				if !test.shouldError {
					for head := form.Errors.Front(); head != nil; head = head.Next() {
						t.Errorf("Unexpected error for %s: %v", head.Key, head.Value)
					}
					for _, err := range form.ErrorList_ {
						t.Errorf("Unexpected error: %v", err)
					}
					return
				}

				var errList = make([]error, 0)
				for head := form.Errors.Front(); head != nil; head = head.Next() {
					errList = append(errList, head.Value...)
				}
				errList = append(errList, form.ErrorList_...)

				var joined = errors.Join(errList...)
				if test.expectedError != nil && !errors.Is(joined, test.expectedError) {
					t.Errorf("Expected error %v, got %v", test.expectedError, joined)
				}

				return
			}

			if test.shouldError {
				t.Errorf("Expected error %v, got nil", test.expectedError)
				return
			}

			var user, err = form.Save()
			if err != nil {
				t.Errorf("Error saving user: %v", err)
				return
			}

			if user.Email.Address != test.user.Email {
				t.Errorf("Expected email %v, got %v", test.user.Email, user.Email)
				return
			}

			if user.Username != test.user.Username {
				t.Errorf("Expected username %v, got %v", test.user.Username, user.Username)
				return
			}

			if user.FirstName != test.user.FirstName {
				t.Errorf("Expected first name %v, got %v", test.user.FirstName, user.FirstName)
				return
			}

			if user.LastName != test.user.LastName {
				t.Errorf("Expected last name %v, got %v", test.user.LastName, user.LastName)
				return
			}

			if user.IsAdministrator != false {
				t.Errorf("Expected is administrator %v, got %v", false, user.IsAdministrator)
				return
			}

			if user.IsActive != !test.formConfig.IsInactive {
				t.Errorf("Expected is active %v, got %v", !test.formConfig.IsInactive, user.IsActive)
				return
			}

			if user.IsLoggedIn != test.formConfig.AutoLogin {
				t.Errorf("Expected is logged in %v, got %v", test.formConfig.AutoLogin, user.IsLoggedIn)
				return
			}
		})
	}
}
