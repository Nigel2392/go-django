package openauth2

import (
	"encoding/json"
	"errors"
	"io"

	openauth2models "github.com/Nigel2392/go-django/src/contrib/openauth2/openauth2_models"
	"github.com/qdm12/reprint"
	"golang.org/x/oauth2"
)

type AuthConfig struct {
	// The base oauth2 config to use.
	//
	// Under the hood the `golang.org/x/oauth2` package is used.
	Oauth2 *oauth2.Config

	// The access type to request from the provider.
	//
	// If this is nil, it will be set to "oauth2.AccessTypeOffline".
	AccessType oauth2.AuthCodeOption

	// The state to use when generating the URL
	// with Oauth2.AuthCodeURL.
	//
	// If this is left empty, it will default to "state"
	State string

	// The name of the provider, e.g. "google", "github", etc.
	Provider string

	// A nice name for the provider, e.g. "Google", "GitHub", etc.
	//
	// This is used for display purposes only.
	ProviderNiceName string

	// The URL of the documentation for the provider.
	//
	// This can be used to link to the provider's documentation.
	DocumentationURL string

	// ExtraParams are extra parameters to be set on the URL when
	// generating the url with Oauth2.AuthCodeURL
	ExtraParams map[string]string

	// An optional URL for the provider's logo.
	//
	// This is used for display purposes only.
	//
	// It is a function so it can possibly callback to django.Static(path).
	ProviderLogoURL func() string

	// DataStructURL is the URL which will be used to retrieve the data from the provider.
	//
	// This will then be used to scan the data into the DataStruct fields.
	// The URL must be a valid URL and must return a JSON object.
	DataStructURL string

	// DataStructIdentifier retrieves the unique identifier from the data struct.
	//
	// This is used to identify the user in the database.
	//
	// It has to be a function that takes the data struct and returns a string.
	DataStructIdentifier func(token *oauth2.Token, dataStruct interface{}) (string, error)

	// DataStruct is the struct that will be used to store the data returned by the provider.
	//
	// It will be copied by means of reflection, using the reprint package.
	// This means that it DOES support unexported fields, though these will
	// NOT be used for JSON unmarshalling.
	DataStruct interface{}

	// UserToString is a function that takes a user and returns a string.
	//
	// It can act on the user's data struct to return a string.
	// It is used for display purposes only.
	UserToString func(user *openauth2models.User, dataStruct interface{}) string
}

func (c *AuthConfig) ReadableName() string {
	if c.ProviderNiceName != "" {
		return c.ProviderNiceName
	}
	return c.Provider
}

func (c *AuthConfig) ScanStruct(r io.Reader) (interface{}, error) {
	if c.DataStruct == nil {
		return nil, errors.New("DataStruct was not provided")
	}

	var copy = reprint.This(c.DataStruct)
	var dec = json.NewDecoder(r)
	var err = dec.Decode(copy)
	return copy, err
}

func (c *AuthConfig) DataForUser(user *openauth2models.User) (interface{}, error) {
	if c.DataStruct == nil {
		return nil, errors.New("DataStruct was not provided")
	}

	var copy = reprint.This(c.DataStruct)
	err := json.Unmarshal(user.Data, copy)
	return copy, err
}
