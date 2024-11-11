package trans

import "fmt"

type EchoBackend struct{}

func (b *EchoBackend) Translate(v string, args ...any) string {
	return fmt.Sprintf(v, args...)
}
