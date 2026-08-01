package editor

import (
	"encoding/json"
	"reflect"
)

func JSONMarshalEditorData(d *EditorJSBlockData) ([]byte, error) {
	var data = ValueToForm(d)
	return json.Marshal(data)
}

func JSONUnmarshalEditorData(features []string, data []byte) (*EditorJSBlockData, error) {
	var d = new(EditorJSBlockData)
	var err = _JSONUnmarshalEditorData(d, features, data)
	return d, err
}

func _JSONUnmarshalEditorData(d *EditorJSBlockData, features []string, data []byte) error {
	var ed = new(EditorJSData)
	var err = json.Unmarshal(data, ed)
	if err != nil {
		return err
	}
	if len(features) == 0 {
		features = EditorRegistry.features.Keys()
	}
	editorData, err := ValueToGo(
		features, *ed,
	)
	if err != nil {
		return err
	}
	var rVal = reflect.ValueOf(d)
	var rPtr = reflect.Indirect(rVal)
	rPtr.Set(
		reflect.ValueOf(editorData).Elem(),
	)
	return nil
}
