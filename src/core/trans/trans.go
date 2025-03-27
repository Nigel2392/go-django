package trans

type TranslationBackend interface {
	Translate(v string) string
	Translatef(v string, args ...any) string
}

var DefaultBackend TranslationBackend = &SprintBackend{}

func S(v string, args ...any) func() string {
	return func() string {
		return T(v, args...)
	}
}

func T(v string, args ...any) string {
	if len(args) == 0 {
		return DefaultBackend.Translate(v)
	}
	return DefaultBackend.Translatef(v, args...)
}
