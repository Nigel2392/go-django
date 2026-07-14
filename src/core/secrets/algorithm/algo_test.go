package algorithm

import (
	"context"
	"crypto/sha256"
	"testing"
)

func TestEncodeDecodeString(t *testing.T) {
	data := []byte("hello world")
	encoded := EncodeToString(data)
	decoded, err := DecodeString(encoded)
	if err != nil {
		t.Fatalf("DecodeString error: %v", err)
	}
	if string(decoded) != string(data) {
		t.Errorf("expected %s, got %s", string(data), string(decoded))
	}
}

func TestSignatureAlgorithm(t *testing.T) {
	algo := NewSignatureAlgorithm("sha256", sha256.New)
	ctx := context.Background()

	data := []byte("test-data")
	salt := []byte("test-salt")
	key := []byte("test-key")

	if algo.Name() != "sha256" {
		t.Errorf("expected 'sha256', got '%s'", algo.Name())
	}

	signature, err := algo.Signature(ctx, data, salt, key)
	if err != nil {
		t.Fatalf("Signature error: %v", err)
	}

	err = algo.Verify(ctx, data, signature, salt, key)
	if err != nil {
		t.Errorf("Verify error: %v", err)
	}

	err = algo.Verify(ctx, []byte("wrong-data"), signature, salt, key)
	if err == nil {
		t.Errorf("Verify should have failed for wrong data")
	}

	err = algo.Verify(ctx, data, signature, []byte("wrong-salt"), key)
	if err == nil {
		t.Errorf("Verify should have failed for wrong salt")
	}

	err = algo.Verify(ctx, data, signature, salt, []byte("wrong-key"))
	if err == nil {
		t.Errorf("Verify should have failed for wrong key")
	}
}

func FuzzSignatureAlgorithm(f *testing.F) {
	f.Add([]byte("data"), []byte("salt"), []byte("key"))
	f.Add([]byte(""), []byte(""), []byte(""))
	f.Add([]byte("hello"), []byte("world"), []byte("secret"))
	f.Fuzz(func(t *testing.T, data, salt, key []byte) {
		algo := NewSignatureAlgorithm("sha256", sha256.New)
		ctx := context.Background()

		signature, err := algo.Signature(ctx, data, salt, key)
		if err != nil {
			t.Fatalf("Signature error: %v", err)
		}

		err = algo.Verify(ctx, data, signature, salt, key)
		if err != nil {
			t.Fatalf("Verify error: %v", err)
		}
	})
}

func FuzzEncodeDecodeString(f *testing.F) {
	f.Add([]byte("data"))
	f.Add([]byte("hello world"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, data []byte) {
		encoded := EncodeToString(data)
		decoded, err := DecodeString(encoded)
		if err != nil {
			t.Fatalf("DecodeString error: %v", err)
		}
		if string(decoded) != string(data) {
			t.Fatalf("expected %s, got %s", string(data), string(decoded))
		}
	})
}
