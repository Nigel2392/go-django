package drivers

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"net/mail"
	"reflect"
	"strconv"
	"time"

	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	formfields "github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/go-django/src/forms/widgets"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/shopspring/decimal"
)

func init() {
	dbtype.Add(reflect.Bool, dbtype.Bool, true)
	dbtype.Add(reflect.Int, dbtype.Int, true)
	dbtype.Add(reflect.Int8, dbtype.Int, true)
	dbtype.Add(reflect.Int16, dbtype.Int, true)
	dbtype.Add(reflect.Int32, dbtype.Int, true)
	dbtype.Add(reflect.Int64, dbtype.Int, true)
	dbtype.Add(reflect.Uint, dbtype.Uint, true)
	dbtype.Add(reflect.Uint8, dbtype.Uint, true)
	dbtype.Add(reflect.Uint16, dbtype.Uint, true)
	dbtype.Add(reflect.Uint32, dbtype.Uint, true)
	dbtype.Add(reflect.Uint64, dbtype.Uint, true)
	dbtype.Add(reflect.Uintptr, dbtype.Uint, true)
	dbtype.Add(reflect.Float32, dbtype.Float, true)
	dbtype.Add(reflect.Float64, dbtype.Float, true)
	dbtype.Add(reflect.Complex64, dbtype.Float, true)
	dbtype.Add(reflect.Complex128, dbtype.Float, true)
	dbtype.Add(reflect.Interface, dbtype.JSON, true)
	dbtype.Add(reflect.Map, dbtype.JSON, true)
	dbtype.Add(reflect.Slice, dbtype.JSON, true)
	dbtype.Add(reflect.String, dbtype.String, true)

	dbtype.Add(Text(""), dbtype.Text)
	dbtype.Add(String(""), dbtype.String)
	dbtype.Add(Char(""), dbtype.Char)
	dbtype.Add(Int(0), dbtype.Int)
	dbtype.Add(Uint(0), dbtype.Uint)
	dbtype.Add(Float(0.0), dbtype.Float)
	dbtype.Add(Bool(false), dbtype.Bool)
	dbtype.Add(Bytes(nil), dbtype.Bytes)
	dbtype.Add(BLOB(nil), dbtype.BLOB)
	dbtype.Add(UUID(uuid.UUID{}), dbtype.UUID)
	dbtype.Add(ULID(ulid.ULID{}), dbtype.ULID)
	dbtype.Add(Timestamp{}, dbtype.Timestamp)
	dbtype.Add(LocalTime{}, dbtype.LocalTime)
	dbtype.Add(DateTime{}, dbtype.DateTime)
	dbtype.Add(Email{}, dbtype.String)
	dbtype.Add(decimal.Decimal{}, dbtype.Decimal)

	dbtype.Add(*new(any), dbtype.JSON)
	dbtype.Add(*new(string), dbtype.String)
	dbtype.Add(*new([]byte), dbtype.Bytes)
	dbtype.Add(*new(int), dbtype.Int)
	dbtype.Add(*new(int8), dbtype.Int)
	dbtype.Add(*new(int16), dbtype.Int)
	dbtype.Add(*new(int32), dbtype.Int)
	dbtype.Add(*new(int64), dbtype.Int)
	dbtype.Add(*new(uint), dbtype.Uint)
	dbtype.Add(*new(uint8), dbtype.Uint)
	dbtype.Add(*new(uint16), dbtype.Uint)
	dbtype.Add(*new(uint32), dbtype.Uint)
	dbtype.Add(*new(uint64), dbtype.Uint)
	dbtype.Add(*new(float32), dbtype.Float)
	dbtype.Add(*new(float64), dbtype.Float)
	dbtype.Add(*new(bool), dbtype.Bool)
	dbtype.Add(*new(uuid.UUID), dbtype.UUID)
	dbtype.Add(*new(ulid.ULID), dbtype.ULID)
	dbtype.Add(*new(time.Time), dbtype.DateTime)
	dbtype.Add(reflect.TypeOf((interface{})(nil)), dbtype.JSON)

	dbtype.Add(sql.NullString{}, dbtype.Text)
	dbtype.Add(sql.NullFloat64{}, dbtype.Float)
	dbtype.Add(sql.NullInt64{}, dbtype.Int)
	dbtype.Add(sql.NullInt32{}, dbtype.Int)
	dbtype.Add(sql.NullInt16{}, dbtype.Int)
	dbtype.Add(sql.NullBool{}, dbtype.Bool)
	dbtype.Add(sql.NullByte{}, dbtype.Bytes)
	dbtype.Add(sql.NullTime{}, dbtype.DateTime)
	dbtype.Add(decimal.NullDecimal{}, dbtype.Decimal)

	dbtype.Add(sql.Null[Text]{}, dbtype.Text)
	dbtype.Add(sql.Null[String]{}, dbtype.String)
	dbtype.Add(sql.Null[Char]{}, dbtype.Char)
	dbtype.Add(sql.Null[Int]{}, dbtype.Int)
	dbtype.Add(sql.Null[Uint]{}, dbtype.Uint)
	dbtype.Add(sql.Null[Float]{}, dbtype.Float)
	dbtype.Add(sql.Null[Bool]{}, dbtype.Bool)
	dbtype.Add(sql.Null[Bytes]{}, dbtype.Bytes)
	dbtype.Add(sql.Null[UUID]{}, dbtype.UUID)
	dbtype.Add(sql.Null[Timestamp]{}, dbtype.Timestamp)
	dbtype.Add(sql.Null[LocalTime]{}, dbtype.LocalTime)
	dbtype.Add(sql.Null[DateTime]{}, dbtype.DateTime)
	dbtype.Add(sql.Null[Email]{}, dbtype.String)
	dbtype.Add(sql.Null[ULID]{}, dbtype.ULID)

	dbtype.Add(sql.Null[any]{}, dbtype.JSON)
	dbtype.Add(sql.Null[string]{}, dbtype.String)
	dbtype.Add(sql.Null[[]byte]{}, dbtype.Bytes)
	dbtype.Add(sql.Null[int]{}, dbtype.Int)
	dbtype.Add(sql.Null[int8]{}, dbtype.Int)
	dbtype.Add(sql.Null[int16]{}, dbtype.Int)
	dbtype.Add(sql.Null[int32]{}, dbtype.Int)
	dbtype.Add(sql.Null[int64]{}, dbtype.Int)
	dbtype.Add(sql.Null[uint]{}, dbtype.Uint)
	dbtype.Add(sql.Null[uint8]{}, dbtype.Uint)
	dbtype.Add(sql.Null[uint16]{}, dbtype.Uint)
	dbtype.Add(sql.Null[uint32]{}, dbtype.Uint)
	dbtype.Add(sql.Null[uint64]{}, dbtype.Uint)
	dbtype.Add(sql.Null[float32]{}, dbtype.Float)
	dbtype.Add(sql.Null[float64]{}, dbtype.Float)
	dbtype.Add(sql.Null[bool]{}, dbtype.Bool)
	dbtype.Add(sql.Null[uuid.UUID]{}, dbtype.UUID)
	dbtype.Add(sql.Null[time.Time]{}, dbtype.DateTime)
	dbtype.Add(sql.Null[decimal.Decimal]{}, dbtype.Decimal)
	dbtype.Add(sql.Null[ulid.ULID]{}, dbtype.ULID)

	attrs.RegisterFormFieldType(Text(""), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.CharField(append(opts, formfields.Widget(widgets.NewTextarea(nil)))...)
	})
	attrs.RegisterFormFieldType(String(""), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.CharField(opts...)
	})
	attrs.RegisterFormFieldType(Char(""), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.CharField(opts...)
	})
	attrs.RegisterFormFieldType(Int(0), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.NumberField[Int](opts...)
	})
	attrs.RegisterFormFieldType(Uint(0), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.NumberField[Uint](opts...)
	})
	attrs.RegisterFormFieldType(Float(0.0), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.NumberField[Float](opts...)
	})
	attrs.RegisterFormFieldType(Bool(false), func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.BooleanField(opts...)
	})
	attrs.RegisterFormFieldType(UUID{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.CharField(opts...)
	})
	attrs.RegisterFormFieldType(ULID{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.CharField(opts...)
	})
	attrs.RegisterFormFieldType(Timestamp{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.DateField(widgets.DateWidgetTypeDateTime, opts...)
	})
	attrs.RegisterFormFieldType(LocalTime{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.DateField(widgets.DateWidgetTypeDateTime, opts...)
	})
	attrs.RegisterFormFieldType(DateTime{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.DateField(widgets.DateWidgetTypeDateTime, opts...)
	})
	attrs.RegisterFormFieldType(Email{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.EmailField(opts...)
	})
	attrs.RegisterFormFieldType(JSON[any]{}, func(opts ...func(formfields.Field)) formfields.Field {
		return formfields.JSONField[any](opts...)
	})
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
	BLOB        []byte
	JSON[T any] struct {
		Data T
		Null bool
	}
	UUID  uuid.UUID
	ULID  ulid.ULID
	Email mail.Address

	timeType  time.Time
	Timestamp time.Time
	LocalTime time.Time
	DateTime  time.Time
)

func (t Int) String() string {
	return strconv.FormatInt(int64(t), 10)
}
func (t Uint) String() string {
	return strconv.FormatUint(uint64(t), 10)
}
func (t Float) String() string {
	return strconv.FormatFloat(float64(t), 'f', -1, 64)
}
func (t Bool) String() string {
	if t {
		return "true"
	}
	return "false"
}
func (t Bytes) String() string {
	return string(t)
}
func (t BLOB) String() string {
	return string(t)
}

// Complete shim for the ulid.ULID type
func (id ULID) Bytes() []byte                      { return (ulid.ULID)(id).Bytes() }
func (id ULID) Compare(other ulid.ULID) int        { return (ulid.ULID)(id).Compare(other) }
func (id ULID) Entropy() []byte                    { return (ulid.ULID)(id).Entropy() }
func (id ULID) IsZero() bool                       { return (ulid.ULID)(id).IsZero() }
func (id ULID) MarshalBinary() ([]byte, error)     { return (ulid.ULID)(id).MarshalBinary() }
func (id ULID) MarshalBinaryTo(dst []byte) error   { return (ulid.ULID)(id).MarshalBinaryTo(dst) }
func (id ULID) MarshalText() ([]byte, error)       { return (ulid.ULID)(id).MarshalText() }
func (id ULID) MarshalTextTo(dst []byte) error     { return (ulid.ULID)(id).MarshalTextTo(dst) }
func (id *ULID) Scan(src interface{}) error        { return (*ulid.ULID)(id).Scan(src) }
func (id *ULID) SetEntropy(e []byte) error         { return (*ulid.ULID)(id).SetEntropy(e) }
func (id *ULID) SetTime(ms uint64) error           { return (*ulid.ULID)(id).SetTime(ms) }
func (id ULID) String() string                     { return (ulid.ULID)(id).String() }
func (id ULID) Time() uint64                       { return (ulid.ULID)(id).Time() }
func (id ULID) Timestamp() time.Time               { return (ulid.ULID)(id).Timestamp() }
func (id *ULID) UnmarshalBinary(data []byte) error { return (*ulid.ULID)(id).UnmarshalBinary(data) }
func (id *ULID) UnmarshalText(v []byte) error      { return (*ulid.ULID)(id).UnmarshalText(v) }
func (id ULID) Value() (driver.Value, error)       { return (ulid.ULID)(id).Value() }

func NewUUID() UUID {
	return (UUID)(uuid.New())
}

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

func (t UUID) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return json.Marshal(nil)
	}
	return json.Marshal(uuid.UUID(t).String())
}

func (t *UUID) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*t = UUID(uuid.Nil)
		return nil
	}
	var u uuid.UUID
	if err := json.Unmarshal(data, &u); err != nil {
		return errors.Wrap(err, "failed to unmarshal UUID value")
	}

	*t = UUID(u)
	return nil
}

func (t JSON[T]) DBType() dbtype.Type {
	return dbtype.JSON
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
		return nil, errors.Wrap(err, "failed to marshal Text value")
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
		return errors.TypeMismatch.Wrapf(
			"cannot scan %T into JSON[T]", value,
		)
	}
	if len(bytes) == 0 {
		j.Null = true
		j.Data = newT
		return nil
	}
	if err := json.Unmarshal(bytes, &newT); err != nil {
		return errors.Wrap(err, "failed to unmarshal JSON value")
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
		return errors.Wrap(err, "failed to unmarshal JSON value")
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
		return errors.TypeMismatch.Wrapf(
			"cannot scan %T into timeType",
			value,
		)
	}
	return nil
}

func (t timeType) MarshalJSON() ([]byte, error) {
	return (time.Time)(t).MarshalJSON()
}

func (t *timeType) UnmarshalJSON(data []byte) error {
	var _t time.Time
	if err := _t.UnmarshalJSON(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal timeType")
	}
	*t = timeType(_t)
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
func (t Timestamp) MarshalJSON() ([]byte, error) { return (time.Time)(t).MarshalJSON() }
func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var _t time.Time
	if err := _t.UnmarshalJSON(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal Timestamp")
	}
	*t = Timestamp(_t)
	return nil
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
func (t LocalTime) MarshalJSON() ([]byte, error) {
	return (time.Time)(t).MarshalJSON()
}
func (t *LocalTime) UnmarshalJSON(data []byte) error {
	var _t time.Time
	if err := _t.UnmarshalJSON(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal LocalTime")
	}
	*t = LocalTime(_t)
	return nil
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
func (t DateTime) MarshalJSON() ([]byte, error) {
	return (time.Time)(t).MarshalJSON()
}
func (t *DateTime) UnmarshalJSON(data []byte) error {
	var _t time.Time
	if err := _t.UnmarshalJSON(data); err != nil {
		return errors.Wrap(err, "failed to unmarshal DateTime")
	}
	*t = DateTime(_t)
	return nil
}

func MustParseEmail(addr string) *Email {
	a, err := mail.ParseAddress(addr)
	if err != nil {
		panic(errors.Wrap(err, "failed to parse email address"))
	}
	return (*Email)(a)
}

func (e Email) String() string {
	return e.Address
}

func (e *Email) Scan(src interface{}) error {
	switch v := src.(type) {
	case string:
		a, err := mail.ParseAddress(v)
		if err != nil {
			return err
		}
		*e = Email(*a)
		return nil
	case []byte:
		a, err := mail.ParseAddress(string(v))
		if err != nil {
			return err
		}
		*e = Email(*a)
		return nil
	default:
		return errors.TypeMismatch.Wrapf(
			"cannot scan %T into Email", src,
		)
	}
}

func (e Email) Value() (driver.Value, error) {
	var addr = e.Address
	return addr, nil
}
