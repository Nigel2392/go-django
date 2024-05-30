package assert

import (
	"fmt"
)

func Assert(cond bool, msg any, args ...interface{}) {

	if len(args) > 0 {
		if s, ok := msg.(string); !ok {
			msg = fmt.Sprint(append([]interface{}{msg}, args...))
		} else {
			msg = fmt.Sprintf(s, args...)
		}
	} else {
		if _, ok := msg.(string); !ok {
			msg = fmt.Sprint(msg)
		}
	}

	if !cond {
		panic(msg)
	}
}

func True(cond bool, msg any, args ...interface{}) {
	Assert(cond, msg, args...)
}

func False(cond bool, msg any, args ...interface{}) {
	Assert(!cond, msg, args...)
}
