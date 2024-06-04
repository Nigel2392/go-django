package telepath

type UnpackableError struct {
	Err error
	Obj Node
}

func (e *UnpackableError) Error() string {
	return e.Err.Error()
}
