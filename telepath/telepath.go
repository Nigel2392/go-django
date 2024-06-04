package telepath

import (
	"github.com/Nigel2392/django/forms/media"
	"golang.org/x/exp/constraints"
)

var DICT_RESERVED_KEYS = []string{
	"_args",
	"_dict",
	"_id",
	"_list",
	"_ref",
	"_type",
	"_val",
}

const STRING_REF_MIN_LENGTH = 20 // Strings shorter than this will not be turned into references

type TelepathValue struct {
	Type string                 `json:"_type,omitempty"`
	Args []any                  `json:"_args,omitempty"`
	Dict map[string]interface{} `json:"_dict,omitempty"`
	List []interface{}          `json:"_list,omitempty"`
	Val  interface{}            `json:"_val,omitempty"`
	ID   int                    `json:"_id,omitempty"`
	Ref  int                    `json:"_ref,omitempty"`
}

type AdapterGetter interface {
	Adapter() Adapter
}

type Adapter interface {
	BuildNode(value interface{}, context Context) (Node, error)
}

type Context interface {
	AddMedia(media media.Media)
	BuildNode(value interface{}) (Node, error)
	Registry() *AdapterRegistry
}

// If this node is assigned an id, emit() should return the verbose representation with the
// id attached on first call, and a reference on subsequent calls. To disable this behaviour
// (e.g. for small primitive values where the reference representation adds unwanted overhead),
// set self.use_id = False.
type Node interface {
	Emit() any                  // emit (returns a dict representation of a value, this should be the main method used by an application.)
	EmitVerbose() TelepathValue // emit_verbose (returns a dict representation of a value that can have an _id attached)
	EmitCompact() any           // emit_compact (returns a compact representation of the value, in any JSON-serialisable type)
	GetValue() interface{}
	GetID() int
	SetID(id int)
	UseID() bool
}

type PrimitiveNodeValue interface {
	constraints.Integer | constraints.Float | bool
}

var GlobalRegistry = NewAdapterRegistry()
var NewContext = GlobalRegistry.Context
var Register = GlobalRegistry.Register
var Find = GlobalRegistry.Find
