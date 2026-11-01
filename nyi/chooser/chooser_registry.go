package chooser

import (
	"fmt"

	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/elliotchance/orderedmap/v2"
)

const (
	DEFAULT_KEY = "default"
)

var choosers = orderedmap.NewOrderedMap[string, *orderedmap.OrderedMap[string, chooser]]()

func Register(chsr chooser, key ...string) {

	var keyName = DEFAULT_KEY
	if len(key) > 0 {
		keyName = key[0]
	}

	var modelType = contenttypes.NewContentType(chsr.GetModel())
	if modelType == nil {
		panic("Chooser model type cannot be nil")
	}

	var typeName = modelType.ShortTypeName()
	var definitionMap, ok = choosers.Get(typeName)
	if !ok {
		definitionMap = orderedmap.NewOrderedMap[string, chooser]()
		choosers.Set(typeName, definitionMap)
	}

	if !definitionMap.Set(keyName, chsr) {
		// replaced existing chooser for key
		panic(fmt.Sprintf(
			"Chooser already registered for model type %s with key %s",
			modelType.String(), keyName,
		))
	}
}
