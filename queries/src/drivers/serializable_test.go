package drivers_test

import (
	"database/sql/driver"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/queries/src/drivers"
	"github.com/Nigel2392/go-django/queries/src/drivers/dbtype"
)

func init() {
	dbtype.Lock()
	drivers.RegisterGoType(CustomInt(0))
	drivers.RegisterGoType(CustomString(""))
	drivers.RegisterGoType(&CustomStruct{})
	drivers.RegisterGoType(&CustomPointer{})
	drivers.RegisterGoType(&CustomValuer{})

	drivers.RegisterGoType(&CustomTextAppender{})
	drivers.RegisterGoType(&CustomTextMarshaler{})
	drivers.RegisterGoType(&CustomBinaryAppender{})
	drivers.RegisterGoType(&CustomBinaryMarshaler{})
	drivers.RegisterGoType(&CustomFallback{})
	drivers.RegisterGoType(&ErrorTextMarshaler{})
}

// Custom type 1: Int alias
type CustomInt int

// Custom type 2: String alias
type CustomString string

// Custom type 3: Struct
type CustomStruct struct {
	A string `json:"a"`
	B int    `json:"b"`
}

// Custom type 4: Pointer wrapper
type CustomPointer struct {
	V float64 `json:"v"`
}

// Custom type 5: Struct implementing driver.Valuer
type CustomValuer struct {
	Data string `json:"data"`
}

func (c CustomValuer) Value() (driver.Value, error) {
	return "valuer:" + c.Data, nil
}

func TestValue_Value(t *testing.T) {
	tests := []struct {
		name    string
		execute func() (driver.Value, error)
		want    any
	}{
		{
			name: "Standard type (int)",
			execute: func() (driver.Value, error) {
				v := &drivers.Value[int]{V: 42}
				return v.Value()
			},
			want: `{"go_type":"int","value":42}`,
		},
		{
			name: "CustomInt",
			execute: func() (driver.Value, error) {
				v := &drivers.Value[CustomInt]{V: CustomInt(100)}
				return v.Value()
			},
			want: fmt.Sprintf(
				`{"go_type":%q,"value":100}`,
				drivers.StringForType(CustomInt(0)),
			),
		},
		{
			name: "CustomString",
			execute: func() (driver.Value, error) {
				v := &drivers.Value[CustomString]{V: CustomString("test")}
				return v.Value()
			},
			want: fmt.Sprintf(
				`{"go_type":%q,"value":"test"}`,
				drivers.StringForType(CustomString("")),
			),
		},
		{
			name: "CustomStruct",
			execute: func() (driver.Value, error) {
				val := CustomStruct{A: "foo", B: 1}
				v := &drivers.Value[CustomStruct]{V: val}
				return v.Value()
			},
			want: fmt.Sprintf(
				`{"go_type":%q,"value":{"a":"foo","b":1}}`,
				drivers.StringForType(CustomStruct{}),
			),
		},
		{
			name: "CustomValuer (Executes Valuer interface)",
			execute: func() (driver.Value, error) {
				v := &drivers.Value[CustomValuer]{V: CustomValuer{Data: "test_data"}}
				return v.Value()
			},
			want: fmt.Sprintf(
				`{"go_type":%q,"value":"valuer:test_data"}`,
				drivers.StringForType(CustomValuer{}),
			),
		},
		{
			name: "CustomPointer (Nil)",
			execute: func() (driver.Value, error) {
				var ptr *CustomPointer
				v := &drivers.Value[*CustomPointer]{V: ptr}
				return v.Value()
			},
			// json.Marshal calls the struct's MarshalJSON which returns "null" due to isNil check
			want: "null",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.execute()
			if err != nil {
				t.Fatalf("Value() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Value() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestValue_Scan(t *testing.T) {
	tests := []struct {
		name    string
		execute func() error
		wantErr bool
	}{
		{
			name: "Nil source returns nil",
			execute: func() error {
				v := &drivers.Value[CustomInt]{}
				return v.Scan(nil)
			},
			wantErr: false,
		},
		{
			name: "Scan into standard int",
			execute: func() error {
				v := &drivers.Value[int]{}
				return v.Scan([]byte(`{"go_type":"int","value":42}`))
			},
			wantErr: false,
		},
		{
			name: "Scan into CustomInt",
			execute: func() error {
				v := &drivers.Value[CustomInt]{}
				payload := fmt.Sprintf(`{"go_type":%q,"value":100}`, drivers.StringForType(CustomInt(0)))
				return v.Scan([]byte(payload))
			},
			wantErr: false,
		},
		{
			name: "Scan into CustomString",
			execute: func() error {
				v := &drivers.Value[CustomString]{}
				payload := fmt.Sprintf(`{"go_type":%q,"value":"hello"}`, drivers.StringForType(CustomString("")))
				return v.Scan([]byte(payload))
			},
			wantErr: false,
		},
		{
			name: "Scan into CustomStruct",
			execute: func() error {
				v := &drivers.Value[CustomStruct]{}
				return v.Scan(fmt.Appendf([]byte{},
					`{"go_type":%q,"value":{"a":"foo","b":1}}`,
					drivers.StringForType(CustomStruct{}),
				))
			},
			wantErr: false,
		},
		{
			name: "Scan into CustomStruct (error)",
			execute: func() error {
				v := &drivers.Value[CustomStruct]{}
				// Niet verpakt in het vereiste valueJSON formaat
				return v.Scan([]byte(`{"a":"foo","b":1}`))
			},
			wantErr: true,
		},
		{
			name: "Scan into CustomValuer",
			execute: func() error {
				v := &drivers.Value[CustomValuer]{}
				payload := fmt.Sprintf(`{"go_type":%q,"value":{"data":"test"}}`, drivers.StringForType(CustomValuer{}))
				return v.Scan([]byte(payload))
			},
			wantErr: false,
		},
		{
			name: "Scan NoChanges error",
			execute: func() error {
				v := &drivers.Value[CustomInt]{}
				// Geen string/byte source om invalid format error af te dwingen
				return v.Scan(make(chan int))
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("Scan() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				t.Logf("[success] Got error %q", err)
			}
		})
	}
}

func TestValue_MarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		execute     func() ([]byte, error)
		wantErr     bool
		wantContain string
	}{
		{
			name: "Marshal CustomInt",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomInt]{V: 123}
				return json.Marshal(v)
			},
			wantContain: `"value":123`,
		},
		{
			name: "Marshal CustomString",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomString]{V: "test_string"}
				return json.Marshal(v)
			},
			wantContain: `"value":"test_string"`,
		},
		{
			name: "Marshal CustomStruct",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomStruct]{V: CustomStruct{A: "foo", B: 1}}
				return json.Marshal(v)
			},
			wantContain: `"value":{"a":"foo","b":1}`,
		},
		{
			name: "Marshal CustomPointer (non-nil)",
			execute: func() ([]byte, error) {
				v := &drivers.Value[*CustomPointer]{V: &CustomPointer{V: 1.5}}
				return json.Marshal(v)
			},
			wantContain: `"value":{"v":1.5}`,
		},
		{
			name: "Marshal CustomPointer (nil)",
			execute: func() ([]byte, error) {
				var ptr *CustomPointer
				v := &drivers.Value[*CustomPointer]{V: ptr}
				return json.Marshal(v)
			},
			// Gewijzigd: de isNil functie zorgt ervoor dat we "null" direct retourneren
			wantContain: `null`,
		},
		{
			name: "Marshal Error (unsupported JSON type)",
			execute: func() ([]byte, error) {
				v := &drivers.Value[chan int]{V: make(chan int)}
				return json.Marshal(v)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				t.Logf("[success] Got error %q", err)
			}

			if !tt.wantErr && !strings.Contains(string(got), tt.wantContain) {
				t.Errorf("MarshalJSON() got = %s, expected to contain %s", string(got), tt.wantContain)
			}
		})
	}
}

func TestValue_UnmarshalJSON(t *testing.T) {
	// Helper to dynamically build test JSON preventing hardcoding `StringForType` logic
	buildJSON := func(typ reflect.Type, defaultVal string) []byte {
		goTypeStr := drivers.StringForType(typ)
		return fmt.Appendf([]byte{}, `{"go_type": %q, "value": %s}`, goTypeStr, defaultVal)
	}

	tests := []struct {
		name    string
		execute func() error
		wantErr bool
	}{
		{
			name: "Unmarshal CustomInt",
			execute: func() error {
				v := &drivers.Value[CustomInt]{}
				data := buildJSON(reflect.TypeOf(CustomInt(0)), `123`)
				return json.Unmarshal(data, v)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal CustomString",
			execute: func() error {
				v := &drivers.Value[CustomString]{}
				data := buildJSON(reflect.TypeOf(CustomString("")), `"hello"`)
				return json.Unmarshal(data, v)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal CustomStruct",
			execute: func() error {
				v := &drivers.Value[CustomStruct]{}
				data := buildJSON(reflect.TypeOf(CustomStruct{}), `{"a":"foo","b":2}`)
				return json.Unmarshal(data, v)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal CustomPointer",
			execute: func() error {
				v := &drivers.Value[*CustomPointer]{}
				data := buildJSON(reflect.TypeOf(&CustomPointer{}), `{"v":3.14}`)
				return json.Unmarshal(data, v)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal Null Literal",
			execute: func() error {
				v := &drivers.Value[CustomStruct]{}
				data := buildJSON(reflect.TypeOf(CustomStruct{}), `null`)
				return json.Unmarshal(data, v)
			},
			wantErr: false, // returns nil early based on _json_nullLiteral check
		},
		{
			name: "Unmarshal Unregistered GOType",
			execute: func() error {
				v := &drivers.Value[CustomInt]{}
				data := []byte(`{"go_type": "unknown_type_xxx", "value": 123}`)
				return json.Unmarshal(data, v)
			},
			wantErr: true, // triggers errors.NotExists
		},
		{
			name: "Unmarshal Type Mismatch Cast Failure",
			execute: func() error {
				v := &drivers.Value[CustomInt]{}
				// The JSON provides a string, and specifies the type as CustomString.
				// Reflection parses it into a CustomString, but `s.V, ok = v.(T)` checks against CustomInt.
				data := buildJSON(reflect.TypeOf(CustomString("")), `"invalid"`)
				return json.Unmarshal(data, v)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// Custom type 6: Implements encoding.TextAppender
type CustomTextAppender struct{ Data string }

func (c CustomTextAppender) AppendText(b []byte) ([]byte, error) {
	return append(b, []byte(c.Data)...), nil
}

// Custom type 7: Implements encoding.TextMarshaler & encoding.TextUnmarshaler
type CustomTextMarshaler struct{ Data string }

func (c CustomTextMarshaler) MarshalText() ([]byte, error) {
	return []byte(c.Data), nil
}
func (c *CustomTextMarshaler) UnmarshalText(text []byte) error {
	c.Data = string(text)
	return nil
}

// Custom type 8: Implements encoding.BinaryAppender
type CustomBinaryAppender struct{ Data string }

func (c CustomBinaryAppender) AppendBinary(b []byte) ([]byte, error) {
	return append(b, []byte(c.Data)...), nil
}

// Custom type 9: Implements encoding.BinaryMarshaler & encoding.BinaryUnmarshaler
type CustomBinaryMarshaler struct{ Data string }

func (c CustomBinaryMarshaler) MarshalBinary() ([]byte, error) {
	return []byte(c.Data), nil
}
func (c *CustomBinaryMarshaler) UnmarshalBinary(data []byte) error {
	c.Data = string(data)
	return nil
}

// Custom type 10: Fallback struct (Uses Base64/JSON)
type CustomFallback struct {
	Val int `json:"val"`
}

// Custom type 11: Error simulating marshaler/unmarshaler
type ErrorTextMarshaler struct{}

func (e ErrorTextMarshaler) MarshalText() ([]byte, error) {
	return nil, fmt.Errorf("forced marshal error")
}
func (e *ErrorTextMarshaler) UnmarshalText(text []byte) error {
	return fmt.Errorf("forced unmarshal error")
}

func TestValue_MarshalText(t *testing.T) {
	tests := []struct {
		name        string
		execute     func() ([]byte, error)
		wantErr     bool
		wantContain string
	}{
		{
			name: "Marshal Fallback Value (JSON to Base64)",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomFallback]{V: CustomFallback{Val: 123}}
				return v.MarshalText()
			},
			// {"val":123} base64 encoded using base64.StdEncoding
			wantContain: base64.StdEncoding.EncodeToString([]byte(`{"val":123}`)),
		},
		{
			name: "Marshal Fallback Pointer (JSON to Base64)",
			execute: func() ([]byte, error) {
				v := &drivers.Value[*CustomFallback]{V: &CustomFallback{Val: 456}}
				return v.MarshalText()
			},
			wantContain: base64.StdEncoding.EncodeToString([]byte(`{"val":456}`)),
		},
		{
			name: "Marshal Untyped Nil",
			execute: func() ([]byte, error) {
				v := &drivers.Value[any]{V: nil}
				return v.MarshalText()
			},
			wantContain: ":null",
		},
		{
			name: "Marshal Typed Nil",
			execute: func() ([]byte, error) {
				var ptr *CustomFallback
				v := &drivers.Value[*CustomFallback]{V: ptr}
				return v.MarshalText()
			},
			wantContain: ":null",
		},
		{
			name: "Marshal TextAppender",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomTextAppender]{V: CustomTextAppender{Data: "appended_text"}}
				return v.MarshalText()
			},
			wantContain: ":appended_text",
		},
		{
			name: "Marshal TextMarshaler",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomTextMarshaler]{V: CustomTextMarshaler{Data: "marshaled_text"}}
				return v.MarshalText()
			},
			wantContain: ":marshaled_text",
		},
		{
			name: "Marshal BinaryAppender",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomBinaryAppender]{V: CustomBinaryAppender{Data: "appended_bin"}}
				return v.MarshalText()
			},
			wantContain: ":appended_bin",
		},
		{
			name: "Marshal BinaryMarshaler",
			execute: func() ([]byte, error) {
				v := &drivers.Value[CustomBinaryMarshaler]{V: CustomBinaryMarshaler{Data: "marshaled_bin"}}
				return v.MarshalText()
			},
			wantContain: ":marshaled_bin",
		},
		{
			name: "Marshal Error from TextMarshaler",
			execute: func() ([]byte, error) {
				v := &drivers.Value[ErrorTextMarshaler]{V: ErrorTextMarshaler{}}
				return v.MarshalText()
			},
			wantErr: true,
		},
		{
			name: "Marshal Fallback JSON Error",
			execute: func() ([]byte, error) {
				// Channels cannot be marshaled into JSON
				v := &drivers.Value[chan int]{V: make(chan int)}
				return v.MarshalText()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !strings.Contains(string(got), tt.wantContain) {
				t.Errorf("MarshalText() got = %s, expected to contain %s", string(got), tt.wantContain)
			}
		})
	}
}

func TestValue_UnmarshalText(t *testing.T) {
	// Helper to dynamically build expected text format payload: `<GOType>:<Data>`
	buildText := func(typ reflect.Type, val string) []byte {
		return []byte(drivers.StringForType(typ) + ":" + val)
	}

	tests := []struct {
		name    string
		execute func() error
		wantErr bool
	}{
		{
			name: "Unmarshal Fallback (Base64 to JSON)",
			execute: func() error {
				v := &drivers.Value[CustomFallback]{}
				// {"val":789} base64 encoded (using StdEncoding as the unmarshaler expects)
				base64Data := base64.StdEncoding.EncodeToString([]byte(`{"val":789}`))
				data := buildText(reflect.TypeOf(CustomFallback{}), base64Data)
				return v.UnmarshalText(data)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal null literal",
			execute: func() error {
				v := &drivers.Value[*CustomFallback]{}
				data := buildText(reflect.TypeOf(&CustomFallback{}), string("null"))
				return v.UnmarshalText(data)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal TextUnmarshaler",
			execute: func() error {
				v := &drivers.Value[CustomTextMarshaler]{}
				data := buildText(reflect.TypeOf(CustomTextMarshaler{}), "test_text_data")
				return v.UnmarshalText(data)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal BinaryUnmarshaler",
			execute: func() error {
				v := &drivers.Value[CustomBinaryMarshaler]{}
				data := buildText(reflect.TypeOf(CustomBinaryMarshaler{}), "test_binary_data")
				return v.UnmarshalText(data)
			},
			wantErr: false,
		},
		{
			name: "Unmarshal Missing Colon (Malformed)",
			execute: func() error {
				v := &drivers.Value[CustomTextMarshaler]{}
				return v.UnmarshalText([]byte("invalid_format_without_colon"))
			},
			wantErr: true,
		},
		{
			name: "Unmarshal Unregistered Type",
			execute: func() error {
				v := &drivers.Value[CustomTextMarshaler]{}
				return v.UnmarshalText([]byte("unknown_type_xxx:data"))
			},
			wantErr: true,
		},
		{
			name: "Unmarshal TextUnmarshaler Error",
			execute: func() error {
				v := &drivers.Value[ErrorTextMarshaler]{}
				data := buildText(reflect.TypeOf(ErrorTextMarshaler{}), "err_data")
				return v.UnmarshalText(data)
			},
			wantErr: true,
		},
		{
			name: "Unmarshal Invalid Base64 in Fallback",
			execute: func() error {
				v := &drivers.Value[CustomFallback]{}
				data := buildText(reflect.TypeOf(CustomFallback{}), "invalid base64!!!")
				return v.UnmarshalText(data)
			},
			wantErr: true,
		},
		{
			name: "Unmarshal Invalid JSON inside Base64",
			execute: func() error {
				v := &drivers.Value[CustomFallback]{}
				// Valid base64, but resolves to "invalid_json" string.
				data := buildText(reflect.TypeOf(CustomFallback{}), "aW52YWxpZF9qc29u")
				return v.UnmarshalText(data)
			},
			wantErr: true,
		},
		{
			name: "Unmarshal Type Mismatch Cast Failure",
			execute: func() error {
				v := &drivers.Value[CustomFallback]{}
				data := buildText(reflect.TypeOf(CustomTextMarshaler{}), "data")
				return v.UnmarshalText(data)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.wantErr {
						t.Errorf("UnmarshalText() unexpectedly panicked: %v", r)
					}
				}
			}()

			err := tt.execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
