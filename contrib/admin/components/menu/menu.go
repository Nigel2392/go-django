package menu

import (
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/a-h/templ"
)

type MenuItem interface {
	// Order is used to sort the menu items
	//
	// The menu items are sorted in ascending order
	//
	// I.E. The menu item with the lowest order will be displayed first
	//
	// If two menu items have the same order, they will remain in the order they were added
	Order() int

	// IsShown is used to determine if the menu item should be displayed
	IsShown() bool

	// Implement a method for the templ.Component interface
	//
	// We explicitly only render the menu with the templ generated code.
	Component() templ.Component

	// Name is used to identify the menu item
	//
	// The name should be unique
	Name() string
}

type SidePanel interface {
	Order() int
	Name() string
	IsShown() bool
	Media() media.Media
	Label() string
	Icon() templ.Component
	Content() templ.Component
}
