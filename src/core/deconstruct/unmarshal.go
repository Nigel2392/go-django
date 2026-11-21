package deconstruct

import "encoding/json/v2"

func UnmarshalDefault[OUT any, DATA ~[]byte](data DATA, opts ...json.Options) (OUT, error) {
	var d OUT
	var err = json.Unmarshal(data, &d, opts...)
	return d, err
}
