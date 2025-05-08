package attrs

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type RelationType int

const (

	// ManyToOne is a many to one relationship, also known as a foreign key relationship.
	//
	// This means that the target model can have multiple instances of the source model,
	// but the source model can only have one instance of the target model.
	// This is the default type for a relation.
	RelManyToOne RelationType = iota

	// OneToOne is a one to one relationship.
	//
	// This means that the target model can only have one instance of the source model.
	// This is the default type for a relation.
	RelOneToOne

	// ManyToMany is a many to many relationship.
	//
	// This means that the target model can have multiple instances of the source model,
	// and the source model can have multiple instances of the target model.
	RelManyToMany

	// OneToMany is a one to many relationship, also known as a reverse foreign key relationship.
	//
	// This means that the target model can only have one instance of the source model,
	// but the source model can have multiple instances of the target model.
	RelOneToMany
)

var (
	// relationTypeNames is a map of relation types to their names.
	relationTypeNames = map[RelationType]string{
		RelManyToOne:  "ManyToOne",
		RelOneToOne:   "OneToOne",
		RelManyToMany: "ManyToMany",
		RelOneToMany:  "OneToMany",
	}

	// relationTypeValues is a map of relation type names to their values.
	relationTypeValues = map[string]RelationType{
		"ManyToOne":  RelManyToOne,
		"OneToOne":   RelOneToOne,
		"ManyToMany": RelManyToMany,
		"OneToMany":  RelOneToMany,
	}
)

func (r RelationType) String() string {
	return relationTypeNames[r]
}

func (r RelationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *RelationType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if v, ok := relationTypeValues[s]; ok {
		*r = v
		return nil
	}
	return errors.Errorf("invalid relation type: %s", s)
}
