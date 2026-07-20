package bitch

import "golang.org/x/exp/constraints"

/*
	package bitch provides functions for bit checking on flag-like types.
*/

type Flag uint // use uint in case someone's developing for MS DOS :^)

func (o Flag) Is(f Flag) bool { return o&f == f }

func (o Flag) Isnt(f Flag) bool { return o&f == 0 }

func (o *Flag) Set(f Flag, val bool) Flag {
	if val {
		return *o | f // Set the bit(s)
	}
	return *o &^ f // Clear the bit(s)
}

func Is[T constraints.Unsigned](src, chk T) bool {
	return src&chk == chk
}
func Isnt[T constraints.Unsigned](src, chk T) bool {
	return src&chk == 0
}
func Set[T constraints.Unsigned](src, chk T, val bool) T {
	if val {
		return src | chk // Set the bit(s)
	}
	return src &^ chk // Clear the bit(s)
}
