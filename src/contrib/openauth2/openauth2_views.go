package openauth2

import (
	"encoding/json"
	"errors"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/query_errors"
	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux/middleware/authentication"
	"golang.org/x/oauth2"
)

const redirectCookieName = "openauth2.redirectURL"

func setCallbackHandlerRedirect(w http.ResponseWriter, redirectURL string, delete bool) {
	var maxAge = -1
	if delete {
		redirectURL = ""
	} else {
		maxAge = 60 * 10 // 10 minutes
	}
	http.SetCookie(w, &http.Cookie{
		Name:     redirectCookieName,
		Value:    redirectURL,
		MaxAge:   maxAge, // 10 minutes
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	})
}

func getCallbackHandlerRedirect(r *http.Request) string {
	var cookie, err = r.Cookie(redirectCookieName)
	if err != nil {
		return ""
	}
	if cookie == nil {
		return ""
	}
	return cookie.Value
}

func (oa *OpenAuth2AppConfig) AuthHandler(w http.ResponseWriter, r *http.Request, a *AuthConfig) {
	var user = authentication.Retrieve(r)
	var redirectURL = r.URL.Query().Get(
		"next",
	)
	if user != nil && user.IsAuthenticated() {
		if redirectURL == "" {
			redirectURL = "/"
		}
		http.Redirect(
			w, r,
			redirectURL,
			http.StatusSeeOther,
		)
		return
	}

	if redirectURL != "" {
		setCallbackHandlerRedirect(w, redirectURL, false)
	}

	// Handle the authentication logic here
	var state = "state"
	if a.State != "" {
		state = a.State
	}

	var accessType = oauth2.AccessTypeOffline
	if a.AccessType != nil {
		accessType = a.AccessType
	}

	var opts = make([]oauth2.AuthCodeOption, 0)
	opts = append(opts, accessType)

	// Set any extra parameters that may be needed for the provider
	if a.ExtraParams != nil {
		for k, v := range a.ExtraParams {
			opts = append(opts, oauth2.SetAuthURLParam(k, v))
		}
	}

	// Generate the URL for the provider's authentication page
	var oauthURL = a.Oauth2.AuthCodeURL(
		state,
		opts...,
	)
	http.Redirect(w, r, oauthURL, http.StatusFound)
}

func (oa *OpenAuth2AppConfig) CallbackHandler(w http.ResponseWriter, r *http.Request, a *AuthConfig) {
	var code = r.URL.Query().Get("code")
	except.Assert(
		code != "", http.StatusBadRequest,
		"Missing code in URL",
	)

	// Exchange the access code for a token
	token, err := a.Oauth2.Exchange(r.Context(), code)
	except.AssertNil(
		err, http.StatusInternalServerError,
		"Failed to exchange code for authentication token",
	)

	if a.DataStructURL == "" {
		logger.Errorf("DataStructURL was not provided, incomplete Oauth2 flow")
		except.Fail(
			http.StatusInternalServerError,
			"Internal server error",
		)
		return
	}

	// Use the token to get the user's data from the provider
	client := a.Oauth2.Client(r.Context(), token)
	resp, err := client.Get(a.DataStructURL)
	except.AssertNil(
		err, http.StatusInternalServerError,
		"Failed to get data from provider",
	)

	defer resp.Body.Close()

	except.Assert(
		resp.StatusCode == http.StatusOK,
		http.StatusInternalServerError,
		"Failed to get data from provider",
	)

	// Scan the response body into the data struct
	data, err := a.ScanContentObject(resp.Body)
	except.AssertNil(
		err, http.StatusInternalServerError,
		"Failed to scan data into struct",
	)

	// Retrieve the identifier from the data struct
	// This is used to identify the user in the database
	identifier, err := a.DataStructIdentifier(token, data)
	except.AssertNil(
		err, http.StatusInternalServerError,
		"Failed to get identifier from data struct",
	)

	except.Assert(
		identifier != "",
		http.StatusInternalServerError,
		"Identifier from data struct is empty",
	)

	// Serialize raw data into JSON
	// This is used to store the data in the database
	// on a per- provider basis
	rawData, err := json.Marshal(data)
	except.AssertNil(
		err, http.StatusInternalServerError,
		"Failed to marshal data into JSON",
	)

	logger.Debugf("Identifier from data struct: %s", identifier)

	// Check if the user already exists in the database
	user, err := GetUserByIdentifier(r.Context(), a.Provider, identifier)
	if err != nil && !errors.Is(err, query_errors.ErrNoRows) {
		// An error occurred while retrieving the user from the database
		// Log the error and return a 500 status code
		logger.Errorf("Failed to retrieve user from database: %s", err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to retrieve user from database",
		)
		return
	} else if err != nil && errors.Is(err, query_errors.ErrNoRows) {
		logger.Debug("User not found in database, creating new user")
		// User not found, create a new user in the database
		user, err = queries.GetQuerySetWithContext(r.Context(), &User{}).Create(&User{
			UniqueIdentifier: identifier,
			ProviderName:     a.Provider,
			Data:             rawData,
			AccessToken:      drivers.Text(token.AccessToken),
			RefreshToken:     drivers.Text(token.RefreshToken),
			TokenType:        token.TokenType,
			ExpiresAt:        drivers.Timestamp(token.Expiry),
			IsAdministrator:  false,
			IsActive:         !oa.Config.UserDefaultIsDisabled,
		})
		except.AssertNil(
			err, http.StatusInternalServerError,
			"Failed to create user in database",
		)

	} else if err == nil {
		logger.Debug("User found in database, updating user")
		// User found, update user token information in the database
		_, err = queries.GetQuerySetWithContext(r.Context(), &User{}).
			ExplicitSave().
			Select("Data", "AccessToken", "RefreshToken", "TokenType", "ExpiresAt").
			Filter("ID", user.ID).
			Update(user)

		except.AssertNil(
			err, http.StatusInternalServerError,
			"Failed to update user in database",
		)
	}

	_, err = Login(r, user)
	if err != nil {
		logger.Errorf("Failed to log user in: %s", err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to log user in",
		)
		return
	}

	var redirectURL = getCallbackHandlerRedirect(r)
	if redirectURL == "" {
		if oa.Config.RedirectAfterLogin != nil {
			redirectURL = oa.Config.RedirectAfterLogin(user, data, r)
		}
		if redirectURL == "" {
			redirectURL = "/"
		}
	} else {
		setCallbackHandlerRedirect(w, "", true)
	}

	logger.Debugf(
		"User logged in successfully, redirecting to: %s",
		redirectURL,
	)

	http.Redirect(
		w, r,
		redirectURL,
		http.StatusSeeOther,
	)
}

func (oa *OpenAuth2AppConfig) LoginHandler(w http.ResponseWriter, r *http.Request) {
	// Handle the authentication logic here
	var v = &views.BaseView{
		BaseTemplateKey: "oauth2",
		TemplateName:    "oauth2/login.tmpl",
	}
	views.Invoke(v, w, r)
}

func (oa *OpenAuth2AppConfig) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	var redirectURL = r.URL.Query().Get("next")
	if redirectURL == "" {
		if oa.Config.RedirectAfterLogout != nil {
			redirectURL = oa.Config.RedirectAfterLogout(r)
		}
	}

	var u = authentication.Retrieve(r)
	if u == nil || !u.IsAuthenticated() {
		http.Redirect(
			w, r,
			redirectURL,
			http.StatusSeeOther,
		)
		return
	}

	if err := Logout(r); err != nil && !errors.Is(err, autherrors.ErrNoSession) {
		logger.Errorf(
			"Failed to log user out: %v", err,
		)
		except.Fail(
			500, "Failed to log out",
		)
		return
	}

	http.Redirect(
		w, r,
		redirectURL,
		http.StatusSeeOther,
	)
}
