package contenttypes_test

import (
	"os"
	"reflect"
	"testing"

	testsql "github.com/Nigel2392/go-django/queries/src/migrator/sql/test_sql"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/mux/middleware/authentication"
)

type User interface {
	attrs.Definer
	authentication.User
}

type Letter struct {
	ID     int64
	Text   string
	Author User
}

func (l *Letter) FieldDefs() attrs.Definitions {
	return attrs.Define(l,
		attrs.NewField(l, "ID", &attrs.FieldConfig{
			Primary: true,
		}),
		attrs.NewField(l, "Text", &attrs.FieldConfig{
			Blank:     true,
			MaxLength: 1024,
		}),
		attrs.NewField(l, "Author", &attrs.FieldConfig{
			RelOneToOne: attrs.RelatedDeferred(
				attrs.RelOneToOne,
				lazyUserModel.LoadString(),
				"", nil,
			),
		}),
	)
}

var (
	lazyUserModel = contenttypes.NewLazyRegistry(func(ctd *contenttypes.ContentTypeDefinition) bool {
		_, ok := ctd.ContentObject.(authentication.User)
		return ok
	})
	lazyBlogPage = contenttypes.NewLazyRegistry(func(ctd *contenttypes.ContentTypeDefinition) bool {
		_, ok := ctd.ContentObject.(*testsql.BlogPost)
		return ok
	})
)

func TestMain(m *testing.M) {

	var app = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DEBUG: true,
		}),
		django.Flag(
			django.FlagSkipChecks,
			django.FlagSkipCmds,
			django.FlagSkipDepsCheck,
		),
		django.Apps(
			testsql.NewAuthAppConfig,
			testsql.NewBlogAppConfig,
		),
	)

	if err := app.Initialize(); err != nil {
		panic(err)
	}

	// Run the tests
	exitCode := m.Run()

	// Exit with the appropriate code
	os.Exit(exitCode)
}

func TestLazyUserModel(t *testing.T) {
	ctd := lazyUserModel.Load()

	if ctd == nil {
		t.Fatal("Lazy user model should not be nil")
	}

	if reflect.TypeOf(ctd.ContentObject) != reflect.TypeOf(&testsql.User{}) {
		t.Fatalf("Expected content object type to be *test_sql.User, got %T", ctd.ContentObject)
	}

	var userString = lazyUserModel.LoadString()
	if userString != "test_sql.User" {
		t.Fatalf("Expected user model string to be 'test_sql.User', got '%s'", userString)
	}
}

func TestLazyBlogPage(t *testing.T) {
	ctd := lazyBlogPage.Load("test_sql.BlogPost")

	if ctd == nil {
		t.Fatal("Lazy blog page should not be nil")
	}

	if reflect.TypeOf(ctd.ContentObject) != reflect.TypeOf(&testsql.BlogPost{}) {
		t.Fatalf("Expected content object type to be *test_sql.BlogPost, got %T", ctd.ContentObject)
	}
}

func TestSetUser(t *testing.T) {
	var user = &testsql.User{
		Name: "Test User",
	}

	var letter = &Letter{
		Text: "Hello, World!",
	}

	var defs = letter.FieldDefs()
	defs.Set("Author", user)

	if letter.Author == nil {
		t.Fatal("Author should not be nil after setting")
	}

	if letter.Author.(*testsql.User).Name != user.Name {
		t.Fatalf("Expected Author Name to be '%s', got '%s'", user.Name, letter.Author.(*testsql.User).Name)
	}

	t.Run("TestFieldRelType", func(t *testing.T) {
		var field, _ = defs.Field("Author")
		var rel = field.Rel()
		if rel.Type() != attrs.RelOneToOne {
			t.Fatalf("Expected Author field to be of type RelOneToOne, got %s", rel.Type())
		}

		if reflect.TypeOf(rel.Model()) != reflect.TypeOf(&testsql.User{}) {
			t.Fatalf("Expected Author field model to be *test_sql.User, got %T", rel.Model())
		}
	})
}
