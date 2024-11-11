package trans

type TranslationBackend interface {
	Translate(v string, args ...any) string
}

var DefaultBackend TranslationBackend = &EchoBackend{}

func S(v string, args ...any) func() string {
	return func() string {
		return T(v, args...)
	}
}

func T(v string, args ...any) string {
	if len(args) == 0 {
		return v
	}
	return DefaultBackend.Translate(v, args...)
}
