//go:build (!mysql && !postgres && !mariadb) || (!mysql && !postgres && !mysql_local && !mariadb && !sqlite)

package auditlogs_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/expr"
	testsql "github.com/Nigel2392/go-django/queries/src/migrator/sql/test_sql"
	django "github.com/Nigel2392/go-django/src"
	auditlogs "github.com/Nigel2392/go-django/src/contrib/reports/audit_logs"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/djester/testdb"
	"github.com/google/uuid"
)

var entries []*auditlogs.Entry

var entryIds = []uuid.UUID{
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
	uuid.New(),
}

func init() {

	var _, db = testdb.Open()
	var app = django.Global
	if django.Global == nil {
		app = django.App(django.Configure(map[string]interface{}{
			django.APPVAR_DATABASE: db,
		}),
			django.Flag(
				django.FlagSkipDepsCheck,
				django.FlagSkipChecks,
				django.FlagSkipCmds,
			),
			django.Apps(
				testsql.NewAuthAppConfig,
				auditlogs.NewAppConfig,
			),
		)

		logger.Setup(&logger.Logger{
			Level:       logger.DBG,
			WrapPrefix:  logger.ColoredLogWrapper,
			OutputDebug: os.Stdout,
			OutputInfo:  os.Stdout,
			OutputWarn:  os.Stdout,
			OutputError: os.Stderr,
		})
	}

	if err := app.Initialize(); err != nil {
		panic(fmt.Errorf("failed to initialize app: %w", err))
	}

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &auditlogs.Entry{},
		GetObject:     func() interface{} { return &auditlogs.Entry{} },
		GetLabel:      func(ctx context.Context) string { return "Entry" },
	})

	for i := 0; i < len(entryIds); i++ {
		var entry = &auditlogs.Entry{
			Id:   drivers.UUID(entryIds[i]),
			Typ:  drivers.String(fmt.Sprintf("type-%d", i)),
			Lvl:  logger.LogLevel(i % 4),
			Time: drivers.CurrentTimestamp(),
			Usr: &testsql.User{
				ID: int64(i),
			},
			ObjID: drivers.JSON[any]{Data: fmt.Sprintf("object-%d", i)},
			CType: contenttypes.NewContentType[any](&auditlogs.Entry{}),
			Src: drivers.JSON[map[string]interface{}]{
				Data: map[string]interface{}{
					"key": fmt.Sprintf("value-%d", i),
				},
			},
		}
		entries = append(entries, entry)
	}

	_, err := queries.GetQuerySet(&auditlogs.Entry{}).BulkCreate(entries)
	if err != nil {
		panic(err)
	}

}

func TestGetByID(t *testing.T) {
	for i, id := range entryIds {
		entryRow, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter("ID", id).Get()
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		entry := entryRow.Object
		if entry == nil {
			t.Fatalf("%d entry not found", i)
		}
		if entry.ID() != id {
			t.Fatalf("%d expected id %s, got %s", i, id, entry.ID())
		}
		if entry.Type() != fmt.Sprintf("type-%d", i) {
			t.Fatalf("%d expected type %s, got %s", i, fmt.Sprintf("type-%d", i), entry.Type())
		}
		if entry.Level() != logger.LogLevel(i%4) {
			t.Fatalf("%d expected level %d, got %d", i, i%4, entry.Level())
		}
	}
}

func TestRetrieveTyped(t *testing.T) {
	for i := 0; i < len(entryIds); i++ {
		typ := fmt.Sprintf("type-%d", i)
		entryRows, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter("Type", typ).All()
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if len(entryRows) != 1 {
			for _, entry := range entryRows {
				t.Logf("Entry: %+v", entry.Object)
			}
			t.Fatalf("%d expected 1 entry, got %d", i, len(entryRows))
		}
		if entryRows[0].Object.Type() != typ {
			t.Fatalf("%d expected type %s, got %s", i, typ, entryRows[0].Object.Type())
		}
	}
}

func TestRetrieveForUser(t *testing.T) {
	for i := 0; i < len(entryIds); i++ {
		entryRows, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter("User", i).All()
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if len(entryRows) != 1 {
			t.Fatalf("%d expected 1 entry, got %d", i, len(entryRows))
		}
		if entryRows[0].Object.ID() != entryIds[i] {
			t.Fatalf("%d expected id %s, got %s", i, entryIds[i], entryRows[0].Object.ID())
		}
		if entryRows[0].Object.Usr == nil {
			t.Fatalf("%d expected user to be set, got nil", i)
		}
		if attrs.Get[int64](entryRows[0].Object.Usr, "ID") != int64(i) {
			t.Fatalf("%d expected user id %v, got %v", i, i, entryRows[0].Object.Usr)
		}
	}
}

func TestRetrieveForObj(t *testing.T) {
	for i := 0; i < len(entryIds); i++ {
		var id = drivers.JSON[any]{Data: fmt.Sprintf("object-%d", i)}
		entryRows, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter("ObjectID", id).All()
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if len(entryRows) != 1 {
			t.Fatalf("%d expected 1 entry, got %d", i, len(entryRows))
		}
		if entryRows[0].Object.ID() != entryIds[i] {
			t.Fatalf("%d expected id %s, got %s", i, entryIds[i], entryRows[0].Object.ID())
		}
		if entryRows[0].Object.ObjID != id {
			t.Fatalf("%d expected object id %v, got %v", i, id, entryRows[0].Object.ObjID)
		}
	}
}

type filterTest struct {
	filters           []expr.Expression
	expectedFilterIDs []uuid.UUID
}

var filterTests = []filterTest{
	{
		filters: []expr.Expression{
			// auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterID, entryIds[0]),
			expr.Q("ID", entryIds[0]),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[0]},
	},
	{
		filters: []expr.Expression{
			auditlogs.FilterType("type-1", "type-2"),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[1], entryIds[2]},
	},
	{
		filters: []expr.Expression{
			auditlogs.FilterLevel(logger.LogLevel(3)),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[3], entryIds[7], entryIds[11], entryIds[15], entryIds[19], entryIds[23]},
	},
	{
		filters: []expr.Expression{
			auditlogs.FilterType("type-1", "type-2", "type-7"),
			auditlogs.FilterLevelGT(logger.LogLevel(1)),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[2], entryIds[7]},
	},
}

func TestFilter(t *testing.T) {

	for i, test := range filterTests {
		t.Run(fmt.Sprintf("filter-%d", i), func(t *testing.T) {
			entryRows, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter(test.filters).All()
			if err != nil {
				t.Fatalf("%d %s", i, err)
			}
			if len(entryRows) != len(test.expectedFilterIDs) {
				t.Fatalf("%d expected %d entries, got %d", i, len(test.expectedFilterIDs), len(entryRows))
			}

			var m = make(map[uuid.UUID]bool)
			for _, id := range test.expectedFilterIDs {
				m[id] = true
			}

			for _, entry := range entryRows {
				if !m[entry.Object.ID()] {
					t.Fatalf("%d expected id %s, got %s", i, test.expectedFilterIDs, entry.Object.ID())
				}

				var fromDB, err = queries.GetQuerySet(&auditlogs.Entry{}).Filter("ID", entry.Object.ID()).Get()
				if err != nil {
					t.Fatalf("%d %s", i, err)
				}

				if fromDB == nil {
					t.Fatalf("%d entry not found", i)
				}

				if fromDB.Object.ID() != entry.Object.ID() {
					t.Fatalf("%d expected id %s, got %s", i, entry.Object.ID(), fromDB.Object.ID())
				}

				if fromDB.Object.Type() != entry.Object.Type() {
					t.Fatalf("%d expected type %s, got %s", i, entry.Object.Type(), fromDB.Object.Type())
				}

				if fromDB.Object.Level() != entry.Object.Level() {
					t.Fatalf("%d expected level %d, got %d", i, entry.Object.Level(), fromDB.Object.Level())
				}

				if attrs.PrimaryKey(fromDB.Object.User()) != attrs.PrimaryKey(entry.Object.User()) {
					t.Fatalf("%d expected user id %v, got %v", i, attrs.PrimaryKey(entry.Object.User()), attrs.PrimaryKey(fromDB.Object.User()))
				}
			} //

		})
	}
}

func TestFilterCount(t *testing.T) {

	for i, test := range filterTests {
		t.Run(fmt.Sprintf("filter-count-%d", i), func(t *testing.T) {
			count, err := queries.GetQuerySet(&auditlogs.Entry{}).Filter(test.filters).Count()
			if err != nil {
				t.Fatalf("%d %s", i, err)
			}
			if count != int64(len(test.expectedFilterIDs)) {
				t.Fatalf("%d expected %d entries, got %d", i, len(test.expectedFilterIDs), count)
			}
		})
	}
}
