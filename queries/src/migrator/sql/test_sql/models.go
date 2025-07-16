package testsql

import (
	"database/sql"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/mux/middleware/authentication"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var (
	ExtendedDefinitions        = false
	ExtendedDefinitionsUser    = false
	ExtendedDefinitionsTodo    = false
	ExtendedDefinitionsProfile = false

	DEFAULT_TIME = time.Date(2000, 9, 23, 22, 51, 0, 0, time.UTC)
	DEFAULT_UUID = uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
)

var _ authentication.User = &User{}

type User struct {
	ID        int64  `attrs:"primary"`
	Name      string `attrs:"max_length=255"`
	Email     string `attrs:"max_length=255"`
	Age       int32  `attrs:"min_value=0;max_value=120"`
	IsActive  bool   `attrs:"-"`
	FirstName string `attrs:"-"`
	LastName  string `attrs:"-"`
}

func (m *User) IsAdmin() bool {
	return m.Name == "admin"
}

func (m *User) IsAuthenticated() bool {
	return m.IsActive
}

func (m *User) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	var fields = fieldDefs.Fields()
	if ExtendedDefinitions {
		fields = append(fields, attrs.NewField(m, "FirstName", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "LastName", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitionsUser {
		fields = append(fields, attrs.NewField(m, "IsActive", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitions || ExtendedDefinitionsUser {
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type Profile struct {
	ID        int64  `attrs:"primary"`
	User      *User  `attrs:"o2o=users.User;column=user_id"`
	Image     string `attrs:"-"`
	Biography string `attrs:"-"`
	Website   string `attrs:"-"`
}

func (m *Profile) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	var fields = fieldDefs.Fields()
	if ExtendedDefinitions {
		fields = append(fields, attrs.NewField(m, "Biography", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "Website", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitionsProfile {
		fields = append(fields, attrs.NewField(m, "Image", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitions || ExtendedDefinitionsProfile {
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type Todo struct {
	ID          int64     `attrs:"primary"`
	Title       string    `attrs:"max_length=255"`
	Completed   bool      `attrs:"default=false"`
	User        *User     `attrs:"fk=users.User;column=user_id"`
	Description string    `attrs:"-"`
	CreatedAt   time.Time `attrs:"-"`
	UpdatedAt   time.Time `attrs:"-"`
}

func (m *Todo) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	var fields = fieldDefs.Fields()
	if ExtendedDefinitions {
		fields = append(fields, attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "UpdatedAt", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitionsTodo {
		fields = append(fields, attrs.NewField(m, "Description", &attrs.FieldConfig{}))
	}
	if ExtendedDefinitions || ExtendedDefinitionsTodo {
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type BlogPost struct {
	ID        int64     `attrs:"primary"`
	Title     string    `attrs:"max_length=255"`
	Body      string    `attrs:"max_length=255"`
	Author    *User     `attrs:"fk=users.User;column=author_id"`
	Published bool      `attrs:"-"`
	CreatedAt time.Time `attrs:"-"`
	UpdatedAt time.Time `attrs:"-"`
}

func (m *BlogPost) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	if ExtendedDefinitions {
		var fields = fieldDefs.Fields()
		fields = append(fields, attrs.NewField(m, "Published", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "UpdatedAt", &attrs.FieldConfig{}))
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type BlogComment struct {
	ID        int64     `attrs:"primary"`
	Body      string    `attrs:"max_length=255"`
	Author    *User     `attrs:"fk=users.User;column=author_id"`
	Post      *BlogPost `attrs:"fk=test_sql.BlogPost;column=post_id"`
	CreatedAt time.Time `attrs:"-"`
	UpdatedAt time.Time `attrs:"-"`
}

func (m *BlogComment) FieldDefs() attrs.Definitions {
	var fieldDefs = attrs.AutoDefinitions(m)
	if ExtendedDefinitions {
		var fields = fieldDefs.Fields()
		fields = append(fields, attrs.NewField(m, "CreatedAt", &attrs.FieldConfig{}))
		fields = append(fields, attrs.NewField(m, "UpdatedAt", &attrs.FieldConfig{}))
		fieldDefs = attrs.Define(m, fields...)
	}
	return fieldDefs
}

type Broad struct {
	models.Model

	Field_Drivers_Text        drivers.Text
	Field_Drivers_String      drivers.String
	Field_Drivers_Char        drivers.Char
	Field_Drivers_Int         drivers.Int
	Field_Drivers_Uint        drivers.Uint
	Field_Drivers_Float       drivers.Float
	Field_Drivers_Bool        drivers.Bool
	Field_Drivers_Bytes       drivers.Bytes
	Field_Drivers_BLOB        drivers.BLOB
	Field_Drivers_UUID        drivers.UUID
	Field_Drivers_Timestamp   drivers.Timestamp
	Field_Drivers_LocalTime   drivers.LocalTime
	Field_Drivers_DateTime    drivers.DateTime
	Field_Drivers_Email       drivers.Email
	Field_Drivers_Decimal     decimal.Decimal
	Field_any                 any
	Field_string              string
	Field_byte                []byte
	Field_int                 int
	Field_int8                int8
	Field_int16               int16
	Field_int32               int32
	Field_int64               int64
	Field_uint                uint
	Field_uint8               uint8
	Field_uint16              uint16
	Field_uint32              uint32
	Field_uint64              uint64
	Field_float32             float32
	Field_float64             float64
	Field_bool                bool
	Field_UUID                uuid.UUID
	Field_Time                time.Time
	Field_NullString          sql.NullString
	Field_NullFloat64         sql.NullFloat64
	Field_NullInt64           sql.NullInt64
	Field_NullInt32           sql.NullInt32
	Field_NullInt16           sql.NullInt16
	Field_NullBool            sql.NullBool
	Field_NullTime            sql.NullTime
	Field_NullDecimal         decimal.NullDecimal
	Field_Null_Drivers_Text   sql.Null[drivers.Text]
	Field_Null_Drivers_String sql.Null[drivers.String]
	Field_Null_Drivers_Char   sql.Null[drivers.Char]
	Field_Null_Drivers_Int    sql.Null[drivers.Int]
	Field_Null_Drivers_Uint   sql.Null[drivers.Uint]
	Field_Null_Drivers_Float  sql.Null[drivers.Float]
	Field_Null_Drivers_Bool   sql.Null[drivers.Bool]
	Field_Null_Drivers_Bytes  sql.Null[drivers.Bytes]
	Field_Null_Drivers_UUID   sql.Null[drivers.UUID]
	Field_Null_Timestamp      sql.Null[drivers.Timestamp]
	Field_Null_LocalTime      sql.Null[drivers.LocalTime]
	Field_Null_DateTime       sql.Null[drivers.DateTime]
	Field_Null_Email          sql.Null[drivers.Email]
	Field_Null_any            sql.Null[any]
	Field_Null_string         sql.Null[string]
	Field_Null_byte           sql.Null[[]byte]
	Field_Null_int            sql.Null[int]
	Field_Null_int8           sql.Null[int8]
	Field_Null_int16          sql.Null[int16]
	Field_Null_int32          sql.Null[int32]
	Field_Null_int64          sql.Null[int64]
	Field_Null_uint           sql.Null[uint]
	Field_Null_uint8          sql.Null[uint8]
	Field_Null_uint16         sql.Null[uint16]
	Field_Null_uint32         sql.Null[uint32]
	Field_Null_uint64         sql.Null[uint64]
	Field_Null_float32        sql.Null[float32]
	Field_Null_float64        sql.Null[float64]
	Field_Null_bool           sql.Null[bool]
	Field_Null_UUID           sql.Null[uuid.UUID]
	Field_Null_Time           sql.Null[time.Time]
	Field_Null_Decimal        sql.Null[decimal.Decimal]
}

var defaultValueMap = map[string]any{
	"Field_Drivers_Text":      drivers.Text("default text"),
	"Field_Drivers_String":    drivers.String("default string"),
	"Field_Drivers_Char":      drivers.Char("default char"),
	"Field_Drivers_Int":       drivers.Int(42),
	"Field_Drivers_Uint":      drivers.Uint(42),
	"Field_Drivers_Float":     drivers.Float(3.14),
	"Field_Drivers_Bool":      drivers.Bool(true),
	"Field_Drivers_Bytes":     drivers.Bytes([]byte("default bytes")),
	"Field_Drivers_BLOB":      drivers.BLOB([]byte("default blob")),
	"Field_Drivers_UUID":      drivers.UUID(DEFAULT_UUID),
	"Field_Drivers_Timestamp": drivers.Timestamp(DEFAULT_TIME),
	"Field_Drivers_LocalTime": drivers.LocalTime(DEFAULT_TIME),
	"Field_Drivers_DateTime":  drivers.DateTime(DEFAULT_TIME),
	"Field_Drivers_Email":     drivers.Email{Address: "default@example.com"},
	"Field_Drivers_Decimal":   decimal.NewFromFloat(420.69),
	"Field_any":               any("\"default any\""),
	"Field_string":            "default string",
	"Field_byte":              []byte("default byte"),
	"Field_int":               42,
	"Field_int8":              int8(42),
	"Field_int16":             int16(42),
	"Field_int32":             int32(42),
	"Field_int64":             int64(42),
	"Field_uint":              uint(42),
	"Field_uint8":             uint8(42),
	"Field_uint16":            uint16(42),
	"Field_uint32":            uint32(42),
	"Field_uint64":            uint64(42),
	"Field_float32":           float32(3.14),
	"Field_float64":           float64(3.14),
	"Field_bool":              true,
	"Field_UUID":              DEFAULT_UUID,
	"Field_Time":              DEFAULT_TIME,
	"Field_NullString": sql.NullString{
		String: "default null string",
		Valid:  true,
	},
	"Field_NullFloat64": sql.NullFloat64{
		Float64: 3.14,
		Valid:   true,
	},
	"Field_NullInt64": sql.NullInt64{
		Int64: 42,
		Valid: true,
	},
	"Field_NullInt32": sql.NullInt32{
		Int32: 42,
		Valid: true,
	},
	"Field_NullInt16": sql.NullInt16{
		Int16: 42,
		Valid: true,
	},
	"Field_NullBool": sql.NullBool{
		Bool:  true,
		Valid: true,
	},
	"Field_NullTime": sql.NullTime{
		Time:  DEFAULT_TIME,
		Valid: true,
	},
	"Field_NullDecimal": decimal.NullDecimal{
		Decimal: decimal.NewFromFloat(420.69),
		Valid:   true,
	},
	"Field_Null_Drivers_Text": sql.Null[drivers.Text]{
		V:     drivers.Text("default null text"),
		Valid: true,
	},
	"Field_Null_Drivers_String": sql.Null[drivers.String]{
		V:     drivers.String("default null string"),
		Valid: true,
	},
	"Field_Null_Drivers_Char": sql.Null[drivers.Char]{
		V:     drivers.Char("default null char"),
		Valid: true,
	},
	"Field_Null_Drivers_Int": sql.Null[drivers.Int]{
		V:     drivers.Int(42),
		Valid: true,
	},
	"Field_Null_Drivers_Uint": sql.Null[drivers.Uint]{
		V:     drivers.Uint(42),
		Valid: true,
	},
	"Field_Null_Drivers_Float": sql.Null[drivers.Float]{
		V:     drivers.Float(3.14),
		Valid: true,
	},
	"Field_Null_Drivers_Bool": sql.Null[drivers.Bool]{
		V:     drivers.Bool(true),
		Valid: true,
	},
	"Field_Null_Drivers_Bytes": sql.Null[drivers.Bytes]{
		V:     drivers.Bytes{1, 2, 3},
		Valid: true,
	},
	"Field_Null_Drivers_UUID": sql.Null[drivers.UUID]{
		V:     drivers.UUID(DEFAULT_UUID),
		Valid: true,
	},
	"Field_Null_Timestamp": sql.Null[drivers.Timestamp]{
		V:     drivers.Timestamp(DEFAULT_TIME),
		Valid: true,
	},
	"Field_Null_LocalTime": sql.Null[drivers.LocalTime]{
		V:     drivers.LocalTime(DEFAULT_TIME),
		Valid: true,
	},
	"Field_Null_DateTime": sql.Null[drivers.DateTime]{
		V:     drivers.DateTime(DEFAULT_TIME),
		Valid: true,
	},
	"Field_Null_Email": sql.Null[drivers.Email]{
		V:     drivers.Email{Address: "default@example.com"},
		Valid: true,
	},
	"Field_Null_any": sql.Null[any]{
		V:     any("default null any"),
		Valid: true,
	},
	"Field_Null_string": sql.Null[string]{
		V:     "default null string",
		Valid: true,
	},
	"Field_Null_int": sql.Null[int]{
		V:     42,
		Valid: true,
	},
	"Field_Null_int8": sql.Null[int8]{
		V:     42,
		Valid: true,
	},
	"Field_Null_int16": sql.Null[int16]{
		V:     42,
		Valid: true,
	},
	"Field_Null_int32": sql.Null[int32]{
		V:     42,
		Valid: true,
	},
	"Field_Null_int64": sql.Null[int64]{
		V:     42,
		Valid: true,
	},
	"Field_Null_uint": sql.Null[uint]{
		V:     42,
		Valid: true,
	},
	"Field_Null_uint8": sql.Null[uint8]{
		V:     42,
		Valid: true,
	},
	"Field_Null_uint16": sql.Null[uint16]{
		V:     42,
		Valid: true,
	},
	"Field_Null_uint32": sql.Null[uint32]{
		V:     42,
		Valid: true,
	},
	"Field_Null_uint64": sql.Null[uint64]{
		V:     42,
		Valid: true,
	},
	"Field_Null_float32": sql.Null[float32]{
		V:     3.14,
		Valid: true,
	},
	"Field_Null_float64": sql.Null[float64]{
		V:     3.14,
		Valid: true,
	},
	"Field_Null_bool": sql.Null[bool]{
		V:     true,
		Valid: true,
	},
	"Field_Null_UUID": sql.Null[uuid.UUID]{
		V:     DEFAULT_UUID,
		Valid: true,
	},
	"Field_Null_Time": sql.Null[time.Time]{
		V:     DEFAULT_TIME,
		Valid: true,
	},
	"Field_Null_Decimal": sql.Null[decimal.Decimal]{
		V:     decimal.NewFromFloat(420.69),
		Valid: true,
	},
}

func BroadDefaultValues() map[string]any {
	return defaultValueMap
}

func (b *Broad) FieldDefs() attrs.Definitions {
	vals := BroadDefaultValues()
	return attrs.Define(b,
		attrs.Unbound("Field_Drivers_Text", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Text"],
		}),
		attrs.Unbound("Field_Drivers_String", &attrs.FieldConfig{
			Default: vals["Field_Drivers_String"],
		}),
		attrs.Unbound("Field_Drivers_Char", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Char"],
		}),
		attrs.Unbound("Field_Drivers_Int", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Int"],
		}),
		attrs.Unbound("Field_Drivers_Uint", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Uint"],
		}),
		attrs.Unbound("Field_Drivers_Float", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Float"],
		}),
		attrs.Unbound("Field_Drivers_Bool", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Bool"],
		}),
		attrs.Unbound("Field_Drivers_Bytes", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Bytes"],
		}),
		attrs.Unbound("Field_Drivers_BLOB", &attrs.FieldConfig{
			Default: vals["Field_Drivers_BLOB"],
		}),
		attrs.Unbound("Field_Drivers_UUID", &attrs.FieldConfig{
			Default: vals["Field_Drivers_UUID"],
		}),
		attrs.Unbound("Field_Drivers_Timestamp", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Timestamp"],
		}),
		attrs.Unbound("Field_Drivers_LocalTime", &attrs.FieldConfig{
			Default: vals["Field_Drivers_LocalTime"],
		}),
		attrs.Unbound("Field_Drivers_DateTime", &attrs.FieldConfig{
			Default: vals["Field_Drivers_DateTime"],
		}),
		attrs.Unbound("Field_Drivers_Email", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Email"],
		}),
		attrs.Unbound("Field_Drivers_Decimal", &attrs.FieldConfig{
			Default: vals["Field_Drivers_Decimal"],
		}),
		attrs.Unbound("Field_any", &attrs.FieldConfig{
			Default: vals["Field_any"],
		}),
		attrs.Unbound("Field_string", &attrs.FieldConfig{
			Default: vals["Field_string"],
		}),
		attrs.Unbound("Field_byte", &attrs.FieldConfig{
			Default: vals["Field_byte"],
		}),
		attrs.Unbound("Field_int", &attrs.FieldConfig{
			Default: vals["Field_int"],
		}),
		attrs.Unbound("Field_int8", &attrs.FieldConfig{
			Default: vals["Field_int8"],
		}),
		attrs.Unbound("Field_int16", &attrs.FieldConfig{
			Default: vals["Field_int16"],
		}),
		attrs.Unbound("Field_int32", &attrs.FieldConfig{
			Default: vals["Field_int32"],
		}),
		attrs.Unbound("Field_int64", &attrs.FieldConfig{
			Default: vals["Field_int64"],
		}),
		attrs.Unbound("Field_uint", &attrs.FieldConfig{
			Default: vals["Field_uint"],
		}),
		attrs.Unbound("Field_uint8", &attrs.FieldConfig{
			Default: vals["Field_uint8"],
		}),
		attrs.Unbound("Field_uint16", &attrs.FieldConfig{
			Default: vals["Field_uint16"],
		}),
		attrs.Unbound("Field_uint32", &attrs.FieldConfig{
			Default: vals["Field_uint32"],
		}),
		attrs.Unbound("Field_uint64", &attrs.FieldConfig{
			Default: vals["Field_uint64"],
		}),
		attrs.Unbound("Field_float32", &attrs.FieldConfig{
			Default: vals["Field_float32"],
		}),
		attrs.Unbound("Field_float64", &attrs.FieldConfig{
			Default: vals["Field_float64"],
		}),
		attrs.Unbound("Field_bool", &attrs.FieldConfig{
			Default: vals["Field_bool"],
		}),
		attrs.Unbound("Field_UUID", &attrs.FieldConfig{
			Default: vals["Field_UUID"],
		}),
		attrs.Unbound("Field_Time", &attrs.FieldConfig{
			Default: vals["Field_Time"],
		}),
		attrs.Unbound("Field_NullString", &attrs.FieldConfig{
			Default: vals["Field_NullString"],
		}),
		attrs.Unbound("Field_NullFloat64", &attrs.FieldConfig{
			Default: vals["Field_NullFloat64"],
		}),
		attrs.Unbound("Field_NullInt64", &attrs.FieldConfig{
			Default: vals["Field_NullInt64"],
		}),
		attrs.Unbound("Field_NullInt32", &attrs.FieldConfig{
			Default: vals["Field_NullInt32"],
		}),
		attrs.Unbound("Field_NullInt16", &attrs.FieldConfig{
			Default: vals["Field_NullInt16"],
		}),
		attrs.Unbound("Field_NullBool", &attrs.FieldConfig{
			Default: vals["Field_NullBool"],
		}),
		attrs.Unbound("Field_NullTime", &attrs.FieldConfig{
			Default: vals["Field_NullTime"],
		}),
		attrs.Unbound("Field_NullDecimal", &attrs.FieldConfig{
			Default: vals["Field_NullDecimal"],
		}),
		attrs.Unbound("Field_Null_Drivers_Text", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Text"],
		}),
		attrs.Unbound("Field_Null_Drivers_String", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_String"],
		}),
		attrs.Unbound("Field_Null_Drivers_Char", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Char"],
		}),
		attrs.Unbound("Field_Null_Drivers_Int", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Int"],
		}),
		attrs.Unbound("Field_Null_Drivers_Uint", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Uint"],
		}),
		attrs.Unbound("Field_Null_Drivers_Float", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Float"],
		}),
		attrs.Unbound("Field_Null_Drivers_Bool", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Bool"],
		}),
		attrs.Unbound("Field_Null_Drivers_Bytes", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_Bytes"],
		}),
		attrs.Unbound("Field_Null_Drivers_UUID", &attrs.FieldConfig{
			Default: vals["Field_Null_Drivers_UUID"],
		}),
		attrs.Unbound("Field_Null_Timestamp", &attrs.FieldConfig{
			Default: vals["Field_Null_Timestamp"],
		}),
		attrs.Unbound("Field_Null_LocalTime", &attrs.FieldConfig{
			Default: vals["Field_Null_LocalTime"],
		}),
		attrs.Unbound("Field_Null_DateTime", &attrs.FieldConfig{
			Default: vals["Field_Null_DateTime"],
		}),
		attrs.Unbound("Field_Null_Email", &attrs.FieldConfig{
			Default: vals["Field_Null_Email"],
		}),
		attrs.Unbound("Field_Null_any", &attrs.FieldConfig{
			Default: vals["Field_Null_any"],
		}),
		attrs.Unbound("Field_Null_string", &attrs.FieldConfig{
			Default: vals["Field_Null_string"],
		}),
		attrs.Unbound("Field_Null_int", &attrs.FieldConfig{
			Default: vals["Field_Null_int"],
		}),
		attrs.Unbound("Field_Null_int8", &attrs.FieldConfig{
			Default: vals["Field_Null_int8"],
		}),
		attrs.Unbound("Field_Null_int16", &attrs.FieldConfig{
			Default: vals["Field_Null_int16"],
		}),
		attrs.Unbound("Field_Null_int32", &attrs.FieldConfig{
			Default: vals["Field_Null_int32"],
		}),
		attrs.Unbound("Field_Null_int64", &attrs.FieldConfig{
			Default: vals["Field_Null_int64"],
		}),
		attrs.Unbound("Field_Null_uint", &attrs.FieldConfig{
			Default: vals["Field_Null_uint"],
		}),
		attrs.Unbound("Field_Null_uint8", &attrs.FieldConfig{
			Default: vals["Field_Null_uint8"],
		}),
		attrs.Unbound("Field_Null_uint16", &attrs.FieldConfig{
			Default: vals["Field_Null_uint16"],
		}),
		attrs.Unbound("Field_Null_uint32", &attrs.FieldConfig{
			Default: vals["Field_Null_uint32"],
		}),
		attrs.Unbound("Field_Null_uint64", &attrs.FieldConfig{
			Default: vals["Field_Null_uint64"],
		}),
		attrs.Unbound("Field_Null_float32", &attrs.FieldConfig{
			Default: vals["Field_Null_float32"],
		}),
		attrs.Unbound("Field_Null_float64", &attrs.FieldConfig{
			Default: vals["Field_Null_float64"],
		}),
		attrs.Unbound("Field_Null_bool", &attrs.FieldConfig{
			Default: vals["Field_Null_bool"],
		}),
		attrs.Unbound("Field_Null_UUID", &attrs.FieldConfig{
			Default: vals["Field_Null_UUID"],
		}),
		attrs.Unbound("Field_Null_Time", &attrs.FieldConfig{
			Default: vals["Field_Null_Time"],
		}),
		attrs.Unbound("Field_Null_Decimal", &attrs.FieldConfig{
			Default: vals["Field_Null_Decimal"],
		}),
	)
}
