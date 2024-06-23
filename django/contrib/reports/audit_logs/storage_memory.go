package auditlogs

import (
	"fmt"
	"slices"
	"sync"

	"github.com/Nigel2392/django/core/logger"
	"github.com/elliotchance/orderedmap/v2"
	"github.com/google/uuid"
)

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

func (i *inMemoryStorageBackend) Setup() error {
	return nil
}

func (i *inMemoryStorageBackend) Store(logEntry LogEntry) (uuid.UUID, error) {
	i.mu.Lock()
	defer i.mu.Unlock()
	return i.store(logEntry)
}

func (i *inMemoryStorageBackend) store(logEntry LogEntry) (uuid.UUID, error) {
	var id = logEntry.ID()
	if id == uuid.Nil {
		var newId = uuid.New()
		for i := 0; i < len(id); i++ {
			id[i] = newId[i]
		}
	}

	var log = logger.NameSpace(logEntry.Type())
	log.Log(logEntry.Level(), fmt.Sprint(logEntry))

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

	var keys = i.entries.Keys()
	var entries = make([]LogEntry, 0, amount)
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
	var idx = 0
	for front := i.entries.Front(); front != nil; front = front.Next() {
		if idx < offset {
			idx++
			continue
		}

		var entry = front.Value
		if entry.Type() == logType {
			entries = append(entries, entry)
		}
		if len(entries) == amount {
			break
		}
		idx++
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
