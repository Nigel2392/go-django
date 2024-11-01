package trans

import "fmt"

func S(v string, args ...any) func() string {
	return func() string {
		return T(v, args...)
	}
}

func T(v string, args ...any) string {
	if len(args) == 0 {
		return v
	}
	return fmt.Sprintf(v, args...)
}

func N(v string, args ...any) string {
	return fmt.Sprintf(v, args...)
}
