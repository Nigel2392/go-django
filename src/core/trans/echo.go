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
