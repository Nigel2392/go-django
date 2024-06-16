package pages

import (
	"context"
	"fmt"
	"math"

	"github.com/Nigel2392/django/contrib/pages/models"
)

var querier models.DBQuerier
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

func CreateRootNode(ctx context.Context, q models.Querier, node *models.PageNode) error {
	if node.Path != "" {
		return fmt.Errorf("node path must be empty")
	}

	node.Path = buildPathPart(0)
	node.Depth = 0

	_, err := q.InsertNode(ctx, node.Title, node.Path, node.Depth, node.Numchild, node.PageID, node.Typehash)
	if err != nil {
		return err
	}

	return nil
}

func CreateChildNode(ctx context.Context, q models.DBQuerier, parent, child *models.PageNode) error {

	var tx, err = q.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	defer tx.Rollback()

	var queries = q.WithTx(tx)

	if parent.Path == "" {
		return fmt.Errorf("parent path must not be empty")
	}

	if child.Path != "" {
		return fmt.Errorf("child path must be empty")
	}

	child.Path = parent.Path + buildPathPart(parent.Numchild)
	child.Depth = parent.Depth + 1

	var id int64
	id, err = queries.InsertNode(ctx, child.Title, child.Path, child.Depth, child.Numchild, child.PageID, child.Typehash)
	if err != nil {
		return err
	}
	child.ID = id

	err = queries.UpdateNode(ctx, parent.Title, parent.Path, parent.Depth, parent.Numchild+1, parent.PageID, parent.Typehash, parent.ID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
