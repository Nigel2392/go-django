package securedb_test

import (
	"crypto/sha256"
	"database/sql"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers/securedb"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/secrets"
)

func setupSettings(t *testing.T) {
	if django.Global == nil {
		django.Global = &django.Application{}
	}
	key := "super-secret-test-key"
	hash := sha256.Sum256([]byte(key))
	django.Global.Settings = django.Config(map[string]interface{}{
		secrets.APPVAR_SECRET_KEY: string(hash[:]),
	})
}

func TestEncryptedString_Lifecycle(t *testing.T) {
	setupSettings(t)

	originalText := "super_secret_stripe_token_12345"
	enc := securedb.NewEncryptedString(originalText)

	// 1. Test Value() - Simulating saving to the database
	val, err := enc.Value()
	if err != nil {
		t.Fatalf("Value() returned unexpected error: %v", err)
	}

	strVal, ok := val.(string)
	if !ok {
		t.Fatalf("Value() expected to return a string, got %T", val)
	}

	if strVal == originalText {
		t.Fatalf("Value() returned unencrypted plaintext!")
	}
	if strVal == "" {
		t.Fatalf("Value() returned empty string")
	}

	// 2. Test Scan() - Simulating reading from the database
	loadedEnc := &securedb.EncryptedString{}
	err = loadedEnc.Scan(strVal)
	if err != nil {
		t.Fatalf("Scan() returned unexpected error: %v", err)
	}

	if loadedEnc.Raw != originalText {
		t.Errorf("Scan() failed to restore original text. Got %q, want %q", loadedEnc.Raw, originalText)
	}
}

func TestEncryptedString_ScanTypes(t *testing.T) {
	setupSettings(t)

	originalText := "another_secret_value"
	enc := securedb.NewEncryptedString(originalText)
	val, err := enc.Value()
	if err != nil {
		t.Fatalf("Value() error: %v", err)
	}

	strVal := val.(string)

	tests := []struct {
		name     string
		scanData any
		want     string
		wantErr  bool
	}{
		{
			name:     "Scan string",
			scanData: strVal,
			want:     originalText,
			wantErr:  false,
		},
		{
			name:     "Scan byte slice",
			scanData: []byte(strVal),
			want:     originalText,
			wantErr:  false,
		},
		{
			name:     "Scan nil",
			scanData: nil,
			want:     "",
			wantErr:  false,
		},
		{
			name:     "Scan another EncryptedString",
			scanData: *enc,
			want:     originalText,
			wantErr:  false,
		},
		{
			name:     "Scan invalid type",
			scanData: 12345, // Int is not supported
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &securedb.EncryptedString{}
			err := e.Scan(tt.scanData)

			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && e.Raw != tt.want {
				t.Errorf("Scan() got Raw = %q, want %q", e.Raw, tt.want)
			}
		})
	}
}

func TestEncryptedString_UpdateState(t *testing.T) {
	setupSettings(t)

	enc := securedb.NewEncryptedString("initial_value")

	// Initial save
	val1, err := enc.Value()
	if err != nil {
		t.Fatalf("First Value() error: %v", err)
	}

	// Calling Value() again without modifying Raw should yield the exact same ciphertext
	val2, err := enc.Value()
	if err != nil {
		t.Fatalf("Second Value() error: %v", err)
	}

	if val1.(string) != val2.(string) {
		t.Errorf("Value() changed without modifying Raw state")
	}

	// Modifying Raw should result in a new ciphertext
	enc.Raw = "updated_value"
	val3, err := enc.Value()
	if err != nil {
		t.Fatalf("Third Value() error: %v", err)
	}

	if val1.(string) == val3.(string) {
		t.Errorf("Value() did not update after Raw was modified")
	}
}

func TestEncryptedString_UpdateStateSetValue(t *testing.T) {
	setupSettings(t)

	enc := securedb.NewEncryptedString("initial_value")

	// Initial save
	val1, err := enc.Value()
	if err != nil {
		t.Fatalf("First Value() error: %v", err)
	}

	// Calling Value() again without modifying Raw should yield the exact same ciphertext
	val2, err := enc.Value()
	if err != nil {
		t.Fatalf("Second Value() error: %v", err)
	}

	if val1.(string) != val2.(string) {
		t.Errorf("Value() changed without modifying Raw state")
	}

	// Modifying Raw should result in a new ciphertext
	enc.SetValue("updated_value")
	val3, err := enc.Value()
	if err != nil {
		t.Fatalf("Third Value() error: %v", err)
	}

	if val1.(string) == val3.(string) {
		t.Errorf("Value() did not update after Raw was modified")
	}
}

func TestEncryptedString_IsZero(t *testing.T) {
	tests := []struct {
		name string
		enc  *securedb.EncryptedString
		want bool
	}{
		{
			name: "Nil pointer",
			enc:  nil,
			want: true,
		},
		{
			name: "Empty struct",
			enc:  &securedb.EncryptedString{},
			want: true,
		},
		{
			name: "Has Raw value",
			enc:  securedb.NewEncryptedString("test"),
			want: false,
		},
		{
			name: "Has Ciphertext but no Raw",
			enc:  &securedb.EncryptedString{Ciphertext: []byte("encrypted")},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.enc.IsZero(); got != tt.want {
				t.Errorf("IsZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEncryptedString_StringDoesNotLeak(t *testing.T) {
	enc := securedb.NewEncryptedString("do_not_leak_me")

	strOut := enc.String()

	if strOut == "do_not_leak_me" {
		t.Errorf("String() leaked plaintext value!")
	}

	if strOut != "********" {
		t.Errorf("String() returned unexpected mask: %q", strOut)
	}

	var nilEnc *securedb.EncryptedString
	if nilEnc.String() != "<nil>" {
		t.Errorf("String() on nil returned %q, want <nil>", nilEnc.String())
	}
}

func TestEncryptedString_EmptyAndNilValues(t *testing.T) {
	setupSettings(t)

	enc := securedb.NewEncryptedString("")

	val, err := enc.Value()
	if err != nil {
		t.Fatalf("Value() returned error on empty string: %v", err)
	}
	if val != nil {
		t.Errorf("Value() on empty string expected nil, got %v", val)
	}

	var nilEnc *securedb.EncryptedString
	val, err = nilEnc.Value()
	if err != nil {
		t.Fatalf("Value() returned error on nil pointer: %v", err)
	}
	if val != nil {
		t.Errorf("Value() on nil pointer expected nil, got %v", val)
	}
}

func TestEncryptedString_SQLite_Integration(t *testing.T) {
	setupSettings(t) // Uses the mock setup from the previous tests

	// 1. Open an in-memory SQLite database
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open SQLite database: %v", err)
	}
	defer db.Close()

	// 2. Create a test table with a TEXT column for the secret
	_, err = db.Exec(`
		CREATE TABLE test_secrets (
			id INTEGER PRIMARY KEY,
			secret TEXT
		)
	`)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	originalText := "my_super_secret_db_value"
	enc := securedb.NewEncryptedString(originalText)

	// Insert using the driver.Valuer interface
	_, err = db.Exec("INSERT INTO test_secrets (id, secret) VALUES (1, ?)", enc)
	if err != nil {
		t.Fatalf("Failed to insert record: %v", err)
	}

	// SECURITY CHECK: Read the raw string directly from the database to ensure
	// the plaintext did not leak into the SQLite storage layer.
	var rawDBValue string
	err = db.QueryRow("SELECT secret FROM test_secrets WHERE id = 1").Scan(&rawDBValue)
	if err != nil {
		t.Fatalf("Failed to query raw database value: %v", err)
	}

	if rawDBValue == originalText {
		t.Fatalf("SECURITY FAILURE: Plaintext was stored directly in the database!")
	}
	if rawDBValue == "" {
		t.Fatalf("Database stored an empty string instead of ciphertext")
	}
	t.Logf("Successfully verified database stored ciphertext: %s", rawDBValue)

	// Retrieve using the sql.Scanner interface
	var loadedEnc securedb.EncryptedString
	err = db.QueryRow("SELECT secret FROM test_secrets WHERE id = 1").Scan(&loadedEnc)
	if err != nil {
		t.Fatalf("Failed to scan record into EncryptedString: %v", err)
	}

	// Verify the decrypted value
	if loadedEnc.Raw != originalText {
		t.Errorf("Decrypted value mismatch. Got %q, want %q", loadedEnc.Raw, originalText)
	}

	emptyEnc := securedb.NewEncryptedString("")

	_, err = db.Exec("INSERT INTO test_secrets (id, secret) VALUES (2, ?)", emptyEnc)
	if err != nil {
		t.Fatalf("Failed to insert empty record: %v", err)
	}

	var loadedEmptyEnc securedb.EncryptedString
	err = db.QueryRow("SELECT secret FROM test_secrets WHERE id = 2").Scan(&loadedEmptyEnc)
	if err != nil {
		t.Fatalf("Failed to scan empty record: %v", err)
	}

	if loadedEmptyEnc.Raw != "" {
		t.Errorf("Expected empty string, got %q", loadedEmptyEnc.Raw)
	}

	var rawEmptyDBValue sql.NullString
	err = db.QueryRow("SELECT secret FROM test_secrets WHERE id = 2").Scan(&rawEmptyDBValue)
	if err != nil {
		t.Fatalf("Failed to query raw empty database value: %v", err)
	}

	if rawEmptyDBValue.Valid && rawEmptyDBValue.String != "" {
		t.Errorf("Expected database to store NULL or empty string for empty struct, got %q", rawEmptyDBValue.String)
	}
}
