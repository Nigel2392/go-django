package chooser

import (
	"fmt"
	"reflect"

	"github.com/elliotchance/orderedmap/v2"
)

const (
	DEFAULT_KEY = "default"
)

var choosers = orderedmap.NewOrderedMap[reflect.Type, *orderedmap.OrderedMap[string, chooser]]()

func Register(chsr chooser, key ...string) {

	var keyName = DEFAULT_KEY
	if len(key) > 0 {
		keyName = key[0]
	}

	var modelType = reflect.TypeOf(chsr.GetModel())
	if modelType == nil {
		panic("Chooser model type cannot be nil")
	}

	var definitionMap, ok = choosers.Get(modelType)
	if !ok {
		definitionMap = orderedmap.NewOrderedMap[string, chooser]()
		choosers.Set(modelType, definitionMap)
	}

	if !definitionMap.Set(keyName, chsr) {
		// replaced existing chooser for key
		panic(fmt.Sprintf(
			"Chooser already registered for model type %s with key %s",
			modelType.String(), keyName,
		))
	}
}
