package auditlogs

import (
	"errors"
	"slices"
	"sync"

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
	mu      *sync.RWMutex
}

func newInMemoryStorageBackend() StorageBackend {
	return &inMemoryStorageBackend{
		entries: orderedmap.NewOrderedMap[uuid.UUID, LogEntry](),
		mu:      &sync.RWMutex{},
	}
}

func (i *inMemoryStorageBackend) Store(logEntry LogEntry) (uuid.UUID, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store(logEntry)
}

func (i *inMemoryStorageBackend) store(logEntry LogEntry) (uuid.UUID, error) {
	var (
		id      = logEntry.ID()
		typ     = logEntry.Type()
		lvl     = logEntry.Level()
		objId   = logEntry.ObjectID()
		cTyp    = logEntry.ContentType()
		srcData = logEntry.Data()
		log     = logger.NameSpace(typ)
	)

	if id == uuid.Nil {
		var newId = uuid.New()
		for i := 0; i < len(id); i++ {
			id[i] = newId[i]
		}
	}

	switch {
	case objId != nil && srcData != nil:
		log.Logf(
			lvl, "<LogEntry: %s> %s(%v) %v",
			id, cTyp.TypeName(), objId, srcData,
		)
	case objId != nil:
		log.Logf(
			lvl, "<LogEntry: %s> %s(%v)",
			id, cTyp.TypeName(), objId,
		)
	case srcData != nil:
		log.Logf(
			lvl, "<LogEntry: %s> %s %v",
			id, cTyp.TypeName(), srcData,
		)
	default:
		log.Logf(
			lvl, "<LogEntry: %s> %s",
			id, cTyp.TypeName(),
		)
	}

	i.entries.Set(id, logEntry)
	return id, nil
}

func (i *inMemoryStorageBackend) StoreMany(logEntries []LogEntry) ([]uuid.UUID, error) {
	i.mu.Lock()
	defer i.mu.Unlock()

	var ids = make([]uuid.UUID, 0, len(logEntries))
	for _, entry := range logEntries {
		var id, err = i.store(entry)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (i *inMemoryStorageBackend) Retrieve(id uuid.UUID) (LogEntry, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

	var entry, ok = i.entries.Get(id)
	if !ok {
		return nil, ErrLogEntryNotFound
	}

	return entry, nil
}

func (i *inMemoryStorageBackend) RetrieveMany(amount, offset int) ([]LogEntry, error) {
	i.mu.RLock()
	defer i.mu.RUnlock()

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
	i.mu.RLock()
	defer i.mu.RUnlock()

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
