package attrs

import (
	"fmt"
	"net/mail"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Nigel2392/django/core/assert"
)

func fieldNames(d Definer, exclude []string) []string {
	var excludeMap = make(map[string]struct{})
	for _, name := range exclude {
		excludeMap[name] = struct{}{}
	}

	var (
		fields = d.FieldDefs().Fields()
		n      = len(fields)
		names  = make([]string, 0, n)
	)

	for _, f := range fields {
		if _, ok := excludeMap[f.Name()]; ok {
			continue
		}
		names = append(names, f.Name())
	}

	return names
}

func FieldNames(d any, exclude []string) []string {
	if d == nil {
		return nil
	}
	if d, ok := d.(Definer); ok {
		return fieldNames(d, exclude)
	}
	var (
		rTyp       = reflect.TypeOf(d)
		excludeMap = make(map[string]struct{})
	)
	if rTyp.Kind() == reflect.Ptr {
		rTyp = rTyp.Elem()
	}
	for _, name := range exclude {
		excludeMap[name] = struct{}{}
	}

	var (
		n     = rTyp.NumField()
		names = make([]string, 0, n)
	)
	for i := 0; i < n; i++ {
		var f = rTyp.Field(i)
		if _, ok := excludeMap[f.Name]; ok {
			continue
		}
		names = append(names, f.Name)
	}

	return names
}

func SetMany(d Definer, values map[string]interface{}) error {
	for name, value := range values {
		if err := assert.Err(set(d, name, value, false)); err != nil {
			return err
		}
	}
	return nil
}

func Set(d Definer, name string, value interface{}) error {
	return set(d, name, value, false)
}

func ForceSet(d Definer, name string, value interface{}) error {
	return set(d, name, value, true)
}

func Get[T any](d Definer, name string) T {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {

		var method, ok = Method[T](d, name)
		if ok {
			return method
		}

		assert.Fail(
			"get (%T): no field named %q",
			d, name,
		)
	}

	var v = f.GetValue()
	switch t := v.(type) {
	case T:
		return t
	case *T:
		return *t
	default:
		assert.Fail(
			"get (%T): field %q is not of type %T",
			d, name, v,
		)
	}
	return *(new(T))
}

func ToString(v any) string {
	if v == nil {
		return ""
	}

	//switch v := v.(type) {
	//case Stringer:
	//	return v.ToString()
	//case fmt.Stringer:
	//	return v.String()
	//}

	if stringer, ok := v.(Stringer); ok {
		return stringer.ToString()
	}

	return toString(v)
}

func toString(v any) string {
	switch v := v.(type) {
	case *mail.Address:
		return v.Address
	case time.Time:
		return v.Format(time.RFC3339)
	case fmt.Stringer:
		return v.String()
	case error:
		return v.Error()
	}

	var r = reflect.ValueOf(v)
	if r.Kind() == reflect.Ptr {
		r = r.Elem()
	}

	switch r.Kind() {
	case reflect.String:
		return r.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(r.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(r.Uint(), 10)
	case reflect.Float32:
		return strconv.FormatFloat(r.Float(), 'f', -1, 32)
	case reflect.Float64:
		return strconv.FormatFloat(r.Float(), 'f', -1, 64)
	case reflect.Bool:
		return strconv.FormatBool(r.Bool())
	case reflect.Slice, reflect.Array:
		var b = make([]string, r.Len())
		for i := 0; i < r.Len(); i++ {
			b[i] = ToString(r.Index(i).Interface())
		}
		return strings.Join(b, ", ")
	}

	return fmt.Sprintf("%v", v)
}

func Method[T any](obj interface{}, name string) (n T, ok bool) {
	if obj == nil {
		return n, false
	}

	var (
		v = reflect.ValueOf(obj)
		m = v.MethodByName(name)
	)
checkValid:
	if !m.IsValid() {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
			goto checkValid
		}
		return n, false
	}

	var i = m.Interface()
	if i == nil {
		return n, false
	}

	n, ok = i.(T)
	return n, ok
}

func set(d Definer, name string, value interface{}, force bool) error {
	var defs = d.FieldDefs()
	var f, ok = defs.Field(name)
	if !ok {
		return assert.Fail(
			fmt.Sprintf("set (%T): no field named %q", d, name),
		)
	}

	return f.SetValue(value, force)
}

func RConvert(v *reflect.Value, t reflect.Type) (*reflect.Value, bool) {
	var original = *v
	if !v.IsValid() {
		var z = reflect.New(t)
		*v = z
		return v, true
	}
	if v.Kind() == reflect.Ptr && t.Kind() != reflect.Ptr {
		*v = v.Elem()
	} else if v.Kind() != reflect.Ptr && t.Kind() == reflect.Ptr {
		var z = reflect.New(v.Type())
		z.Elem().Set(*v)
		*v = z
	}
	if v.Type().AssignableTo(t) {
		return v, true
	}
	if v.CanConvert(t) {
		*v = v.Convert(t)
		return v, true
	}
	*v = original
	return v, false
}

func rSet(src, dst *reflect.Value, isPointer bool) {
	if isPointer {
		if dst.IsZero() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst.Elem().Set(src.Elem())
	} else {
		dst.Set(*src)
	}
}

//func rMethodTakesArg(method reflect.Method, argType reflect.Type, index uint) bool {
//	var nArgs = method.Type.NumIn()
//	var i = int(index)
//	if nArgs != int(i+2) {
//		return false
//	}
//	return method.Type.In(int(i)).AssignableTo(argType)
//}

func RSet(src, dst *reflect.Value, convert bool) (canset bool) {
	if !src.IsValid() || !dst.IsValid() {
		return false
	}
	if !dst.CanSet() {
		return false
	}
	var isPointer = dst.Kind() == reflect.Ptr
	var isImmediatelyAssignable = src.Type().AssignableTo(dst.Type())
	if isImmediatelyAssignable {
		rSet(src, dst, isPointer)
		return true
	}
	if convert {
		src, canset = RConvert(src, dst.Type())
		if !canset {
			return false
		}
	}
	rSet(src, dst, isPointer)
	return true
}
