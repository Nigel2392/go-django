package marshallers

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

type JSON struct{}

func (s *JSON) Serialize(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JSON) Deserialize(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

type YAML struct{}

func (s *YAML) Serialize(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func (s *YAML) Deserialize(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
