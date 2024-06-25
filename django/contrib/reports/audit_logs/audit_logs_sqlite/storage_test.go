package auditlogs_sqlite_test

import (
	"database/sql"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	auditlogs_sqlite "github.com/Nigel2392/django/contrib/reports/audit_logs/audit_logs_sqlite"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"
)

// For now only used to make sure tests pass on github actions
// This will be removed when the package is properly developed and tested
// This makes sure that the authentication check is enabled only when running on github actions
var IS_GITHUB_ACTIONS = true

var db *sql.DB

var entries []auditlogs.LogEntry

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

	var actionsVar = os.Getenv("GITHUB_ACTIONS")
	if slices.Contains([]string{"true", "1"}, strings.ToLower(actionsVar)) {
		IS_GITHUB_ACTIONS = true
	}

	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}

	var err error
	db, err = sql.Open("sqlite3", ":memory:")
	if err != nil {
		panic(err)
	}
	var backend = auditlogs_sqlite.NewSQLiteStorageBackend(db)
	auditlogs.RegisterBackend(backend)

	db.Exec("DROP TABLE IF EXISTS audit_logs;")

	err = backend.Setup()
	if err != nil {
		panic(err)
	}

	contenttypes.Register(&contenttypes.ContentTypeDefinition{
		ContentObject: &auditlogs.Entry{},
		GetObject:     func() interface{} { return &auditlogs.Entry{} },
		GetLabel:      func() string { return "Entry" },
	})

	for i := 0; i < len(entryIds); i++ {
		var entry = &auditlogs.Entry{
			Id:    entryIds[i],
			Typ:   fmt.Sprintf("type-%d", i),
			Lvl:   logger.LogLevel(i % 5),
			UsrID: fmt.Sprintf("user-%d", i),
			Time:  time.Now(),
			ObjID: fmt.Sprintf("object-%d", i),
			CType: contenttypes.NewContentType[any](&auditlogs.Entry{}),
			Obj:   &auditlogs.Entry{},
			Src: map[string]interface{}{
				"key": fmt.Sprintf("value-%d", i),
			},
		}
		entries = append(entries, entry)
	}

	_, err = backend.StoreMany(entries)
	if err != nil {
		panic(err)
	}

}

func TestGetByID(t *testing.T) {
	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}
	for i, id := range entryIds {
		entry, err := auditlogs.Backend().Retrieve(id)
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if entry == nil {
			t.Fatalf("%d entry not found", i)
		}
		if entry.ID() != id {
			t.Fatalf("%d expected id %s, got %s", i, id, entry.ID())
		}
		if entry.Type() != fmt.Sprintf("type-%d", i) {
			t.Fatalf("%d expected type %s, got %s", i, fmt.Sprintf("type-%d", i), entry.Type())
		}
		if entry.Level() != logger.LogLevel(i%5) {
			t.Fatalf("%d expected level %d, got %d", i, i%5, entry.Level())
		}
	}
}

func TestRetrieveTyped(t *testing.T) {
	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}
	for i := 0; i < len(entryIds); i++ {
		typ := fmt.Sprintf("type-%d", i)
		entries, err := auditlogs.Backend().EntryFilter(
			[]auditlogs.AuditLogFilter{
				auditlogs.FilterType(typ),
			},
			25,
			0,
		)
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if len(entries) != 1 {
			t.Fatalf("%d expected 1 entry, got %d", i, len(entries))
		}
		if entries[0].Type() != typ {
			t.Fatalf("%d expected type %s, got %s", i, typ, entries[0].Type())
		}
	}
}

func TestRetrieveForUser(t *testing.T) {
	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}
	for i := 0; i < len(entryIds); i++ {
		var id = fmt.Sprintf("user-%d", i)
		entries, err := auditlogs.Backend().EntryFilter(
			[]auditlogs.AuditLogFilter{
				auditlogs.FilterUserID(id),
			},
			25,
			0,
		)
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if len(entries) != 1 {
			t.Fatalf("%d expected 1 entry, got %d", i, len(entries))
		}
		if entries[0].ID() != entryIds[i] {
			t.Fatalf("%d expected id %s, got %s", i, entryIds[i], entries[0].ID())
		}
		if entries[0].UserID() != id {
			t.Fatalf("%d expected user id %v, got %v", i, id, entries[0].UserID())
		}
	}
}

func TestRetrieveForObj(t *testing.T) {
	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}
	for i := 0; i < len(entryIds); i++ {
		var id = fmt.Sprintf("object-%d", i)
		entries, err := auditlogs.Backend().EntryFilter(
			[]auditlogs.AuditLogFilter{
				auditlogs.FilterObjectID(id),
			},
			25,
			0,
		)
		if err != nil {
			t.Fatalf("%d %s", i, err)
		}
		if len(entries) != 1 {
			t.Fatalf("%d expected 1 entry, got %d", i, len(entries))
		}
		if entries[0].ID() != entryIds[i] {
			t.Fatalf("%d expected id %s, got %s", i, entryIds[i], entries[0].ID())
		}
		if entries[0].ObjectID() != id {
			t.Fatalf("%d expected object id %v, got %v", i, id, entries[0].ObjectID())
		}
	}
}

type filterTest struct {
	filters           []auditlogs.AuditLogFilter
	expectedFilterIDs []uuid.UUID
}

var filterTests = []filterTest{
	{
		filters: []auditlogs.AuditLogFilter{
			auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterID, entryIds[0]),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[0]},
	},
	{
		filters: []auditlogs.AuditLogFilter{
			auditlogs.FilterType("type-1", "type-2"),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[1], entryIds[2]},
	},
	{
		filters: []auditlogs.AuditLogFilter{
			auditlogs.FilterLevelEqual(logger.LogLevel(3)),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[3], entryIds[8], entryIds[13], entryIds[18], entryIds[23]},
	},
	{
		filters: []auditlogs.AuditLogFilter{
			auditlogs.FilterObjectID("object-1", "object-2"),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[1], entryIds[2]},
	},
	{
		filters: []auditlogs.AuditLogFilter{
			auditlogs.FilterType("type-1", "type-2", "type-7"),
			auditlogs.FilterLevelGreaterThan(logger.LogLevel(1)),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[2], entryIds[7]},
	},
}

func TestFilter(t *testing.T) {
	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}

	for i, test := range filterTests {
		t.Run(fmt.Sprintf("filter-%d-%s", i, test.filters[0].Name()), func(t *testing.T) {
			entries, err := auditlogs.Backend().EntryFilter(test.filters, 25, 0)
			if err != nil {
				t.Fatalf("%d %s", i, err)
			}
			if len(entries) != len(test.expectedFilterIDs) {
				t.Fatalf("%d expected %d entries, got %d", i, len(test.expectedFilterIDs), len(entries))
			}

			var m = make(map[uuid.UUID]bool)
			for _, id := range test.expectedFilterIDs {
				m[id] = true
			}

			for _, entry := range entries {
				if !m[entry.ID()] {
					t.Fatalf("%d expected id %s, got %s", i, test.expectedFilterIDs, entry.ID())
				}

				var fromDB, err = auditlogs.Backend().Retrieve(entry.ID())
				if err != nil {
					t.Fatalf("%d %s", i, err)
				}

				if fromDB == nil {
					t.Fatalf("%d entry not found", i)
				}

				if fromDB.ID() != entry.ID() {
					t.Fatalf("%d expected id %s, got %s", i, entry.ID(), fromDB.ID())
				}

				if fromDB.Type() != entry.Type() {
					t.Fatalf("%d expected type %s, got %s", i, entry.Type(), fromDB.Type())
				}

				if fromDB.Level() != entry.Level() {
					t.Fatalf("%d expected level %d, got %d", i, entry.Level(), fromDB.Level())
				}

				if fromDB.UserID() != entry.UserID() {
					t.Fatalf("%d expected user id %v, got %v", i, entry.UserID(), fromDB.UserID())
				}
			}

		})
	}
}

func TestFilterCount(t *testing.T) {
	if IS_GITHUB_ACTIONS {
		// Skip tests if not running on github actions
		return
	}

	for i, test := range filterTests {
		t.Run(fmt.Sprintf("filter-count-%d-%s", i, test.filters[0].Name()), func(t *testing.T) {
			count, err := auditlogs.Backend().CountFilter(test.filters)
			if err != nil {
				t.Fatalf("%d %s", i, err)
			}
			if count != len(test.expectedFilterIDs) {
				t.Fatalf("%d expected %d entries, got %d", i, len(test.expectedFilterIDs), count)
			}
		})
	}
}
