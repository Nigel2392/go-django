package yml

import (
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

/*

Provides utilities for working with YAML files in Go.

This package includes:
- Utility types for deserialization of YAML data, such as [OrderedMap].
- Functions for iterating over YAML nodes and scanning them into Go types.
- Utility functions for reading YAML files and working with YAML nodes.

It uses the "gopkg.in/yaml.v3" package for YAML parsing.

*/

func Unmarshal(filePath string, into any, strict bool) error {
	var file, err = os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open YAML file %s: %w", filePath, err)
	}
	defer file.Close()

	return UnmarshalReader(file, into, strict)
}

func UnmarshalReader(r io.Reader, into any, strict bool) (err error) {
	var decoder = yaml.NewDecoder(r)

	decoder.KnownFields(strict)

	if err = decoder.Decode(into); err != nil {
		return fmt.Errorf("failed to unmarshal into %T: %w", into, err)
	}

	return nil
}
