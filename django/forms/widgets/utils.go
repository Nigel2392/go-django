package widgets

func S(v string) func() string {
	return func() string {
		return v
	}
}
