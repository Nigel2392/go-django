package users

import (
	"fmt"
	"net/http"
	"reflect"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/mux/middleware/authentication"
)

const APPVAR_USER_FROM_REQUEST = "USER_FROM_REQUEST"

func UserFromRequest(r *http.Request) authentication.User {
	var fn, ok = django.Global.Settings.Get(APPVAR_USER_FROM_REQUEST)
	if !ok {
		return &Base{}
	}

	var rv = reflect.ValueOf(fn)
	var typ = rv.Type()
	if rv.Kind() != reflect.Func {
		panic(fmt.Sprintf("expected setting %q to be of kind reflect.Func, got %s", APPVAR_USER_FROM_REQUEST, rv.Kind()))
	}

	if typ.NumIn() != 1 {
		panic(fmt.Sprintf("invalid function signature for setting %q, expected a single input parameter", APPVAR_USER_FROM_REQUEST))
	}

	if typ.NumOut() != 1 {
		panic(fmt.Sprintf("invalid function signature for setting %q, expected a single output parameter", APPVAR_USER_FROM_REQUEST))
	}

	var (
		in0  = typ.In(0)
		reqV = reflect.ValueOf(r)
	)

	if !reqV.Type().AssignableTo(in0) {
		panic(fmt.Sprintf("invalid function signature for setting %q, expected *http.Request input parameter", APPVAR_USER_FROM_REQUEST))
	}

	out := rv.Call([]reflect.Value{reflect.ValueOf(r)})
	val := out[0]

	if !val.IsValid() || val.IsNil() {
		return &Base{}
	}

	u, ok := val.Interface().(authentication.User)
	if !ok {
		return &Base{}
	}

	return u
}
