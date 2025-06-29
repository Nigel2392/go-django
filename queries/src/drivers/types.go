package drivers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"time"

	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/google/uuid"
)

func init() {
	TYPES.Add(reflect.Bool, TypeBool, "", true)
	TYPES.Add(reflect.Int, TypeInt, "", true)
	TYPES.Add(reflect.Int8, TypeInt, "", true)
	TYPES.Add(reflect.Int16, TypeInt, "", true)
	TYPES.Add(reflect.Int32, TypeInt, "", true)
	TYPES.Add(reflect.Int64, TypeInt, "", true)
	TYPES.Add(reflect.Uint, TypeUint, "", true)
	TYPES.Add(reflect.Uint8, TypeUint, "", true)
	TYPES.Add(reflect.Uint16, TypeUint, "", true)
	TYPES.Add(reflect.Uint32, TypeUint, "", true)
	TYPES.Add(reflect.Uint64, TypeUint, "", true)
	TYPES.Add(reflect.Uintptr, TypeUint, "", true)
	TYPES.Add(reflect.Float32, TypeFloat, "", true)
	TYPES.Add(reflect.Float64, TypeFloat, "", true)
	TYPES.Add(reflect.Complex64, TypeFloat, "", true)
	TYPES.Add(reflect.Complex128, TypeFloat, "", true)
	TYPES.Add(reflect.Interface, TypeJSON, "", true)
	TYPES.Add(reflect.Map, TypeJSON, "", true)
	TYPES.Add(reflect.Slice, TypeJSON, "", true)
	TYPES.Add(reflect.String, TypeString, "", true)

	TYPES.Add(Text(""), TypeText, "TEXT")
	TYPES.Add(String(""), TypeString, "STRING")
	TYPES.Add(Char(""), TypeChar, "CHAR")
	TYPES.Add(Int(0), TypeInt, "INT")
	TYPES.Add(Uint(0), TypeUint, "UINT")
	TYPES.Add(Float(0.0), TypeFloat, "FLOAT")
	TYPES.Add(Bool(false), TypeBool, "BOOLEAN")
	TYPES.Add(Bytes(nil), TypeBytes, "BYTES")
	TYPES.Add(UUID(uuid.UUID{}), TypeUUID, "UUID")
	TYPES.Add(Timestamp{}, TypeTimestamp, "TIMESTAMP")
	TYPES.Add(LocalTime{}, TypeLocalTime, "LOCALTIME")
	TYPES.Add(DateTime{}, TypeDateTime, "DATETIME")

	TYPES.Add(*new(any), TypeJSON, "")
	TYPES.Add(*new(string), TypeString, "")
	TYPES.Add(*new([]byte), TypeBytes, "")
	TYPES.Add(*new(int), TypeInt, "")
	TYPES.Add(*new(int8), TypeInt, "")
	TYPES.Add(*new(int16), TypeInt, "")
	TYPES.Add(*new(int32), TypeInt, "")
	TYPES.Add(*new(int64), TypeInt, "")
	TYPES.Add(*new(uint), TypeUint, "")
	TYPES.Add(*new(uint8), TypeUint, "")
	TYPES.Add(*new(uint16), TypeUint, "")
	TYPES.Add(*new(uint32), TypeUint, "")
	TYPES.Add(*new(uint64), TypeUint, "")
	TYPES.Add(*new(float32), TypeFloat, "")
	TYPES.Add(*new(float64), TypeFloat, "")
	TYPES.Add(*new(bool), TypeBool, "")
	TYPES.Add(*new(uuid.UUID), TypeUUID, "")
	TYPES.Add(*new(time.Time), TypeDateTime, "")

	TYPES.Add(sql.NullString{}, TypeText, "")
	TYPES.Add(sql.NullFloat64{}, TypeFloat, "")
	TYPES.Add(sql.NullInt64{}, TypeInt, "")
	TYPES.Add(sql.NullInt32{}, TypeInt, "")
	TYPES.Add(sql.NullInt16{}, TypeInt, "")
	TYPES.Add(sql.NullBool{}, TypeBool, "")
	TYPES.Add(sql.NullByte{}, TypeBytes, "")
	TYPES.Add(sql.NullTime{}, TypeDateTime, "")

	TYPES.Add(sql.Null[Text]{}, TypeText, "")
	TYPES.Add(sql.Null[String]{}, TypeString, "")
	TYPES.Add(sql.Null[Char]{}, TypeChar, "")
	TYPES.Add(sql.Null[Int]{}, TypeInt, "")
	TYPES.Add(sql.Null[Uint]{}, TypeUint, "")
	TYPES.Add(sql.Null[Float]{}, TypeFloat, "")
	TYPES.Add(sql.Null[Bool]{}, TypeBool, "")
	TYPES.Add(sql.Null[Bytes]{}, TypeBytes, "")
	TYPES.Add(sql.Null[UUID]{}, TypeUUID, "")
	TYPES.Add(sql.Null[Timestamp]{}, TypeTimestamp, "")
	TYPES.Add(sql.Null[LocalTime]{}, TypeLocalTime, "")
	TYPES.Add(sql.Null[DateTime]{}, TypeDateTime, "")

	TYPES.Add(sql.Null[any]{}, TypeJSON, "")
	TYPES.Add(sql.Null[string]{}, TypeString, "")
	TYPES.Add(sql.Null[[]byte]{}, TypeBytes, "")
	TYPES.Add(sql.Null[int]{}, TypeInt, "")
	TYPES.Add(sql.Null[int8]{}, TypeInt, "")
	TYPES.Add(sql.Null[int16]{}, TypeInt, "")
	TYPES.Add(sql.Null[int32]{}, TypeInt, "")
	TYPES.Add(sql.Null[int64]{}, TypeInt, "")
	TYPES.Add(sql.Null[uint]{}, TypeUint, "")
	TYPES.Add(sql.Null[uint8]{}, TypeUint, "")
	TYPES.Add(sql.Null[uint16]{}, TypeUint, "")
	TYPES.Add(sql.Null[uint32]{}, TypeUint, "")
	TYPES.Add(sql.Null[uint64]{}, TypeUint, "")
	TYPES.Add(sql.Null[float32]{}, TypeFloat, "")
	TYPES.Add(sql.Null[float64]{}, TypeFloat, "")
	TYPES.Add(sql.Null[bool]{}, TypeBool, "")
	TYPES.Add(sql.Null[uuid.UUID]{}, TypeUUID, "")
	TYPES.Add(sql.Null[time.Time]{}, TypeDateTime, "")

	TYPES.Add((contenttypes.ContentType)(nil), TypeText, "")
	TYPES.Add(contenttypes.BaseContentType[attrs.Definer]{}, TypeText, "")
	TYPES.Add(contenttypes.BaseContentType[any]{}, TypeText, "")
	TYPES.Add(&contenttypes.BaseContentType[attrs.Definer]{}, TypeText, "")
	TYPES.Add(&contenttypes.BaseContentType[any]{}, TypeText, "")
}

type (
	Text        string
	String      string
	Char        string
	Int         int64
	Uint        uint64
	Float       float64
	Bool        bool
	Bytes       []byte
	JSON[T any] struct {
		Data T
		Null bool
	}
	UUID uuid.UUID

	timeType  time.Time
	Timestamp time.Time
	LocalTime time.Time
	DateTime  time.Time
)

func (t UUID) String() string {
	return uuid.UUID(t).String()
}

func (t UUID) IsZero() bool {
	if !(t[0] == 0 && t[1] == 0 && t[14] == 0 && t[15] == 0) {
		return false
	}
	return uuid.UUID(t) == uuid.Nil
}

func (t *UUID) Scan(value any) error {

	// handle pgx uuid - it is provided as a [16]byte slice
	// which google/uuid.UUID does not support scanning
	if v, ok := value.([16]byte); ok {
		*t = UUID(v)
		return nil
	}

	return (*uuid.UUID)(t).Scan(value)
}

func (t UUID) Value() (driver.Value, error) {
	return uuid.UUID(t).Value()
}

func (t JSON[T]) DBType() Type {
	return TypeJSON
}

func isZero(rval reflect.Value) bool {
	if !rval.IsValid() {
		return true
	}
	switch rval.Kind() {
	case reflect.Ptr, reflect.Interface:
		return rval.IsNil() || (!rval.IsNil() && isZero(rval.Elem()))
	case reflect.Array, reflect.Slice:
		if rval.IsNil() {
			return true
		}
		if rval.Len() == 0 {
			return true
		}
	case reflect.Map:
		if rval.IsNil() {
			return true
		}
		if rval.Len() == 0 {
			return true
		}
	}
	return rval.IsZero()
}

func (t JSON[T]) IsZero() bool {
	if t.Null {
		return true
	}
	var val = reflect.ValueOf(t.Data)
	return isZero(val)
}

func (t JSON[T]) Value() (driver.Value, error) {
	var bytes, err = json.Marshal(t.Data)
	if err != nil {
		return nil, errs.Wrap(err, "failed to marshal Text value")
	}
	return string(bytes), nil
}

func (j *JSON[T]) Scan(value any) error {
	var newT T
	var bytes []byte
	switch v := value.(type) {
	case string:
		bytes = []byte(v)
	case []byte:
		bytes = v
	case nil:
		j.Null = true
	default:
		return errs.ErrInvalidType
	}
	if len(bytes) == 0 {
		j.Null = true
		j.Data = newT
		return nil
	}
	if err := json.Unmarshal(bytes, &newT); err != nil {
		return errs.Wrap(err, "failed to unmarshal JSON value")
	}
	j.Null = false
	j.Data = newT
	return nil
}

func (t JSON[T]) MarshalJSON() ([]byte, error) {
	if t.Null {
		return json.Marshal(nil)
	}
	return json.Marshal(t.Data)
}

func (t *JSON[T]) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		t.Null = true
		t.Data = *new(T)
		return nil
	}

	var newT T
	if err := json.Unmarshal(data, &newT); err != nil {
		return errs.Wrap(err, "failed to unmarshal JSON value")
	}

	t.Null = false
	t.Data = newT
	return nil
}

func (t *timeType) Scan(value any) error {
	switch v := value.(type) {
	case time.Time:
		*t = timeType(v)
		return nil
	case string:
		var _t, err = time.Parse(time.RFC3339, v)
		*t = timeType(_t)
		return err
	case []byte:
		var _t, err = time.Parse(time.RFC3339, string(v))
		*t = timeType(_t)
		return err
	case int64:
		*t = timeType(time.Unix(v, 0))
	case uint64:
		*t = timeType(time.Unix(int64(v), 0))
	default:
		return fmt.Errorf(
			"cannot scan %T into timeType: %w",
			value, errs.ErrInvalidType,
		)
	}
	return nil
}

func CurrentTimestamp() Timestamp {
	return Timestamp(time.Now().UTC().Truncate(time.Millisecond))
}
func (t Timestamp) String() string {
	return t.Time().Format(time.RFC3339Nano)
}
func (t Timestamp) IsZero() bool { return t.Time().IsZero() }
func (t Timestamp) Add(d time.Duration) Timestamp {
	return Timestamp(t.Time().Add(d).Truncate(time.Millisecond))
}
func (t Timestamp) Time() time.Time       { return time.Time(t).Truncate(time.Millisecond) }
func (t *Timestamp) Scan(value any) error { return (*timeType)(t).Scan(value) }
func (t Timestamp) Value() (driver.Value, error) {
	return t.Time(), nil
}

func CurrentLocalTime() LocalTime {
	return LocalTime(time.Now().Local().Truncate(time.Second))
}
func (t LocalTime) String() string {
	return t.Time().Format(time.RFC3339)
}
func (t LocalTime) IsZero() bool { return t.Time().IsZero() }
func (t LocalTime) Add(d time.Duration) LocalTime {
	return LocalTime(t.Time().Add(d).Truncate(time.Second))
}
func (t LocalTime) Time() time.Time       { return time.Time(t).Truncate(time.Second) }
func (t *LocalTime) Scan(value any) error { return (*timeType)(t).Scan(value) }
func (t LocalTime) Value() (driver.Value, error) {
	return t.Time(), nil
}

func CurrentDateTime() DateTime {
	return DateTime(time.Now().UTC().Truncate(time.Second))
}
func (t DateTime) String() string {
	return t.Time().Format(time.RFC3339)
}
func (t DateTime) IsZero() bool { return t.Time().IsZero() }
func (t DateTime) Add(d time.Duration) DateTime {
	return DateTime(t.Time().Add(d).Truncate(time.Second))
}
func (t DateTime) Time() time.Time       { return time.Time(t).Truncate(time.Second) }
func (t *DateTime) Scan(value any) error { return (*timeType)(t).Scan(value) }
func (t DateTime) Value() (driver.Value, error) {
	return t.Time(), nil
}
