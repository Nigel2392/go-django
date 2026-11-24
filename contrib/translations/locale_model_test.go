package translations

import (
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"golang.org/x/text/language"
)

var (
	_ driver.Valuer = LanguageTag{}
	_ sql.Scanner   = (*LanguageTag)(nil)
)

func TestLanguageTagDatabaseRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name   string
		source any
		want   string
	}{
		{"base language", "nl", "nl"},
		{"string region", "en-us", "en-US"},
		{"byte region", []byte("pt-pt"), "pt-PT"},
		{"script and region", "zh-hant-tw", "zh-Hant-TW"},
		{"variant", "sl-rozaj", "sl-rozaj"},
		{"Unicode extension", "en-US-u-ca-gregory", "en-US-u-ca-gregory"},
		{"private use", "x-fixture", "x-fixture"},
		{"undetermined language", "und", "und"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tag := LanguageTag(language.Dutch)
			if err := tag.Scan(tc.source); err != nil {
				t.Fatal(err)
			}
			if tag.String() != tc.want {
				t.Errorf("scanned tag = %q, want %q", tag.String(), tc.want)
			}
			value, err := tag.Value()
			if err != nil || value != tc.want {
				t.Errorf("stored tag = (%v, %v), want (%q, nil)", value, err, tc.want)
			}
		})
	}
}

func TestLanguageTagScanKeepsValueOnFailure(t *testing.T) {
	for _, tc := range []struct {
		name    string
		source  any
		wantErr string
	}{
		{"null is ignored", nil, ""},
		{"non-text driver value", int64(37), "int64 is not of type string or []byte"},
		{"invalid text", "en!!!", "invalid language tag"},
		{"invalid bytes", []byte("zh---"), "invalid language tag"},
		{"empty string", "", "invalid language tag"},
		{"empty bytes", []byte{}, "invalid language tag"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tag := LanguageTag(language.Dutch)
			err := tag.Scan(tc.source)
			if tc.wantErr == "" && err != nil || tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("Scan error = %v, want %q", err, tc.wantErr)
			}
			if tag.String() != "nl" {
				t.Errorf("scan changed tag to %q, want nl", tag.String())
			}
		})
	}
}

func TestLanguageTagDatabaseLength(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value string
		ok    bool
	}{
		{"at column width", "x-" + strings.Repeat("private-", 31) + "parts", true},
		{"over column width", "x-" + strings.Repeat("private-", 31) + "partss", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tag := LanguageTag(language.MustParse(tc.value))
			value, err := tag.Value()
			if tc.ok && (err != nil || value != tc.value) {
				t.Errorf("Value = (%q, %v), want %q", value, err, tc.value)
			}
			if !tc.ok && (err == nil || value != nil) {
				t.Errorf("Value = (%q, %v), want length error", value, err)
			}
		})
	}
}

func TestLocaleModelPrimaryKeyDialects(t *testing.T) {
	model := &Locale{}
	attrs.RegisterModel(model)
	table := migrator.NewModelTable(model)
	cols := table.Columns()
	if len(cols) != 1 {
		t.Fatalf("columns = %v, want one language ID column", cols)
	}
	column := cols[0]
	if column.Name != "LanguageTag" || column.Column != "language_id" || !column.Primary ||
		column.Auto || column.Nullable || column.DBType() != dbtype.String {
		t.Fatalf("unexpected locale primary key: %+v", column)
	}
	for _, tc := range []struct {
		name     string
		driver   driver.Driver
		wantType string
	}{
		{"SQLite", &drivers.DriverSQLite{}, "TEXT"},
		{"MySQL", &drivers.DriverMySQL{}, "VARCHAR(255)"},
		{"MariaDB", &drivers.DriverMariaDB{}, "VARCHAR(255)"},
		{"PostgreSQL", &drivers.DriverPostgres{}, "VARCHAR(255)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := strings.ToUpper(migrator.GetFieldType(tc.driver, column)); got != tc.wantType {
				t.Errorf("%s language ID column type = %q, want %q", tc.name, got, tc.wantType)
			}
		})
	}
}
