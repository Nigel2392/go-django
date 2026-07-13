package expr_test

import (
	"strings"
	"testing"
)

// SQL Generation 1
func TestInfoFormatSQLField(t *testing.T) {
	info := getTestInfo()
	if info.QuoteIdentifier == nil {
		t.Fatalf("QuoteIdentifier is nil")
	}
	formatted := info.QuoteIdentifier("name")
	if !strings.Contains(formatted, fixSQL(info, "`name`")) {
		t.Errorf("Unexpected quote output: %s", formatted)
	}
}

// SQL Generation 2
func TestInfoFormatSQLValue(t *testing.T) {
	info := getTestInfo()
	if info.Lookups.OperatorsRHS == nil {
		t.Log("OperatorsRHS nil")
	} else if fmtStr, ok := info.Lookups.OperatorsRHS["iexact"]; ok {
		if fmtStr == "" {
			t.Errorf("Unexpected empty format string")
		}
	}
}

// Happy Path 1
func TestInfoModel(t *testing.T) {
	info := getTestInfo()
	if info.Model == nil {
		t.Errorf("Expected Model to be set on info")
	}
}

// Happy Path 2
func TestInfoDriver(t *testing.T) {
	info := getTestInfo()
	if info.Driver == nil {
		t.Errorf("Expected Driver to be set on info")
	}
}
