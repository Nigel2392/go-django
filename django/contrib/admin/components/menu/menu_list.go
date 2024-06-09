package menu

type menuList struct {
	items []MenuItem
}

func NewItems() Items {
	return &menuList{}
}

func (i *menuList) All() []MenuItem {
	return i.items
}

func (i *menuList) Append(item MenuItem) {
	i.items = append(i.items, item)
}

func (i *menuList) Delete(name string) (ok bool) {
	for idx, item := range i.items {
		if item.Name() == name {
			i.items = append(i.items[:idx], i.items[idx+1:]...)
			return true
		}
	}
	return false
}
