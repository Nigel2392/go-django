package auditlogs

import (
	"github.com/Nigel2392/go-django/src/contrib/reports/audit_logs/backend"
)

type Entry = backend.Entry

//func (l *Entry) FieldDefs() attrs.Definitions {
//	return attrs.Define(l,
//		attrs.Unbound("Id", &attrs.FieldConfig{
//			Primary: true,
//			Column:  "id",
//			Null:    false,
//		}),
//		attrs.Unbound("Typ", &attrs.FieldConfig{
//			Column:    "type",
//			MaxLength: 255,
//			Null:      false,
//		}),
//		attrs.Unbound("Lvl", &attrs.FieldConfig{
//			Column: "level",
//			Null:   false,
//		}),
//		attrs.Unbound("Time", &attrs.FieldConfig{
//			Column: "timestamp",
//			Null:   false,
//		}),
//		attrs.Unbound("UsrID", &attrs.FieldConfig{
//			Column: "user_id",
//			Null:   true,
//		}),
//		attrs.Unbound("Obj", &attrs.FieldConfig{
//			Column: "object",
//			Null:   true,
//		}),
//		attrs.Unbound("ObjID", &attrs.FieldConfig{
//			Column: "object_id",
//			Null:   true,
//		}),
//		attrs.Unbound("CType", &attrs.FieldConfig{
//			Column: "content_type",
//			Null:   true,
//		}),
//		attrs.Unbound("Src", &attrs.FieldConfig{
//			Column: "data",
//			Null:   true,
//		}),
//	)
//}
