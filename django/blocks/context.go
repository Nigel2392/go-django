package blocks

import (
	"html/template"
	"net/http"

	"github.com/Nigel2392/django/core/ctx"
	"github.com/Nigel2392/django/core/tpl"
)

type BlockContext struct {
	Request_  *http.Request
	BlockDef  Block
	ID        string
	Name      string
	BlockHTML template.HTML
	Value     interface{}
	Errors    []error
	Attrs     map[string]string
	Context   ctx.Context
}

func NewBlockContext(b Block, context ctx.Context) *BlockContext {
	var (
		blockCtx *BlockContext
		r        *http.Request
	)

	if context != nil {
		switch ctx := context.(type) {
		case *BlockContext:
			blockCtx = ctx
		case tpl.RequestContext:
			r = ctx.Request()
		}
	}

	if blockCtx == nil {
		blockCtx = &BlockContext{
			BlockDef: b,
			Context:  context,
			Request_: r,
		}
	}

	return blockCtx
}

func (bc *BlockContext) Set(name string, value interface{}) {
	switch name {
	case "ID":
		bc.ID = value.(string)
	case "Block":
		bc.BlockDef = value.(Block)
	case "Name":
		bc.Name = value.(string)
	case "Value":
		bc.Value = value
	case "Errors":
		bc.Errors = value.([]error)
	case "Attrs":
		bc.Attrs = value.(map[string]string)
	case "Context":
		bc.Context = value.(ctx.Context)
	case "Request":
		bc.Request_ = value.(*http.Request)
	case "BlockHTML":
		bc.BlockHTML = value.(template.HTML)
	default:
		bc.Context.Set(name, value)
	}
}

func (bc *BlockContext) Get(name string) interface{} {
	switch name {
	case "ID":
		return bc.ID
	case "Block":
		return bc.BlockDef
	case "Name":
		return bc.Name
	case "Value":
		return bc.Value
	case "Errors":
		return bc.Errors
	case "Attrs":
		return bc.Attrs
	case "Context":
		return bc.Context
	case "Request":
		return bc.Request_
	case "BlockHTML":
		return bc.BlockHTML
	default:
		return bc.Context.Get(name)
	}
}

func (bc *BlockContext) Request() *http.Request {
	return bc.Request_
}
