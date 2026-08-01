package pages

import (
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

const (
	ErrCodeNoPage   errors.GoCode = "NoPage"
	ErrCodeNoPageID errors.GoCode = "NoPageID"
)

var (
	ErrNoPage             = errors.New(ErrCodeNoPage, "pages: PageNode has no Page set")
	ErrNoPageID           = errors.New(ErrCodeNoPageID, "pages: PageNode has no PageID set", ErrNoPage)
	ErrInvalidPathLength  = errors.ValueError.Wrap("invalid path length")
	ErrTooLittleAncestors = errors.ValueError.Wrap("too little ancestors provided")
	ErrTooManyAncestors   = errors.ValueError.Wrap("too many ancestors provided")
	ErrPageIsRoot         = errors.ValueError.Wrap("page is root")
)
