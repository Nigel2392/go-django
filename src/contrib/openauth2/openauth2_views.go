package openauth2

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/views"
	"github.com/Nigel2392/mux/middleware/authentication"
	"golang.org/x/oauth2"
)

func (oa *OpenAuth2AppConfig) AuthHandler(w http.ResponseWriter, r *http.Request, a *AuthConfig) {
	var user = authentication.Retrieve(r)
	if user != nil && user.IsAuthenticated() {
		var redirectURL = r.URL.Query().Get(
			"next",
		)
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

	// Handle the authentication logic here
	var url = a.Oauth2.AuthCodeURL("state", oauth2.AccessTypeOffline)
	http.Redirect(w, r, url, http.StatusFound)
	logger.Infof("AuthHandler called for provider: %s (%s)", a.Provider, url)
}

func (oa *OpenAuth2AppConfig) CallbackHandler(w http.ResponseWriter, r *http.Request, a *AuthConfig) {
	var code = r.URL.Query().Get("code")
	if code == "" {
		except.Fail(
			http.StatusBadRequest,
			"Missing code in URL",
		)
		return
	}

	token, err := a.Oauth2.Exchange(r.Context(), code)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to exchange code for authentication token",
		)
		return
	}

	if a.DataStructURL == "" {
		logger.Warnf("DataStructURL was not provided, incomplete Oauth2 flow")
		except.Fail(
			http.StatusInternalServerError,
			"Internal server error",
		)
		return
	}

	client := a.Oauth2.Client(r.Context(), token)
	resp, err := client.Get(a.DataStructURL)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to get data from provider",
		)
		return
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to get data from provider",
		)
		return
	}

	data, err := a.ScanStruct(resp.Body)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to scan data into struct",
		)
		return
	}

	logger.Debugf("Data received from provider: %+v", data)

	identifier, err := a.DataStructIdentifier(token, data)
	if err != nil {
		except.Fail(
			http.StatusInternalServerError,
			"Failed to get identifier from data struct",
		)
		return
	}

	if identifier == "" {
		except.Fail(
			http.StatusInternalServerError,
			"Identifier from data struct is empty",
		)
		return
	}

	logger.Debugf("Identifier from data struct: %s", identifier)

	user, err := oa.queryset.RetrieveUserByIdentifier(
		r.Context(), identifier,
	)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Errorf("Failed to retrieve user from database: %s", err)
		except.Fail(
			http.StatusInternalServerError,
			"Failed to retrieve user from database",
		)
		return
	} else if err != nil && errors.Is(err, sql.ErrNoRows) {
		logger.Debug("User not found in database, creating new user")
		// User not found, create a new user
		var rawData, err = json.Marshal(data)
		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				"Failed to marshal data into JSON",
			)
			return
		}

		lastId, err := oa.queryset.CreateUser(
			r.Context(),
			identifier,
			json.RawMessage(rawData),
			false,
			!oa.Config.UserDefaultIsDisabled,
		)
		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				"Failed to create user in database",
			)
			return
		}

		user, err = oa.queryset.RetrieveUserByID(
			r.Context(), lastId,
		)
		if err != nil {
			except.Fail(
				http.StatusInternalServerError,
				"Failed to retrieve user from database",
			)
			return
		}
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

	var redirectURL string
	if oa.Config.RedirectAfterLogin != nil {
		redirectURL = oa.Config.RedirectAfterLogin(data, r)
	}
	if redirectURL == "" {
		redirectURL = "/"
	}
	logger.Debugf("User logged in successfully, redirecting to: %s", redirectURL)
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
	var redirectURL = r.URL.Query().Get(
		"next",
	)
	if redirectURL == "" {
		if oa.Config.RedirectAfterLogout != nil {
			redirectURL = oa.Config.RedirectAfterLogout(r)
		}
	}
	var u = authentication.Retrieve(
		r,
	)
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
