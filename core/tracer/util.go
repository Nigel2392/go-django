package tracer

func containsMap[T1 comparable, T2 any](f map[T1]T2, o T1) bool {
	_, ok := f[o]
	return ok
}
