package trans

import (
	"context"
	"fmt"
)

type SprintBackend struct{}

func (b *SprintBackend) Translate(ctx context.Context, v string) string {
	return v
}

func (b *SprintBackend) Translatef(ctx context.Context, v string, args ...any) string {
	return fmt.Sprintf(v, args...)
}

func (b *SprintBackend) Locale(ctx context.Context) string {
	return ""
}

func (b *SprintBackend) Pluralize(ctx context.Context, singular, plural string, n int) string {
	if n == 1 {
		return singular
	}
	return plural
}

func (b *SprintBackend) Pluralizef(ctx context.Context, singular, plural string, n int, args ...any) string {
	return fmt.Sprintf(b.Pluralize(ctx, singular, plural, n), args...)
}
