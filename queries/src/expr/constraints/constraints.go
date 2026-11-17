package constraints

type ConstraintType uint

const (
	C_INVALID ConstraintType = iota
	C_PRIMARY_KEY
	C_UNIQUE
	C_FOREIGN_KEY
	C_CHECK
)

// Action is the action to take when the target model is deleted or updated.
type MutationAction int

func (a MutationAction) String() string {
	var action, ok = actions_map[a]
	if !ok {
		return "<ACTION_UNKNOWN>"
	}
	return action
}

const (
	// INVALID MutationAction = iota

	// OnDeleteRestrict is the action to restrict the delete of the target model.
	A_RESTRICT MutationAction = iota

	// OnDeleteCascade is the action to cascade the delete to the target model.
	A_CASCADE

	// OnDeleteSetNull is the action to set the field to null when the target model is deleted.
	A_SET_NULL

	// OnDeleteSetDefault is the action to set the field to the default value when the target model is deleted.
	A_SET_DEFAULT
)

var actions_map = map[MutationAction]string{
	A_RESTRICT:    "RESTRICT",
	A_SET_NULL:    "SET NULL",
	A_CASCADE:     "CASCADE",
	A_SET_DEFAULT: "SET DEFAULT",
}

type Constraint interface {
	ID() string
	Type() ConstraintType
	ErrorMessage() string
}

type Base struct {
	Identifier string
	Error      string
}
