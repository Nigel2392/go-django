package editor

import "encoding/json"

func JSONMarshalEditorData(d *EditorJSBlockData) ([]byte, error) {
	var data = ValueToForm(d)
	return json.Marshal(data)
}

func JSONUnmarshalEditorData(features []string, data []byte) (*EditorJSBlockData, error) {
	var ed = new(EditorJSData)
	if err := json.Unmarshal(data, ed); err != nil {
		return nil, err
	}
	return ValueToGo(features, *ed)
}
