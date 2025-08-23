package openauth2_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/contrib/openauth2"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"golang.org/x/oauth2"
)

var shimChan = make(chan *oauth2.Token, 1)

type User struct {
	Name string `json:"name"`
}

func init() {
	var openauth = openauth2.NewAppConfig(openauth2.Config{
		AuthConfigurations: []openauth2.AuthConfig{
			{
				ProviderInfo: openauth2.ConfigInfo{
					Provider: "test",
				},
				Oauth2: &oauth2.Config{},
				GetTokenSource: func(context context.Context, token *oauth2.Token) oauth2.TokenSource {
					return &shimTokenSource{
						tkn:    token,
						newTkn: shimChan,
					}
				},
				DataConfig: openauth2.DataConfig{
					GetUniqueIdentifier: func(token *oauth2.Token, dataStruct interface{}) (string, error) {
						return dataStruct.(*User).Name, nil
					},
					Object: &User{},
				},
				UserToString: func(user *openauth2.User, dataStruct interface{}) string {
					var ds = dataStruct.(*User)
					return fmt.Sprintf("USER[%s/%s]", user.ProviderName, ds.Name)
				},
			},
		},
	})

	var _, db = testdb.Open()
	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: db,
		}),
		django.Apps(
			openauth,
		),
		django.Flag(
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
			django.FlagSkipChecks,
		),
		django.AppLogger(&logger.Logger{
			Level:       logger.DBG,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
		}),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}
}

type shimTokenSource struct {
	tkn    *oauth2.Token
	newTkn <-chan *oauth2.Token
}

var errNoToken = errors.New("Token not found")

func (s *shimTokenSource) Token() (*oauth2.Token, error) {
	select {
	case t := <-s.newTkn:
		s.tkn = t
	default:
	}
	if s.tkn == nil {
		return nil, errNoToken
	}
	return s.tkn, nil
}

func getUser(t *testing.T, id uint64) *openauth2.User {
	var user, err = openauth2.GetUserByID(context.Background(), id)
	if err != nil {
		t.Fatalf("Failed to get user by ID %d: %v", id, err)
	}
	if user == nil {
		t.Fatalf("User with ID %d not found", id)
	}
	return user
}

func TestContentObject(t *testing.T) {
	var user = &openauth2.User{
		UniqueIdentifier: "TestContentObject",
		ProviderName:     "test",
		Data:             []byte(`{"name": "Test User"}`),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		TokenType:        "Bearer",
		ExpiresAt:        drivers.Timestamp(time.Now()),
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	}

	var contentObj, err = user.ContentObject()
	if err != nil {
		t.Fatalf("Failed to get content object: %v", err)
	}

	var obj = contentObj.(*User)
	if obj.Name != "Test User" {
		t.Errorf("Expected user name 'Test User', got '%s'", obj.Name)
	}
}

func TestToString(t *testing.T) {
	var user = &openauth2.User{
		UniqueIdentifier: "TestToString",
		ProviderName:     "test",
		Data:             []byte(`{"name": "Test User"}`),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		TokenType:        "Bearer",
		ExpiresAt:        drivers.Timestamp(time.Now()),
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	}

	var s = attrs.ToString(user)
	if s != "USER[test/Test User]" {
		t.Errorf("Expected string 'USER[test/Test User]', got '%s'", s)
	}
}

func TestTokenSource(t *testing.T) {
	var ctx = context.Background()
	var timeNow = drivers.CurrentTimestamp()
	var user = models.Setup(&openauth2.User{
		UniqueIdentifier: "TestTokenSource",
		ProviderName:     "test",
		Data:             []byte(`{"name": "Test User"}`),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		TokenType:        "Bearer",
		ExpiresAt:        timeNow,
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	})

	var err = user.Save(ctx)
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("Created user: %s", attrs.ToString(user))

	//	defer func() {
	//		if _, err := queries.DeleteObject(user); err != nil {
	//			t.Fatalf("Failed to delete user: %v", err)
	//		}
	//	}()

	var ts = openauth2.TokenSource(user.SetContext(ctx))
	if ts == nil {
		t.Fatal("TokenSource returned nil")
	}

	token, err := ts.Token()
	if err != nil {
		t.Fatal(err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("Expected access token 'test-access-token', got '%s'", token.AccessToken)
	}

	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("Expected refresh token 'test-refresh-token', got '%s'", token.RefreshToken)
	}

	if token.TokenType != "Bearer" {
		t.Errorf("Expected token type 'Bearer', got '%s'", token.TokenType)
	}

	if !token.Expiry.Equal(timeNow.Time()) {
		t.Errorf("Expected expiry %s, got %s", timeNow.Time(), token.Expiry)
	}

	var expiry = timeNow.Add(1 * time.Hour)
	shimChan <- &oauth2.Token{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		Expiry:       expiry.Time(),
	}

	t.Run("TokenFromSource", func(t *testing.T) {
		// the token should be automatically saved to the user
		// in the database when it is refreshed
		token, err = ts.Token()
		if err != nil {
			t.Fatal(err)
		}

		if token.AccessToken != "new-access-token" {
			t.Errorf("Expected access token 'new-access-token', got '%s'", token.AccessToken)
		}

		if token.RefreshToken != "new-refresh-token" {
			t.Errorf("Expected refresh token 'new-refresh-token', got '%s'", token.RefreshToken)
		}

		if token.TokenType != "Bearer" {
			t.Errorf("Expected token type 'Bearer', got '%s'", token.TokenType)
		}

		if !token.Expiry.Equal(expiry.Time()) {
			t.Errorf("Expected expiry %s, got %s", expiry.Time(), token.Expiry)
		}
	})

	t.Run("CheckDatabaseUpdated", func(t *testing.T) {
		var (
			updatedUser  = getUser(t, user.ID)
			updatedToken = updatedUser.Token()
		)

		if updatedToken.AccessToken != "new-access-token" {
			t.Errorf("Expected updated user access token 'new-access-token', got '%s'", updatedUser.AccessToken)
		}

		if updatedToken.RefreshToken != "new-refresh-token" {
			t.Errorf("Expected updated user refresh token 'new-refresh-token', got '%s'", updatedUser.RefreshToken)
		}

		if updatedToken.TokenType != "Bearer" {
			t.Errorf("Expected updated user token type 'Bearer', got '%s'", updatedUser.TokenType)
		}

		if !updatedToken.Expiry.Equal(expiry.Time()) {
			t.Errorf("Expected updated user expiry %s, got %s (%s)", expiry.Time(), updatedToken.Expiry, token.Expiry)
		}
	})
}

func TestRefreshToken(t *testing.T) {
	var ctx = context.Background()
	var timeNow = drivers.CurrentTimestamp()
	var user, err = openauth2.CreateUser(ctx, &openauth2.User{
		UniqueIdentifier: "TestRefreshToken",
		ProviderName:     "test",
		Data:             []byte(`{"name": "Test User"}`),
		AccessToken:      "test-access-token",
		RefreshToken:     "test-refresh-token",
		TokenType:        "Bearer",
		ExpiresAt:        timeNow,
		Base: users.Base{
			IsAdministrator: false,
			IsActive:        true,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	//	defer func() {
	//		if _, err := queries.DeleteObject(user); err != nil {
	//			t.Fatalf("Failed to delete user: %v", err)
	//		}
	//	}()

	token, err := openauth2.RefreshTokens(user)
	if err != nil {
		t.Fatal(err)
	}

	if token.AccessToken != "test-access-token" {
		t.Errorf("Expected access token 'test-access-token', got '%s'", token.AccessToken)
	}

	if token.RefreshToken != "test-refresh-token" {
		t.Errorf("Expected refresh token 'test-refresh-token', got '%s'", token.RefreshToken)
	}

	var expiry = timeNow.Add(1 * time.Hour)
	shimChan <- &oauth2.Token{
		AccessToken:  "new-access-token",
		RefreshToken: "new-refresh-token",
		TokenType:    "Bearer",
		Expiry:       expiry.Time(),
	}

	t.Run("TokenFromSource", func(t *testing.T) {
		// the token should be automatically saved to the user
		// in the database when it is refreshed
		token, err = openauth2.RefreshTokens(user)
		if err != nil {
			t.Fatal(err)
		}

		if token.AccessToken != "new-access-token" {
			t.Errorf("Expected access token 'new-access-token', got '%s'", token.AccessToken)
		}

		if token.RefreshToken != "new-refresh-token" {
			t.Errorf("Expected refresh token 'new-refresh-token', got '%s'", token.RefreshToken)
		}

		if token.TokenType != "Bearer" {
			t.Errorf("Expected token type 'Bearer', got '%s'", token.TokenType)
		}

		if !token.Expiry.Equal(expiry.Time()) {
			t.Errorf("Expected expiry %s, got %s", expiry.Time(), token.Expiry)
		}
	})

	t.Run("CheckDatabaseUpdated", func(t *testing.T) {
		var (
			updatedUser  = getUser(t, user.ID)
			updatedToken = updatedUser.Token()
		)

		if updatedToken.AccessToken != "new-access-token" {
			t.Errorf("Expected updated user access token 'new-access-token', got '%s'", updatedUser.AccessToken)
		}

		if updatedToken.RefreshToken != "new-refresh-token" {
			t.Errorf("Expected updated user refresh token 'new-refresh-token', got '%s'", updatedUser.RefreshToken)
		}

		if updatedToken.TokenType != "Bearer" {
			t.Errorf("Expected updated user token type 'Bearer', got '%s'", updatedUser.TokenType)
		}

		if !updatedToken.Expiry.Equal(expiry.Time()) {
			t.Errorf("Expected updated user expiry %s, got %s", expiry.Time(), updatedToken.Expiry)
		}
	})
}
