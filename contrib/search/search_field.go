package search

type searchField struct {
	weight int8
	field  string
	lookup string
}

func NewSearchField(weight int8, field string, lookup string) SearchField {
	return &searchField{
		weight: weight,
		field:  field,
		lookup: lookup,
	}
}

func (f *searchField) Weight() int8 {
	return f.weight
}

func (f *searchField) Field() string {
	return f.field
}

func (f *searchField) Lookup() string {
	return f.lookup
}
