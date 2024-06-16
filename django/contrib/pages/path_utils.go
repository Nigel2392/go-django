package pages

import (
	"fmt"
	"math"
)

var maxPathLen = int64(math.Pow(10, float64(STEP_LEN))) - 1

const STEP_LEN = 3
const ALPHABET = "0123456789"

func buildPathPart(numPreviousAncestors int64) string {
	if numPreviousAncestors < 0 {
		panic(ErrTooLittleAncestors)
	}

	numPreviousAncestors++

	if numPreviousAncestors > maxPathLen {
		panic(fmt.Errorf("numPreviousAncestors must be less than %d: %w", maxPathLen, ErrTooManyAncestors))
	}

	return fmt.Sprintf("%0*d", STEP_LEN, numPreviousAncestors)
}

func ancestorPath(path string, numAncestors int64) (string, error) {
	if numAncestors < 0 {
		return "", ErrTooLittleAncestors
	}

	if len(path)%STEP_LEN != 0 {
		return "", ErrInvalidPathLength
	}

	if numAncestors == 0 {
		return path, nil
	}

	if len(path) < int(numAncestors)*(STEP_LEN) {
		return "", ErrTooManyAncestors
	}

	return path[:len(path)-int(numAncestors)*(STEP_LEN)], nil
}
