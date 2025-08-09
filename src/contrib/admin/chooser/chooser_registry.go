package chooser

import (
	"reflect"

	"github.com/elliotchance/orderedmap/v2"
)

var choosers = orderedmap.NewOrderedMap[reflect.Type, Chooser]()

func Register(chooser Chooser) {
	var modelType = reflect.TypeOf(chooser.GetModel())
	if modelType == nil {
		panic("Chooser model type cannot be nil")
	}

	if _, exists := choosers.Get(modelType); exists {
		panic("Chooser already registered for model type: " + modelType.String())
	}

	choosers.Set(modelType, chooser)
}
