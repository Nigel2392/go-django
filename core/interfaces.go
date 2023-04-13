package core

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	"gorm.io/gorm"
)

// Search the database for the model.
//
// Provide a function to modify the query.
//
// Do not execute the query!
type AdminSearchField interface {
	// Provide a `tx.Where`
	AdminSearch(query string, tx *gorm.DB) *gorm.DB
}

// Validate a model field in adminsite forms.
//
// This is an interface type which is to be applied to struct fields.
//
// The Validate method is called when the model is saved to the admin site.
type FieldValidator interface {
	Validate(any) error
}

// Display the model in the admin.
type DisplayableModel interface {
	// Display the model in the admin.
	String() string
}

// Display a model field in the admin.
type DisplayableField interface {
	// Display a model field in the admin.
	Display() string
}

// How a model should be displayed in the admin list.
type ListDisplayer interface {
	ListDisplay() string
}

// Get the absolute URL of the model.
type AbsoluteURLModel interface {
	// Get the absolute URL of the model.
	AbsoluteURL() string
}

// Namer to retrieve the model name, from any given model.
//
// This is used in the admin site for example.
//
// This is not the name to display for a given instance of the model!
//
// -> modelutils.namer.GetModelName
type Namer interface {
	NameOf() string
}

// Namer to retrieve the app name, from any given model.
//
// -> modelutils.namer.GetAppName
type AppNamer interface {
	AppName() string
}

// How a model value should be transformed from a string to the type in question.
type FromStringer interface {
	FromString(string) error
}

// BaseField is the interface that all database fields should implement.
//
// This is used in the admin site for example.
type BaseField interface {
	fmt.Stringer
	driver.Valuer
	sql.Scanner
	FromStringer
}
