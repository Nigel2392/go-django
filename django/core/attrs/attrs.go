package attrs

type Definer interface {
	FieldDefs() Definitions
}

type Definitions interface {
	Set(name string, value interface{})
	Get(name string) interface{}
	Field(name string) (f Field, ok bool)
	ForceSet(name string, value interface{})
}

type Field interface {
	IsBlank() bool
	IsEditable() bool
	GetValue() interface{}
	SetValue(v interface{}, force bool)
}
