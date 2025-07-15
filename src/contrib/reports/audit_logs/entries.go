package auditlogs

import (
	"context"
	"fmt"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/contrib/auth/users"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/google/uuid"
)

var (
	_ queries.ActsBeforeSave = (*Entry)(nil)
)

type LogEntry interface {
	ID() uuid.UUID
	Type() string
	Level() logger.LogLevel
	Timestamp() time.Time
	User() users.User
	ObjectID() interface{}
	ContentType() contenttypes.ContentType
	Data() map[string]interface{}
}

type Entry struct {
	models.Model
	Id    drivers.UUID                         `db:"id" json:"id"`
	Typ   drivers.String                       `db:"type" json:"type"`
	Time  drivers.Timestamp                    `db:"timestamp" json:"timestamp"`
	Usr   users.User                           `db:"user" json:"user"`
	ObjID drivers.JSON[any]                    `db:"object_id" json:"object_id"`
	CType *contenttypes.BaseContentType[any]   `db:"content_type" json:"content_type"`
	Src   drivers.JSON[map[string]interface{}] `db:"data" json:"data"`
	Lvl   logger.LogLevel                      `db:"level" json:"level"`

	Obj interface{} `json:"-"`
}

func (l *Entry) BeforeSave(ctx context.Context) error {
	if l.Id.IsZero() {
		l.Id = drivers.UUID(uuid.New())
	}

	if l.Time.IsZero() {
		l.Time = drivers.CurrentTimestamp()
	}

	if l.Typ == "" {
		return fmt.Errorf("log entry type cannot be empty")
	}

	return nil
}

func (l *Entry) String() string {
	var (
		id      = l.ID()
		typ     = l.Type()
		objId   = l.ObjectID()
		cTyp    = l.ContentType()
		srcData = l.Data()
	)

	switch {
	case objId != nil && srcData != nil:
		return fmt.Sprintf(
			"<LogEntry(%q): %s> %s(%v) %v",
			typ, id, cTyp.TypeName(), objId, srcData,
		)
	case objId != nil:
		return fmt.Sprintf(
			"<LogEntry(%q): %s> %s(%v)",
			typ, id, cTyp.TypeName(), objId,
		)
	case srcData != nil:
		return fmt.Sprintf(
			"<LogEntry(%q): %s> %s %v",
			typ, id, cTyp.TypeName(), srcData,
		)
	}

	return fmt.Sprintf(
		"<LogEntry(%q): %s> %s",
		typ, id, cTyp.TypeName(),
	)
}

func (l *Entry) ID() uuid.UUID {
	return uuid.UUID(l.Id)
}

func (l *Entry) Type() string {
	return string(l.Typ)
}

func (l *Entry) Level() logger.LogLevel {
	return l.Lvl
}

func (l *Entry) Timestamp() time.Time {
	return time.Time(l.Time)
}

func (l *Entry) User() users.User {
	return l.Usr
}

func (l *Entry) Object() interface{} {
	return l.Obj
}

func (l *Entry) ObjectID() interface{} {
	return l.ObjID.Data
}

func (l *Entry) ContentType() contenttypes.ContentType {
	return l.CType
}

func (l *Entry) Data() map[string]interface{} {
	return l.Src.Data
}

func (l *Entry) Fields(attrs.Definer) []attrs.Field {
	return []attrs.Field{
		attrs.NewField(l, "Id", &attrs.FieldConfig{
			NameOverride: "ID",
			Primary:      true,
			Column:       "id",
			Null:         false,
		}),
		attrs.NewField(l, "Typ", &attrs.FieldConfig{
			NameOverride: "Type",
			Column:       "type",
			MaxLength:    255,
			Null:         false,
		}),
		attrs.NewField(l, "Lvl", &attrs.FieldConfig{
			NameOverride: "Level",
			Column:       "level",
			Null:         false,
		}),
		attrs.NewField(l, "Time", &attrs.FieldConfig{
			NameOverride: "Timestamp",
			Column:       "timestamp",
			Null:         false,
		}),
		attrs.NewField(l, "Usr", &attrs.FieldConfig{
			NameOverride: "User",
			Column:       "user_id",
			Null:         true,
			RelForeignKey: attrs.RelatedDeferred(
				attrs.RelManyToOne,
				users.GetUserModelString(),
				"", nil,
			),
		}),
		attrs.NewField(l, "Obj", &attrs.FieldConfig{
			NameOverride: "Object",
			Column:       "object",
			Null:         true,
		}),
		attrs.NewField(l, "ObjID", &attrs.FieldConfig{
			NameOverride: "ObjectID",
			Column:       "object_id",
			Null:         true,
		}),
		attrs.NewField(l, "CType", &attrs.FieldConfig{
			NameOverride: "ContentType",
			Column:       "content_type",
			Null:         true,
		}),
		attrs.NewField(l, "Src", &attrs.FieldConfig{
			NameOverride: "Data",
			Column:       "data",
			Null:         true,
		}),
	}
}

func (l *Entry) FieldDefs() attrs.Definitions {
	return l.Model.Define(l, l.Fields)
}
