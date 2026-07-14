package safety

import (
	"context"
	"testing"
)

func TestIsSecretField_Password(t *testing.T) {
	if !IsSecretField(context.Background(), "password") {
		t.Errorf("Expected 'password' to be secret")
	}
}

func TestIsSecretField_Token(t *testing.T) {
	if !IsSecretField(context.Background(), "access_token") {
		t.Errorf("Expected 'access_token' to be secret")
	}
}

func TestIsSecretField_Pin(t *testing.T) {
	if !IsSecretField(context.Background(), "pin") {
		t.Errorf("Expected 'pin' to be secret")
	}
}

func TestIsSecretField_NonSecret(t *testing.T) {
	if IsSecretField(context.Background(), "username") {
		t.Errorf("Expected 'username' to not be secret")
	}
}

func TestIsSecretField_MixedCase(t *testing.T) {
	if !IsSecretField(context.Background(), "PassWord") {
		t.Errorf("Expected 'PassWord' to be secret")
	}
}

func TestIsSecretField_WithSymbols(t *testing.T) {
	if !IsSecretField(context.Background(), "db_password") {
		t.Errorf("Expected 'db_password' to be secret")
	}
}

func TestIsSecretField_AwsAccessKey(t *testing.T) {
	if !IsSecretField(context.Background(), "AWS_ACCESS_KEY_ID") {
		t.Errorf("Expected 'AWS_ACCESS_KEY_ID' to be secret")
	}
}

func TestIsSecretField_Cookie(t *testing.T) {
	if !IsSecretField(context.Background(), "cookie") {
		t.Errorf("Expected 'cookie' to be secret")
	}
}

func TestIsSecretField_CsrfToken(t *testing.T) {
	if !IsSecretField(context.Background(), "csrf_token") {
		t.Errorf("Expected 'csrf_token' to be secret")
	}
}

func TestRegisterChecker(t *testing.T) {
	ctx := context.Background()
	
	if IsSecretField(ctx, "customsecret") {
		t.Fatalf("expected 'customsecret' to be false initially")
	}

	RegisterChecker(func(ctx context.Context, fieldName string) bool {
		return fieldName == "customsecret"
	})

	if !IsSecretField(ctx, "customsecret") {
		t.Errorf("expected 'customsecret' to be true after registration")
	}
}

func FuzzIsSecretField(f *testing.F) {
	f.Add("password")
	f.Add("PASSWORD")
	f.Add("pass_word")
	f.Add("username")
	f.Add("email")
	f.Add("token")
	f.Add("")
	f.Add("123456")

	f.Fuzz(func(t *testing.T, fieldName string) {
		ctx := context.Background()
		// Fuzzing to ensure no panics and standard behaviour
		_ = IsSecretField(ctx, fieldName)
	})
}
