package forms

import "fmt"

func assert(cond bool, msg string, args ...interface{}) {

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	if !cond {
		panic(msg)
	}
}

func assertTrue(cond bool, msg string, args ...interface{}) {
	assert(cond, msg, args...)
}

func assertFalse(cond bool, msg string, args ...interface{}) {
	assert(!cond, msg, args...)
}
