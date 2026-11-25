//go:build cgo

package translations

import (
	"flag"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/migrator/sql/sqlite"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/command"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/djangoapps"
	"github.com/elliotchance/orderedmap/v2"
	"golang.org/x/text/language"
)

func TestSyncLocalesCommandRegistersAndPersists(t *testing.T) {
	db, cfg := setupLocaleDatabase(t)
	if _, err := db.ExecContext(t.Context(), "INSERT INTO translations_locale (language_id) VALUES (?), (?)", "de", "nl"); err != nil {
		t.Fatal(err)
	}

	registry := command.NewRegistry("translations-test", flag.PanicOnError)
	for _, cmd := range cfg.Commands() {
		registry.Register(cmd)
	}
	django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []string{"nl", " EN-us ", "en-US", "nl", "zh-hant-tw"})
	args := []string{"sync-locales", "-locales=fr", "-locales=nl", "-locales=fr"}
	for range 2 {
		if err := registry.ExecCommand(args); err != nil {
			t.Fatal(err)
		}
		assertStoredLocales(t, "de", "en-US", "fr", "nl", "zh-Hant-TW")
	}

	django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []language.Tag{language.Portuguese, language.MustParse("pt-PT"), language.Und})
	if err := registry.ExecCommand([]string{"sync-locales"}); err != nil {
		t.Fatal(err)
	}
	assertStoredLocales(t, "de", "en-US", "fr", "nl", "pt", "pt-PT", "und", "zh-Hant-TW")
}

func TestSyncLocalesRejectsInputWithoutWriting(t *testing.T) {
	setupLocaleDatabase(t)
	django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []string{"nl"})
	if err := executeSyncLocalesCommand(); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name    string
		setting any
		args    []string
		wantErr string
	}{
		{"invalid setting type", "en-US", nil, "must be []string or []language.Tag"},
		{"invalid configured tag", []string{"fr", "nl!!"}, nil, "invalid locale \"nl!!\""},
		{"empty configured tag", []string{"fr", ""}, nil, "invalid locale \"\""},
		{"whitespace-only tag", []string{"fr", " "}, nil, "invalid locale \" \""},
		{"invalid flag after valid input", []string{"fr"}, []string{"-locales=de", "-locales=??"}, "invalid locale \"??\""},
		{"unexpected positional argument", []string{"fr"}, []string{"de"}, "does not accept positional arguments"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, tc.setting)
			err := executeSyncLocalesCommand(tc.args...)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("sync-locales error = %v, want %q", err, tc.wantErr)
			}
			assertStoredLocales(t, "nl")
		})
	}
}

func TestSyncLocalesRollsBackOnWriteFailure(t *testing.T) {
	db, _ := setupLocaleDatabase(t)
	const trigger = `CREATE TRIGGER reject_locale BEFORE INSERT ON translations_locale
WHEN NEW.language_id = 'fr'
BEGIN SELECT RAISE(ABORT, 'fixture rejects fr'); END;`
	if _, err := db.ExecContext(t.Context(), trigger); err != nil {
		t.Fatal(err)
	}

	django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []string{"de", "fr"})
	err := executeSyncLocalesCommand()
	if err == nil || !strings.Contains(err.Error(), "could not save locales") || !strings.Contains(err.Error(), "fixture rejects fr") {
		t.Fatalf("sync-locales error = %v, want trigger failure", err)
	}
	assertStoredLocales(t)
	if _, err := db.ExecContext(t.Context(), "DROP TRIGGER reject_locale"); err != nil {
		t.Fatal(err)
	}
	if err := executeSyncLocalesCommand(); err != nil {
		t.Fatal(err)
	}
	assertStoredLocales(t, "de", "fr")
}

func TestSyncLocalesReportsMissingTable(t *testing.T) {
	db, _ := setupLocaleDatabase(t)
	if _, err := db.ExecContext(t.Context(), "DROP TABLE translations_locale"); err != nil {
		t.Fatal(err)
	}
	django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []string{"en"})
	if err := executeSyncLocalesCommand(); err == nil || !strings.Contains(err.Error(), "could not read existing locales") {
		t.Errorf("sync-locales error = %v, want missing table", err)
	}
}

func TestSyncLocalesReportsClosedDatabase(t *testing.T) {
	db, _ := setupLocaleDatabase(t)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	django.Global.Settings.Set(trans.APPVAR_TRANSLATIONS_LOCALES, []string{"en"})
	if err := executeSyncLocalesCommand(); err == nil || !strings.Contains(err.Error(), "could not start a transaction") {
		t.Errorf("sync-locales error = %v, want transaction startup error", err)
	}
}

func setupLocaleDatabase(t *testing.T) (drivers.Database, *TranslationsAppConfig) {
	t.Helper()
	db, err := drivers.Open(t.Context(), "sqlite3", filepath.ToSlash(filepath.Join(t.TempDir(), "locales.sqlite")))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	setTestDjango(t, django.Config(map[string]any{django.APPVAR_DATABASE: db}))
	setTestTranslationsApp(t, nil)
	cfg := NewAppConfig().(*TranslationsAppConfig)
	django.Global.Apps = orderedmap.NewOrderedMap[string, django.AppConfig]()
	django.Global.Apps.Set(cfg.Name(), cfg)
	registered := &Locale{}
	if models := cfg.Models(); len(models) != 1 {
		t.Fatalf("translations app models = %v, want Locale", models)
	} else if model, ok := models[0].(*Locale); !ok {
		t.Fatalf("translations app model = %T, want *Locale", models[0])
	} else {
		registered = model
	}
	djangoapps.RegisterModel(registered, cfg)
	contenttypes.Register(&contenttypes.ContentTypeDefinition{ContentObject: registered})
	attrs.RegisterModel(registered)
	engine := migrator.NewMigrationEngine(t.TempDir(), sqlite.NewSQLiteSchemaEditor(db), migrator.EngineOptionApps(cfg.Name()))
	if err := engine.Migrate(t.Context(), cfg.Name()); err != nil {
		t.Fatalf("migrate translations app: %v", err)
	}
	if drift, err := engine.NeedsToMakeMigrations(t.Context(), cfg.Name()); err != nil || len(drift) != 0 {
		t.Fatalf("Locale model differs from its migration: %v, %v", drift, err)
	}
	return db, cfg
}

func assertStoredLocales(t *testing.T, want ...string) {
	t.Helper()
	rows, err := queries.GetQuerySetWithContext(t.Context(), &Locale{}).All()
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.Object.LanguageTag.String())
	}
	slices.Sort(got)
	if !slices.Equal(got, want) {
		t.Errorf("locales in database = %v, want %v", got, want)
	}
}
