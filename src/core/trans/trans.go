package trans

import "context"

type TranslationBackend interface {
	Translate(ctx context.Context, v string) string
	Translatef(ctx context.Context, v string, args ...any) string
	Locale(ctx context.Context) string
}

var DefaultBackend TranslationBackend = &SprintBackend{}

func S(v string, args ...any) func(ctx context.Context) string {
	return func(ctx context.Context) string {
		return T(ctx, v, args...)
	}
}

func T(ctx context.Context, v string, args ...any) string {
	if len(args) == 0 {
		return DefaultBackend.Translate(ctx, v)
	}
	return DefaultBackend.Translatef(ctx, v, args...)
}

func Locale(ctx context.Context) string {
	if DefaultBackend == nil {
		return ""
	}
	return DefaultBackend.Locale(ctx)
}
