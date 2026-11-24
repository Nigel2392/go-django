package translations

import (
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/goldcrest"
	"golang.org/x/text/language"
)

type catalogTestFS struct {
	fstest.MapFS
	openErr error
	readErr error
	opens   int
	closes  int
}

func (f *catalogTestFS) Open(name string) (fs.File, error) {
	f.opens++
	if f.openErr != nil {
		return nil, f.openErr
	}
	file, err := f.MapFS.Open(name)
	if err != nil {
		return nil, err
	}
	return &catalogTestFile{File: file, fsys: f}, nil
}

type catalogTestFile struct {
	fs.File
	fsys *catalogTestFS
}

func (f *catalogTestFile) Read(p []byte) (int, error) {
	if f.fsys.readErr != nil {
		return 0, f.fsys.readErr
	}
	return f.File.Read(p)
}

func (f *catalogTestFile) Close() error {
	f.fsys.closes++
	return f.File.Close()
}

func initTestCatalogApp(t *testing.T, catalogs ...fs.FS) *TranslationsAppConfig {
	t.Helper()
	setTestDjango(t, django.Config(nil))
	setTestTranslationsApp(t, nil)
	originalRegistry := goldcrest.DefaultRegistry
	t.Cleanup(func() { goldcrest.DefaultRegistry = originalRegistry })
	goldcrest.DefaultRegistry = make(goldcrest.HookRegistry)
	goldcrest.Register(TranslationFilesystemHookName, 0, TranslationFilesystemHook(func(django.Settings) []fs.FS {
		return catalogs
	}))
	return NewAppConfig().(*TranslationsAppConfig)
}

func TestCatalogFilesystemErrorsAndCleanup(t *testing.T) {
	for _, tc := range []struct {
		name       string
		files      fstest.MapFS
		openErr    error
		readErr    error
		wantErr    string
		wantOpens  int
		wantCloses int
	}{
		{name: "missing catalog"},
		{name: "directory in place of catalog", files: fstest.MapFS{translationsFile: {Mode: fs.ModeDir}}},
		{name: "open failure", files: fstest.MapFS{translationsFile: {}}, openErr: fs.ErrPermission,
			wantErr: "failed to open translations.yml", wantOpens: 1},
		{name: "read failure", files: fstest.MapFS{translationsFile: {}}, readErr: errors.New("catalog read failed"),
			wantErr: "catalog read failed", wantOpens: 1, wantCloses: 1},
		{name: "decode failure", files: fstest.MapFS{translationsFile: {Data: []byte("locales: {}\n---\ntext: [invalid]")}},
			wantErr: "error decoding Translation", wantOpens: 1, wantCloses: 1},
		{name: "successful decode", files: fstest.MapFS{translationsFile: {Data: []byte("locales: {}\n")}},
			wantOpens: 1, wantCloses: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Chdir(t.TempDir())
			catalog := &catalogTestFS{MapFS: tc.files, openErr: tc.openErr, readErr: tc.readErr}
			app := initTestCatalogApp(t, catalog)
			err := app.Initialize(django.Global.Settings)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("Initialize() = %v", err)
				}
			} else {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("Initialize() = %v, want error containing %q", err, tc.wantErr)
				}
				if app.translationHeader != nil || len(app.translations) > 0 {
					t.Error("initialization published a partial catalog")
				}
			}
			if catalog.opens != tc.wantOpens || catalog.closes != tc.wantCloses {
				t.Errorf("opens and closes = (%d, %d), want (%d, %d)", catalog.opens, catalog.closes, tc.wantOpens, tc.wantCloses)
			}
		})
	}
}

func TestCatalogLayersRetainLocalizedMetadata(t *testing.T) {
	t.Chdir(t.TempDir())
	const projectCatalog = `generator: project
locales:
  es: {nplural: 2, rule: "n != 1", short_time_format: "%d/%m/%Y"}
  fr: {nplural: 2, rule: "n > 1", short_time_format: "%d.%m.%Y"}
---
text: file
plural: files
locales:
  es: {0: archivo, 1: archivos}
  fr: {0: fichier, 1: fichiers}
`
	if err := os.WriteFile(translationsFile, []byte(projectCatalog), 0600); err != nil {
		t.Fatal(err)
	}
	const pluginCatalog = `generator: plugin
locales:
  fr: {nplural: 2, rule: "n != 1", short_time_format: "%d-%m-%Y"}
  ja: {nplural: 1, rule: "0", short_time_format: "%Y年%m月%d日"}
---
text: file
plural: files
locales:
  fr: {0: document, 1: documents}
  ja: ファイル
`
	app := initTestCatalogApp(t,
		fstest.MapFS{translationsFile: {Data: []byte(pluginCatalog)}},
		fstest.MapFS{translationsFile: {Data: []byte("generator: last plugin\n")}},
	)
	if err := app.Initialize(django.Global.Settings); err != nil {
		t.Fatal(err)
	}
	header := app.translationHeader.hdr
	if header.Generator != "last plugin" {
		t.Errorf("generator = %q, want last plugin", header.Generator)
	}
	if got, want := header.Locales.Keys(), []string{"nl", "es", "fr", "ja"}; !slices.Equal(got, want) {
		t.Errorf("catalog locales = %v, want %v", got, want)
	}
	translator := NewTranslator(header, app.translations, nil)
	for _, tc := range []struct {
		tag        language.Tag
		wantPlural string
		wantFormat string
	}{
		{language.Spanish, "archivos", "%d/%m/%Y"},
		{language.French, "documents", "%d-%m-%Y"},
		{language.Japanese, "ファイル", "%Y年%m月%d日"},
	} {
		t.Run(tc.tag.String(), func(t *testing.T) {
			ctx := trans.ContextWithLocale(t.Context(), tc.tag)
			if got := translator.Pluralize(ctx, "file", "files", 0); got != tc.wantPlural {
				t.Errorf("Pluralize(0) = %q, want %q", got, tc.wantPlural)
			}
			if got := translator.TimeFormat(ctx, true); got != tc.wantFormat {
				t.Errorf("TimeFormat() = %q, want %q", got, tc.wantFormat)
			}
		})
	}
}
