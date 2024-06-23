package auditlogs_test

import (
	"fmt"
	"testing"

	auditlogs "github.com/Nigel2392/django/contrib/reports/audit_logs"
	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"

	_ "github.com/mattn/go-sqlite3"
)

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
	var backend = auditlogs.NewInMemoryStorageBackend()
	auditlogs.RegisterBackend(backend)

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
			ObjID: fmt.Sprintf("object-%d", i),
			CType: contenttypes.NewContentType[any](&auditlogs.Entry{}),
			Obj:   &auditlogs.Entry{},
			Src: map[string]interface{}{
				"key": fmt.Sprintf("value-%d", i),
			},
		}
		entries = append(entries, entry)
	}

	var _, err = backend.StoreMany(entries)
	if err != nil {
		panic(err)
	}
}

func TestGetByID(t *testing.T) {
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
	for i := 0; i < len(entryIds); i++ {
		typ := fmt.Sprintf("type-%d", i)
		entries, err := auditlogs.Backend().RetrieveTyped(typ, 25, 0)
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
	for i := 0; i < len(entryIds); i++ {
		var id = fmt.Sprintf("user-%d", i)
		entries, err := auditlogs.Backend().RetrieveForUser(id, 25, 0)
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
	for i := 0; i < len(entryIds); i++ {
		var id = fmt.Sprintf("object-%d", i)
		entries, err := auditlogs.Backend().RetrieveForObject(id, 25, 0)
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
			auditlogs.FilterType("type-1", "type-2", "type-7"),
			auditlogs.FilterLevelGreaterThan(logger.LogLevel(1)),
		},
		expectedFilterIDs: []uuid.UUID{entryIds[2], entryIds[7]},
	},
}

func TestFilter(t *testing.T) {
	//for i := 0; i < len(entryIds); i++ {
	//	var filters = []auditlogs.AuditLogFilter{
	//		auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterID, entryIds[i]),
	//		// auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterType, fmt.Sprintf("type-%d", i)),
	//		// auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterLevel_EQ, i%5),
	//		// auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterUserID, fmt.Sprintf("user-%d", i)),
	//		// auditlogs.NewAuditLogFilter(auditlogs.AuditLogFilterObjectID, fmt.Sprintf("object-%d", i)),
	//	}
	//	entries, err := auditlogs.Backend().Filter(filters, 25, 0)
	//	if err != nil {
	//		t.Fatalf("%d %s", i, err)
	//	}
	//	if len(entries) != 1 {
	//		t.Fatalf("%d expected 1 entry, got %d", i, len(entries))
	//	}
	//	if entries[0].ID() != entryIds[i] {
	//		t.Fatalf("%d expected id %s, got %s", i, entryIds[i], entries[0].ID())
	//	}
	//	if entries[0].Type() != fmt.Sprintf("type-%d", i) {
	//		t.Fatalf("%d expected type %s, got %s", i, fmt.Sprintf("type-%d", i), entries[0].Type())
	//	}
	//	if entries[0].Level() != logger.LogLevel(i%5) {
	//		t.Fatalf("%d expected level %d, got %d", i, i%5, entries[0].Level())
	//	}
	//	if entries[0].UserID() != fmt.Sprintf("user-%d", i) {
	//		t.Fatalf("%d expected user id %v, got %v", i, fmt.Sprintf("user-%d", i), entries[0].UserID())
	//	}
	//	if entries[0].ObjectID() != fmt.Sprintf("object-%d", i) {
	//		t.Fatalf("%d expected object id %v, got %v", i, fmt.Sprintf("object-%d", i), entries[0].ObjectID())
	//	}
	//}

	for i, test := range filterTests {
		t.Run(fmt.Sprintf("filter-%d", i), func(t *testing.T) {
			entries, err := auditlogs.Backend().Filter(test.filters, 25, 0)
			if err != nil {
				t.Fatalf("%d %s", i, err)
			}
			if len(entries) != len(test.expectedFilterIDs) {
				t.Fatalf("%d expected %d entries, got %d", i, len(test.expectedFilterIDs), len(entries))
			}
			for j, entry := range entries {
				if entry.ID() != test.expectedFilterIDs[j] {
					t.Fatalf("%d expected id %s, got %s", i, test.expectedFilterIDs[j], entry.ID())
				}
			}
		})
	}
}
