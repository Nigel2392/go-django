package modeltree

import (
	"fmt"
	"math"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
)

var maxPathLen = int64(math.Pow(10, float64(STEP_LEN))) - 1

const (
	STEP_LEN = 3
	ALPHABET = "0123456789"
)

var (
	ErrInvalidPathLength  = errors.ValueError.Wrap("invalid path length")
	ErrTooLittleAncestors = errors.ValueError.Wrap("too little ancestors provided")
	ErrTooManyAncestors   = errors.ValueError.Wrap("too many ancestors provided")
)

func BuildPath(path []int64) (string, error) {
	if len(path) == 0 {
		return "", ErrTooLittleAncestors
	}

	if len(path) > int(maxPathLen) {
		return "", ErrTooManyAncestors
	}

	var pathParts = make([]byte, STEP_LEN*len(path))
	var fmtStr = fmt.Sprintf("%%0%dd", STEP_LEN)
	for i, p := range path {

		if i == 0 && len(path) == 1 && p == 0 {
			p++
		}

		if p <= 0 || p > maxPathLen {
			return "", fmt.Errorf("path part %d is out of bounds: %w", p, ErrInvalidPathLength)
		}

		copy(pathParts[i*STEP_LEN:], fmt.Sprintf(fmtStr, p))
	}

	return string(pathParts), nil
}

func ParsePath(path string) ([]int64, error) {
	if len(path)%STEP_LEN != 0 {
		return nil, ErrInvalidPathLength
	}

	var numParts = len(path) / STEP_LEN
	var parts = make([]int64, numParts)

	for i := 0; i < numParts; i++ {
		var part int64
		var _, err = fmt.Sscanf(path[i*STEP_LEN:(i+1)*STEP_LEN], "%3d", &part)
		if err != nil {
			return nil, fmt.Errorf("failed to parse path part %s: %w", path[i*STEP_LEN:(i+1)*STEP_LEN], err)
		}
		parts[i] = part
	}

	return parts, nil
}

func BuildNextPathPart(numPreviousAncestors int64) string {
	if numPreviousAncestors < 0 {
		panic(ErrTooLittleAncestors)
	}

	numPreviousAncestors++

	if numPreviousAncestors > maxPathLen {
		panic(fmt.Errorf("numPreviousAncestors must be less than %d: %w", maxPathLen, ErrTooManyAncestors))
	}

	var pathParts = make([]byte, STEP_LEN)
	for i := STEP_LEN - 1; i >= 0; i-- {
		pathParts[i] = ALPHABET[numPreviousAncestors%10]
		numPreviousAncestors /= 10
	}

	return string(pathParts)
}

func BuildNextPathPartFromFullPath(path string) (string, error) {
	if len(path)%STEP_LEN != 0 {
		return "", ErrInvalidPathLength
	}

	if len(path) < STEP_LEN {
		return "", ErrTooLittleAncestors
	}

	var length = 0
	var lastPart = path[len(path)-STEP_LEN:]
	var _, err = fmt.Sscanf(lastPart, "%3d", &length)
	if err != nil {
		return "", fmt.Errorf("failed to parse last path part %s: %w", lastPart, err)
	}

	if int64(length)+1 > maxPathLen {
		return "", ErrTooManyAncestors
	}

	return BuildNextPathPart(int64(length)), nil
}

func ParentPath(path string, depthUp int64) (string, error) {
	if depthUp < 0 {
		return "", ErrTooLittleAncestors
	}

	if len(path)%STEP_LEN != 0 {
		return "", ErrInvalidPathLength
	}

	if depthUp == 0 {
		return path, nil
	}

	if len(path) < int(depthUp)*(STEP_LEN) {
		return "", ErrTooManyAncestors
	}

	return path[:len(path)-int(depthUp)*(STEP_LEN)], nil
}
