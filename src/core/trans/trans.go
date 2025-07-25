package trans

import "context"

type TranslationBackend interface {
	Translate(v string) string
	Translatef(v string, args ...any) string
}

var DefaultBackend TranslationBackend = &SprintBackend{}

func S(v string, args ...any) func(ctx context.Context) string {
	return func(ctx context.Context) string {
		return T(ctx, v, args...)
	}
}

func T(ctx context.Context, v string, args ...any) string {
	if len(args) == 0 {
		return DefaultBackend.Translate(v)
	}
	return DefaultBackend.Translatef(v, args...)
}
