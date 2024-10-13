package cache

type errorType int

const (
	ErrItemNotFound errorType = iota
)

var errMap = map[errorType]string{
	ErrItemNotFound: "item not found",
}

func (e errorType) Error() string {
	return errMap[e]
}

func (e errorType) Is(target error) bool {
	t, ok := target.(errorType)
	if !ok {
		return false
	}
	return t == e
}
