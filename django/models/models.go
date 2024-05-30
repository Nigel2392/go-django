package models

import "github.com/Nigel2392/django/core/attrs"

type Saver interface {
	Save() error
}

type Updater interface {
	Update() error
}

type Deleter interface {
	Delete() error
}

type Reloader interface {
	Reload() error
}

type Model interface {
	attrs.Definer
	Saver
	Updater
	Deleter
}
