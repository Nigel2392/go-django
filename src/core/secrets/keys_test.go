package secrets

import (
	"context"
	"testing"

	django "github.com/Nigel2392/go-django/src"
)

func setupTestSettings() {
	if django.Global == nil {
		django.Global = &django.Application{}
	}
	django.Global.Settings = django.Config(map[string]interface{}{
		APPVAR_SECRET_KEY: "super-secret-test-key",
	})
}

func TestSecretKeyMethods(t *testing.T) {
	var key = SecretKey("test-key")
	if key.String() != "test-key" {
		t.Errorf("expected string 'test-key', got '%s'", key.String())
	}
	if string(key.Bytes()) != "test-key" {
		t.Errorf("expected bytes 'test-key', got '%s'", string(key.Bytes()))
	}
	if key.IsZero() {
		t.Errorf("expected non-zero key, got zero")
	}

	var emptyKey SecretKey
	if !emptyKey.IsZero() {
		t.Errorf("expected zero key, got non-zero")
	}
}

func TestSecretKeySignAndUnsign(t *testing.T) {
	setupTestSettings()

	var key = SECRET_KEY()
	var ctx = context.Background()
	var data = []byte("hello world")

	signed, err := key.Sign(ctx, data)
	if err != nil {
		t.Fatalf("Sign error: %v", err)
	}

	unsigned, err := key.Unsign(ctx, signed)
	if err != nil {
		t.Fatalf("Unsign error: %v", err)
	}

	if string(unsigned) != string(data) {
		t.Errorf("expected '%s', got '%s'", string(data), string(unsigned))
	}
}

func TestSecretKeySignObjectAndUnsignObject(t *testing.T) {
	setupTestSettings()

	var key = SECRET_KEY()
	var ctx = context.Background()

	type TestObj struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	var obj = TestObj{Name: "Alice", Age: 30}

	signed, err := key.SignObject(ctx, obj)
	if err != nil {
		t.Fatalf("SignObject error: %v", err)
	}

	var decoded TestObj
	err = key.UnsignObject(ctx, signed, &decoded)
	if err != nil {
		t.Fatalf("UnsignObject error: %v", err)
	}

	if decoded.Name != obj.Name || decoded.Age != obj.Age {
		t.Errorf("expected %+v, got %+v", obj, decoded)
	}
}

func FuzzSecretKeySignAndUnsign(f *testing.F) {
	setupTestSettings()

	f.Add([]byte("data"))
	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	
	f.Fuzz(func(t *testing.T, data []byte) {
		var key = SECRET_KEY()
		var ctx = context.Background()

		signed, err := key.Sign(ctx, data)
		if err != nil {
			t.Fatalf("Sign error: %v", err)
		}

		unsigned, err := key.Unsign(ctx, signed)
		if err != nil {
			t.Fatalf("Unsign error: %v", err)
		}

		if string(unsigned) != string(data) {
			t.Fatalf("expected '%s', got '%s'", string(data), string(unsigned))
		}
	})
}
