package flags

import (
	"fmt"
	"strings"
)

type listIndex = int

type List struct {
	data []string
	map_ map[string]listIndex
}

func NewList(apps ...string) List {
	var i = &List{
		data: make([]string, 0, len(apps)),
		map_: make(map[string]listIndex, len(apps)),
	}

	for _, app := range apps {
		if err := i.Set(app); err != nil {
			panic(fmt.Errorf("error setting app %s: %w", app, err))
		}
	}

	return *i
}

func (i *List) String() string {
	return strings.Join(i.data, ", ")
}

func (i *List) Len() int {
	if i.data == nil {
		return 0
	}
	return len(i.data)
}

func (i *List) List() []string {
	if i.data == nil {
		return nil
	}
	return i.data
}

func (i *List) Set(value string) error {
	if i.map_ == nil {
		i.map_ = make(map[string]listIndex)
	}
	if i.data == nil {
		i.data = make([]string, 0)
	}
	if _, ok := i.map_[value]; ok {
		return fmt.Errorf("app %s already set", value)
	}
	i.map_[value] = len(i.data)
	i.data = append(i.data, value)
	return nil
}

func (i *List) Lookup(name string) (listIndex, bool) {
	if i.map_ == nil {
		return -1, false
	}
	index, ok := i.map_[name]
	return index, ok
}
