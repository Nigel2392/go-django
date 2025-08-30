package safety

import (
	"context"
	"regexp"
	"strings"
)

type SecretChecker func(ctx context.Context, fieldName string) (isSecret bool)

var checkers = make([]SecretChecker, 0)

func RegisterChecker(checker SecretChecker) {
	checkers = append(checkers, checker)
}

func IsSecretField(ctx context.Context, fieldName string) bool {
	for _, checker := range checkers {
		if checker(ctx, fieldName) {
			return true
		}
	}
	return false
}

var fieldname_replace_regex = regexp.MustCompile(`[^a-zA-Z0-9.]`)

var secretFields = map[string]struct{}{
	// Authentication
	"password":        {},
	"passwordconfirm": {},
	"passwordhash":    {},
	"passwd":          {},
	"passcode":        {},
	"pin":             {},
	"otp":             {},

	// Tokens & keys
	"token":        {},
	"accesstoken":  {},
	"refreshtoken": {},
	"idtoken":      {},
	"apitoken":     {},
	"auth":         {},
	"authkey":      {},
	"authcode":     {},
	"authtoken":    {},
	"secret":       {},
	"secretkey":    {},
	"apikey":       {},
	"appkey":       {},
	"privatekey":   {},
	"publickey":    {}, // sometimes still sensitive
	"sshkey":       {},
	"cert":         {},
	"certificate":  {},
	"keystore":     {},
	"truststore":   {},

	// Database credentials
	"dbpassword": {},
	"dbpasswd":   {},
	"dbuser":     {}, // username may be less sensitive, but can still be protected
	"dbusername": {},
	"dbtoken":    {},

	// Cloud/service credentials
	"awsaccesskeyid":     {},
	"awssecretaccesskey": {},
	"awstoken":           {},
	"azureclientsecret":  {},
	"azuretenantid":      {},
	"gcloudkey":          {},
	"gcpkey":             {},
	"serviceaccountkey":  {},
	"serviceaccountjson": {},
	"slacktoken":         {},
	"slackapikey":        {},
	"pat":                {}, // personal access token
	"ghpat":              {}, // GitHub PAT
	"gitlabtoken":        {},

	// Encryption keys
	"encryptionkey": {},
	"cryptokey":     {},
	"jwtsecret":     {},
	"hmacsecret":    {},
	"salt":          {},
	"pepper":        {},

	// Misc
	"sessionid":           {},
	"sessiontoken":        {},
	"cookie":              {},
	"csrf":                {},
	"csrftoken":           {},
	"csrfmiddlewaretoken": {},
}

func init() {
	RegisterChecker(func(ctx context.Context, fieldName string) (isSecret bool) {
		fieldName = fieldname_replace_regex.ReplaceAllString(fieldName, "")
		fieldName = strings.ToLower(fieldName)
		_, isSecret = secretFields[fieldName]
		return isSecret
	})
}
