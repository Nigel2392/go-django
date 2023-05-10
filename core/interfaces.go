package core

// Display a model field in the admin app list.
type DisplayableField interface {
	// Display a model field in the admin.
	Display() string
}

// How a model should be displayed in the admin list.
type ListDisplayer interface {
	ListDisplay() string
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
