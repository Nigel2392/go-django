package openauth2

import (
	"net/http"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	"github.com/Nigel2392/go-django/src/core/errs"
	"golang.org/x/oauth2"
)

const (
	ErrUserNil errs.Error = "User is nil"
)

func newSavingTokenSource(src oauth2.TokenSource, u *openauth2models.User) oauth2.TokenSource {
	return &cachedTokenSource{
		pts: src,
		u:   u,
	}
}

type cachedTokenSource struct {
	pts oauth2.TokenSource // called when t is expired.
	u   *openauth2models.User
}

func (s *cachedTokenSource) Token() (*oauth2.Token, error) {
	t, err := s.pts.Token()
	if err != nil {
		return nil, err
	}

	// Save the token to the user.
	if s.u != nil {
		s.u.SetToken(t)
		err = App.Querier().UpdateUser(
			s.u.Context(),
			s.u.ProviderName,
			s.u.Data,
			t.AccessToken,
			t.RefreshToken,
			t.TokenType,
			t.Expiry,
			s.u.IsAdministrator,
			s.u.IsActive,
			s.u.ID,
		)
	}

	return t, err
}

// RefreshTokens refreshes the tokens for a user. It will return the new token and an error if one occurred.
// It will also update the user with the new token in the database.
func RefreshTokens(u *openauth2models.User) (*oauth2.Token, error) {
	if u == nil {
		return nil, ErrUserNil
	}
	var token = u.Token()
	var conf, err = App.Provider(u.ProviderName)
	if err != nil {
		return nil, err
	}

	var ts = conf.Oauth2.TokenSource(
		u.Context(),
		token,
	)

	newToken, err := ts.Token()
	if err != nil {
		return nil, err
	}

	u.SetToken(newToken)
	err = App.Querier().UpdateUser(
		u.Context(),
		u.ProviderName,
		u.Data,
		newToken.AccessToken,
		newToken.RefreshToken,
		newToken.TokenType,
		newToken.Expiry,
		u.IsAdministrator,
		u.IsActive,
		u.ID,
	)

	return newToken, err
}

func Client(u *openauth2models.User) (*http.Client, error) {
	if u == nil {
		return nil, ErrUserNil
	}
	var token = u.Token()
	var conf, err = App.Provider(u.ProviderName)
	if err != nil {
		return nil, err
	}

	var ts = conf.Oauth2.TokenSource(
		u.Context(),
		token,
	)

	var cts = newSavingTokenSource(ts, u)
	return oauth2.NewClient(
		u.Context(),
		cts,
	), nil
}
