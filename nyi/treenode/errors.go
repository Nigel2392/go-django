package treenode

import "github.com/Nigel2392/go-django/queries/src/drivers/errors"

var (
	ErrInvalidPathLength  = errors.ValueError.Wrap("invalid path length")
	ErrTooLittleAncestors = errors.ValueError.Wrap("too little ancestors provided")
	ErrTooManyAncestors   = errors.ValueError.Wrap("too many ancestors provided")
	ErrPageIsRoot         = errors.ValueError.Wrap("page is root")
)
