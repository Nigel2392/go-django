package menu

type MenuURL struct {
	WholeURL string
	URLPart  string
}

type Items []*Item

type Item struct {
	// The name of the menu item
	Name string
	// The URL of the menu item
	URL *MenuURL
	// Custom data for the menu item
	Data any
	// Children of the menu item
	children Items

	// The weight of the menu item
	Weight int
}

func NewItem(name, wholeURL, urlPart string, weight int, data ...any) *Item {
	var d any
	if len(data) > 0 {
		d = data[0]
	}
	return &Item{
		Name: name,
		URL: &MenuURL{
			WholeURL: wholeURL,
			URLPart:  urlPart,
		},
		Data:   d,
		Weight: weight,
	}
}

func (m *Item) Children(add ...*Item) Items {
	if m.children == nil {
		m.children = make([]*Item, 0)
	}
	if len(add) > 0 {
		m.children = append(m.children, add...)
	}
	return m.children
}

func (m *Item) ForEach(f func(*Item)) {
	f(m)
	for _, child := range m.children {
		child.ForEach(f)
	}
}
