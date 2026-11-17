package securedb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/base64"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/src/core/secrets"
)

var _ driver.Valuer = &EncryptedString{}
var _ sql.Scanner = &EncryptedString{}

// EncryptedString is a database-aware type that automatically encrypts
// its contents when saved to the database, and decrypts when loaded.
type EncryptedString struct {
	// Raw is the unencrypted plaintext value.
	Raw string `json:"raw"`

	// raw holds the state of Raw the last time it was encrypted/decrypted
	// to track changes.
	raw string `json:"-"`

	// Ciphertext is the raw encrypted byte array.
	Ciphertext []byte `json:"-"`
}

// NewEncryptedString creates a new EncryptedString ready to be saved.
func NewEncryptedString(raw string) *EncryptedString {
	return &EncryptedString{
		Raw: raw,
		raw: raw,
	}
}

// DBType tells the migrator to use a String column (VARCHAR/TEXT).
func (e *EncryptedString) DBType() dbtype.Type {
	return dbtype.String
}

// IsZero checks if the field is empty.
func (e *EncryptedString) IsZero() bool {
	if e == nil {
		return true
	}
	if len(e.Ciphertext) != 0 {
		return false
	}
	if e.Raw != "" {
		return false
	}
	return true
}

// String prevents the plaintext or ciphertext from leaking into logs by accident.
func (e *EncryptedString) String() string {
	if e.IsZero() {
		return "<nil>"
	}
	return "********" // Masked for security
}

func (e *EncryptedString) SetValue(raw string) (err error) {
	encrypted, err := secrets.Encrypt(context.Background(), []byte(raw))
	if err != nil {
		return ErrSecureValue.Wrapf("failed to encrypt value: %v", err)
	}

	e.Raw = raw
	e.raw = raw
	e.Ciphertext = encrypted
	return nil
}

// Scan reads the base64 encoded ciphertext from the database and decrypts it.
func (e *EncryptedString) Scan(value any) error {
	if value == nil {
		e.Raw = ""
		e.raw = ""
		e.Ciphertext = nil
		return nil
	}

	var ciphertext []byte

	switch v := value.(type) {
	case string:
		// Decode base64 string from the database
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			// Fallback just in case it was saved as raw string/bytes previously
			ciphertext = []byte(v)
		} else {
			ciphertext = decoded
		}
	case []byte:
		// If the driver returns bytes directly, try to decode as base64 first
		decoded, err := base64.StdEncoding.DecodeString(string(v))
		if err != nil {
			ciphertext = v
		} else {
			ciphertext = decoded
		}
	case EncryptedString:
		e.Ciphertext = v.Ciphertext
		e.Raw = v.Raw
		e.raw = v.raw
		return nil
	default:
		return ErrSecureValue.Wrapf(
			"cannot scan value %T into EncryptedString: %v", value, value,
		)
	}

	e.Ciphertext = ciphertext

	ctx := context.Background()
	decrypted, err := secrets.Decrypt(ctx, ciphertext)
	if err != nil {
		return ErrSecureValue.Wrapf("failed to decrypt value: %v", err)
	}

	e.Raw = string(decrypted)
	e.raw = e.Raw

	return nil
}

// Value encrypts the raw string and returns it as a base64 encoded string for the database.
func (e *EncryptedString) Value() (driver.Value, error) {
	if e == nil {
		return nil, nil
	}

	// Only encrypt if the raw text has changed, or if we have text but no ciphertext yet.
	if e.Raw != "" && (e.Raw != e.raw || len(e.Ciphertext) == 0) {
		ctx := context.Background()

		encrypted, err := secrets.Encrypt(ctx, []byte(e.Raw))
		if err != nil {
			return nil, ErrSecureValue.Wrapf("failed to encrypt value: %v", err)
		}

		e.Ciphertext = encrypted
		e.raw = e.Raw
		goto returnBase64

	} else if len(e.Ciphertext) == 0 && e.Raw == "" {
		return nil, nil
	}

returnBase64:
	// Return as base64 string to safely store in VARCHAR/TEXT fields
	return base64.StdEncoding.EncodeToString(e.Ciphertext), nil
}
