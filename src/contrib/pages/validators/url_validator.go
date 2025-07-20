package validators

import (
	"fmt"
	"net/url"
)

func ValidateSettingsURL(value interface{}) error {
	if str, ok := value.(string); ok {
		if str == "" {
			return nil // Empty string is considered valid
		}

		if str == "localhost" || str == "127.0.0.1" {
			return nil
		}

		var u, err = url.Parse(str)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		if u.Host == "" {
			return fmt.Errorf("invalid URL: missing host in %q", str)
		}

		if u.Port() != "" {
			return fmt.Errorf("invalid URL: port is not allowed in %q", str)
		}
		return nil
	}

	return fmt.Errorf("expected a string, got %T", value)
}
