package cache

type cacheError int

func (e cacheError) Error() string {
	return cacheErrors[e]
}

const (
	ErrNotFound cacheError = iota
	ErrInvalidKey
	ErrInvalidValue
)

var cacheErrors = map[cacheError]string{
	ErrNotFound:     "item not found",
	ErrInvalidKey:   "invalid key",
	ErrInvalidValue: "invalid value",
}
