package encryption

import "context"

type additionalDataContextKey struct{}

func ContextWithAdditionalData(c context.Context, additionalData []byte) context.Context {
	return context.WithValue(c, additionalDataContextKey{}, additionalData)
}

func AdditionalDataFromContext(c context.Context) ([]byte, bool) {
	v, ok := c.Value(additionalDataContextKey{}).([]byte)
	return v, ok
}
