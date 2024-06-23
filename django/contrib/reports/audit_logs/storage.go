package auditlogs

import (
	"errors"
	"slices"

	"github.com/Nigel2392/django/core/logger"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
)

var ErrLogEntryNotFound = errors.New("log entry not found")

type StorageBackend interface {
	Store(logEntry LogEntry) (uuid.UUID, error)
	StoreMany(logEntries []LogEntry) ([]uuid.UUID, error)
	Retrieve(id uuid.UUID) (LogEntry, error)
	RetrieveMany(amount, offset int) ([]LogEntry, error)
	RetrieveTyped(logType string, amount, offset int) ([]LogEntry, error)
}

type inMemoryStorageBackend struct {
	entries *orderedmap.OrderedMap[uuid.UUID, LogEntry]
}

func newInMemoryStorageBackend() StorageBackend {
	return &inMemoryStorageBackend{
		entries: orderedmap.NewOrderedMap[uuid.UUID, LogEntry](),
	}
}

func (i *inMemoryStorageBackend) Store(logEntry LogEntry) (uuid.UUID, error) {
	var id = logEntry.ID()
	var typ = logEntry.Type()
	var lvl = logEntry.Level()
	var t = logEntry.Timestamp()
	var objId = logEntry.ObjectID()
	var cTyp = logEntry.ContentType()
	var srcData = logEntry.Data()

	if id == uuid.Nil {
		var newId = uuid.New()
		for i := 0; i < len(id); i++ {
			id[i] = newId[i]
		}
	}

	var log = logger.NameSpace(typ)
	var timeFormatted = t.Format("2006-01-02 15:04:05")
	switch {
	case objId != nil && srcData != nil:
		log.Logf(
			lvl, "<LogEntry: %s - %s> %s(%v) %v",
			id, timeFormatted, cTyp.TypeName(), objId, srcData,
		)
	case objId != nil:
		log.Logf(
			lvl, "<LogEntry: %s - %s> %s(%v)",
			id, timeFormatted, cTyp.TypeName(), objId,
		)
	case srcData != nil:
		log.Logf(
			lvl, "<LogEntry: %s - %s> %s %v",
			id, timeFormatted, cTyp.TypeName(), srcData,
		)
	default:
		log.Logf(
			lvl, "<LogEntry: %s - %s> %s",
			id, timeFormatted, cTyp.TypeName(),
		)
	}

	i.entries.Set(id, logEntry)
	return id, nil
}

func (i *inMemoryStorageBackend) StoreMany(logEntries []LogEntry) ([]uuid.UUID, error) {
	var ids = make([]uuid.UUID, 0, len(logEntries))
	for _, entry := range logEntries {
		var id, err = i.Store(entry)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (i *inMemoryStorageBackend) Retrieve(id uuid.UUID) (LogEntry, error) {
	var entry, ok = i.entries.Get(id)
	if !ok {
		return nil, ErrLogEntryNotFound
	}
	return entry, nil
}

func (i *inMemoryStorageBackend) RetrieveMany(amount, offset int) ([]LogEntry, error) {
	var entries = make([]LogEntry, 0)
	var keys = i.entries.Keys()
	if offset >= len(keys) {
		return nil, nil
	}
	if offset+amount > len(keys) {
		amount = len(keys) - offset
	}
	keys = keys[offset : offset+amount]
	for _, key := range keys {
		var entry, _ = i.entries.Get(key)
		entries = append(entries, entry)
	}
	slices.SortFunc(entries, sortEntries)
	return entries, nil
}

func (i *inMemoryStorageBackend) RetrieveTyped(logType string, amount, offset int) ([]LogEntry, error) {
	var entries = make([]LogEntry, 0)
	var keys = i.entries.Keys()
	for _, key := range keys {
		var entry, _ = i.entries.Get(key)
		if entry.Type() == logType {
			entries = append(entries, entry)
		}
	}
	slices.SortFunc(entries, sortEntries)
	return entries, nil
}

func sortEntries(i, j LogEntry) int {
	var t1, t2 = i.Timestamp(), j.Timestamp()
	if t1.Before(t2) {
		return -1
	}
	if t1.After(t2) {
		return 1
	}
	return 0
}
