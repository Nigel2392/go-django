package algorithm

import (
	"crypto/sha256"
	"testing"
)

func TestRegistry(t *testing.T) {
	algo, ok := GetSignatureAlgo("sha256")
	if !ok {
		t.Fatalf("expected sha256 to be registered by init()")
	}
	if algo.Name() != "sha256" {
		t.Errorf("expected 'sha256', got '%s'", algo.Name())
	}

	dummy := NewSignatureAlgorithm("dummy", sha256.New)
	RegisterSignatureAlgos(dummy)

	algo, ok = GetSignatureAlgo("dummy")
	if !ok {
		t.Fatalf("expected dummy to be registered")
	}
	if algo.Name() != "dummy" {
		t.Errorf("expected 'dummy', got '%s'", algo.Name())
	}

	_, ok = GetSignatureAlgo("non-existent")
	if ok {
		t.Fatalf("expected non-existent to not be registered")
	}
}
