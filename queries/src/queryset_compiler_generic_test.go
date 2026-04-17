package queries

import (
	"testing"
	"time"
)

func TestGenericQueryBuilderPrepareValueNoop(t *testing.T) {
	g := &genericQueryBuilder{}

	v := time.Now().UTC()
	got := g.PrepareValue(nil, v)
	if got != v {
		t.Fatalf("expected value to be unchanged, got %v", got)
	}
}

func TestMariaDBQueryBuilderPrepareValueZeroTimeToNil(t *testing.T) {
	g := &mariaDBQueryBuilder{}

	got := g.PrepareValue(nil, time.Time{})
	if got != nil {
		t.Fatalf("expected nil for zero time.Time, got %T (%v)", got, got)
	}
}

func TestMySQLQueryBuilderPrepareValueZeroTimeToNil(t *testing.T) {
	g := &mysqlQueryBuilder{}

	got := g.PrepareValue(nil, time.Time{})
	if got != nil {
		t.Fatalf("expected nil for zero time.Time, got %T (%v)", got, got)
	}
}

func TestGenericQueryBuilderThisDispatchesPrepareValueOverride(t *testing.T) {
	base := &genericQueryBuilder{}
	mysqlBuilder := &mysqlQueryBuilder{
		mariaDBQueryBuilder: &mariaDBQueryBuilder{
			genericQueryBuilder: base,
		},
	}
	base.self = mysqlBuilder

	got := base.This().PrepareValue(nil, time.Time{})
	if got != nil {
		t.Fatalf("expected nil from mysql PrepareValue override, got %T (%v)", got, got)
	}
}
