package fields

import "github.com/Nigel2392/versatile"

// For compatibility purposes this function stays defined here.
// IsZero checks if the value is zero or not.
func IsZero(value interface{}) bool {
	return versatile.IsZero(value)
}
