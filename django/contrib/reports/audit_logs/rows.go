package auditlogs

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/Nigel2392/django/core/contenttypes"
	"github.com/Nigel2392/django/core/logger"
	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func SerializeRow(l LogEntry) (id uuid.UUID, typeStr string, level int, timestamp time.Time, userId, objectID []byte, contentType contenttypes.ContentType, data string) {
	id = l.ID()
	typeStr = l.Type()
	level = int(l.Level())
	timestamp = l.Timestamp()
	contentType = l.ContentType()
	objectID = nil
	var objId = l.ObjectID()
	if objId != nil {
		var b = new(bytes.Buffer)
		enc := json.NewEncoder(b)
		err := enc.Encode(objId)
		if err != nil {
			return
		}
		objectID = b.Bytes()
	}

	var usrId = l.UserID()
	if usrId != nil {
		var b = new(bytes.Buffer)
		enc := json.NewEncoder(b)
		err := enc.Encode(usrId)
		if err != nil {
			return
		}
		userId = b.Bytes()
	}

	var jsonData []byte
	if l.Data() != nil {
		var b = new(bytes.Buffer)
		if err := json.NewEncoder(b).Encode(l.Data()); err != nil {
			return
		}
		jsonData = b.Bytes()
	}
	return id, typeStr, level, timestamp, userId, objectID, contentType, string(jsonData)
}

func ScanRow(row interface{ Scan(dest ...any) error }) (LogEntry, error) {
	var (
		id          uuid.UUID
		typeStr     string
		level       int
		timestamp   time.Time
		userId      string
		objectID    string
		data        string
		contentType = contenttypes.BaseContentType[any]{}
	)

	err := row.Scan(&id, &typeStr, &level, &timestamp, &userId, &objectID, &contentType, &data)
	if err != nil {
		return nil, err
	}

	var goObjectID interface{}
	if len(objectID) > 0 {
		var err = json.Unmarshal([]byte(objectID), &goObjectID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode objectID")
		}
	}

	var goUserID interface{}
	if len(userId) > 0 {
		var err = json.Unmarshal([]byte(userId), &goUserID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode userID")
		}
	}

	var goData map[string]interface{}
	if len(data) > 0 {
		var err = json.Unmarshal([]byte(data), &goData)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode data")
		}
	}

	return &Entry{
		Id:    id,
		Typ:   typeStr,
		Lvl:   logger.LogLevel(level),
		Time:  timestamp,
		ObjID: goObjectID,
		UsrID: goUserID,
		CType: &contentType,
		Src:   goData,
	}, nil
}
