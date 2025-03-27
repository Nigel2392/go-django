package trans

import "fmt"

type SprintBackend struct{}

func (b *SprintBackend) Translate(v string) string {
	return v
}

func (b *SprintBackend) Translatef(v string, args ...any) string {
	return fmt.Sprintf(v, args...)
}
