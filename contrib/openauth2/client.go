package openauth2

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	"github.com/Nigel2392/go-django/src/core/errs"
	"golang.org/x/oauth2"
)

const (
	ErrUserNil errs.Error = "User is nil"
)

func newSavingTokenSource(src oauth2.TokenSource, u *User) oauth2.TokenSource {
	if u == nil {
		panic(fmt.Errorf(
			"newSavingTokenSource: User is nil, cannot create token source for nil user: %w",
			ErrUserNil,
		))
	}
	return &savingTokenSource{
		pts: src,
		u:   u,
	}
}

type savingTokenSource struct {
	pts oauth2.TokenSource
	mu  sync.Mutex
	u   *User
}

func (s *savingTokenSource) Token() (*oauth2.Token, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.u.Token().Valid() {
		return s.u.Token(), nil
	}

	var t, err = s.pts.Token()
	if err != nil {
		return nil, err
	}

	// Save the token to the user.
	if s.u != nil && t.Valid() {
		s.u.SetToken(t)

		ctx := s.u.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		err = s.u.Save(ctx)
	}

	return t, err
}

type ProviderTokenSource struct {
	pts      oauth2.TokenSource
	Provider *AuthConfig
	User     *User
}

func (pts *ProviderTokenSource) Token() (*oauth2.Token, error) {
	return pts.pts.Token()
}

// TokenSource returns a new oauth2.TokenSource for the user.
//
// This token source will automatically refresh the access token when it expires.
func TokenSource(u *User) *ProviderTokenSource {
	var token = u.Token()
	var conf, err = App.Provider(u.ProviderName)
	if err != nil {
		panic(fmt.Errorf(
			"failed to get provider %s: %w",
			u.ProviderName, err,
		))
	}

	var ts = conf.TokenSource(
		u.Context(),
		token,
	)

	return &ProviderTokenSource{
		pts:      newSavingTokenSource(ts, u),
		Provider: conf,
		User:     u,
	}
}

// RefreshTokens refreshes the tokens for a user if the access token is expired.
// If the token is still valid, that token will be returned instead.
//
// It will return the new token and an error if one occurred.
//
// It will also update the user with the new token in the database.
func RefreshTokens(u *User) (*oauth2.Token, error) {
	if u == nil {
		return nil, ErrUserNil
	}

	return TokenSource(u).Token()
}

// Create a new HTTP client with the token source for the user.
//
// This can be used to make requests to the API of the provider.
//
// The client will automatically refresh the access token when it expires.
//
// It will also update the user with the new token in the database.
func Client(u *User) (*http.Client, error) {
	if u == nil {
		return nil, ErrUserNil
	}
	return oauth2.NewClient(u.Context(), TokenSource(u)), nil
}
