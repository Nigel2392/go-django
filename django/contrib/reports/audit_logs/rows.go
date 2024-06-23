package auditlogs

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"time"

	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
)

func SerializeRow(l LogEntry) (id uuid.UUID, typeStr string, level int, timestamp time.Time, objectID []byte, contentType contenttypes.ContentType, data string) {
	id = l.ID()
	typeStr = l.Type()
	level = int(l.Level())
	timestamp = l.Timestamp()
	contentType = l.ContentType()
	objectID = nil
	if l.ObjectID() != nil {
		var b = new(bytes.Buffer)
		enc := gob.NewEncoder(b)
		err := enc.Encode(l.ObjectID())
		if err != nil {
			return
		}
		objectID = b.Bytes()
	}
	var jsonData []byte
	if l.Data() != nil {
		var b = new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(l.Data()); err != nil {
			return
		}
		jsonData = b.Bytes()
	}
	return id, typeStr, level, timestamp, objectID, contentType, string(jsonData)
}

func ScanRow(row interface{ Scan(dest ...any) error }) (LogEntry, error) {
	var (
		id          uuid.UUID
		typeStr     string
		level       int
		timestamp   time.Time
		objectID    []byte
		data        []byte
		contentType = contenttypes.BaseContentType[any]{}
	)

	err := row.Scan(&id, &typeStr, &level, &timestamp, &objectID, &contentType, &data)
	if err != nil {
		return nil, err
	}

	var goObjectID interface{}
	if objectID != nil {
		b := bytes.NewBuffer(objectID)
		dec := gob.NewDecoder(b)
		err = dec.Decode(&goObjectID)
		if err != nil {
			return nil, err
		}
	}

	var goData map[string]interface{}
	if data != nil {
		b := bytes.NewBuffer(data)
		if err := json.NewDecoder(b).Decode(&goData); err != nil {
			return nil, err
		}
	}

	return &Entry{
		Id:    id,
		Typ:   typeStr,
		Lvl:   logger.LogLevel(level),
		Time:  timestamp,
		ObjID: goObjectID,
		CType: &contentType,
		Src:   goData,
	}, nil
}
