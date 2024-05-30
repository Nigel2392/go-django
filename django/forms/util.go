package forms

import "fmt"

func assert(cond bool, msg any, args ...interface{}) {

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

func assertTrue(cond bool, msg any, args ...interface{}) {
	assert(cond, msg, args...)
}

func assertFalse(cond bool, msg any, args ...interface{}) {
	assert(!cond, msg, args...)
}
