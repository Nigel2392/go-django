package bitch_test

import (
	"testing"

	"github.com/Nigel2392/go-django/internal/bitch"
)

// Dummy constants for testing
const (
	FlagNone bitch.Flag = 0
	FlagA    bitch.Flag = 1 << iota // 1
	FlagB                           // 2
	FlagC                           // 4
	FlagD                           // 8
)

func TestFlag_Is_SinglePresent(t *testing.T) {
	f := FlagA | FlagB
	if !f.Is(FlagA) {
		t.Errorf("expected FlagA to be present")
	}
}

func TestFlag_Is_SingleMissing(t *testing.T) {
	f := FlagA
	if f.Is(FlagB) {
		t.Errorf("expected FlagB to be missing")
	}
}

func TestFlag_Is_MultiplePresent(t *testing.T) {
	f := FlagA | FlagB | FlagC
	if !f.Is(FlagA | FlagC) {
		t.Errorf("expected both FlagA and FlagC to be present")
	}
}

func TestFlag_Is_MultiplePartial(t *testing.T) {
	f := FlagA | FlagB
	if f.Is(FlagA | FlagC) {
		t.Errorf("Is should return false if not ALL requested flags are present")
	}
}

func TestFlag_Isnt_SingleMissing(t *testing.T) {
	f := FlagA | FlagB
	if !f.Isnt(FlagC) {
		t.Errorf("expected Isnt to be true for FlagC")
	}
}

func TestFlag_Isnt_SinglePresent(t *testing.T) {
	f := FlagA | FlagB
	if f.Isnt(FlagA) {
		t.Errorf("expected Isnt to be false because FlagA is present")
	}
}

func TestFlag_Isnt_Multiple(t *testing.T) {
	f := FlagA | FlagB
	if !f.Isnt(FlagC | FlagD) {
		t.Errorf("expected Isnt to be true for FlagC and FlagD")
	}
}

func TestFlag_Set_TurnOn(t *testing.T) {
	f := FlagA
	res := f.Set(FlagB, true)
	if !res.Is(FlagA | FlagB) {
		t.Errorf("expected Set to add FlagB")
	}
}

func TestFlag_Set_TurnOff(t *testing.T) {
	f := FlagA | FlagB
	res := f.Set(FlagB, false)
	if res.Is(FlagB) {
		t.Errorf("expected Set to remove FlagB")
	}
}

func TestFlag_Set_PointerRetrievalAndImmutability(t *testing.T) {
	f := FlagA
	ptr := &f

	// Calling the pointer receiver method
	res := ptr.Set(FlagB, true)

	if !res.Is(FlagB) {
		t.Errorf("expected returned flag to have FlagB")
	}

	// Verify that the pointer itself was not mutated
	// (because Set returns a new Flag rather than assigning to *o)
	if f.Is(FlagB) {
		t.Errorf("expected original flag to remain unchanged (no mutation)")
	}
}
