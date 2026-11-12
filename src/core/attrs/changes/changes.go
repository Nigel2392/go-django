package changes

import (
	"context"
	"html/template"
	"reflect"

	"github.com/Nigel2392/go-django/contrib/admin/compare"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type Change struct {
	Name string        `json:"name"`
	HTML template.HTML `json:"html"`
}

func Changes(ctx context.Context, oldV, newV attrs.Definer) ([]Change, error) {
	if reflect.TypeOf(oldV) != reflect.TypeOf(newV) {
		return nil, errors.TypeMismatch.Wrapf("%T != %T", oldV, newV)
	}

	var changes = make([]Change, 0)
	for _, f := range attrs.Define(ctx, oldV).Fields() {
		cmp, err := compare.GetComparison(ctx, f.Type(), f.Label, f.Name(), oldV, newV)
		if err != nil && !errors.Is(err, errors.NotImplemented) {
			return nil, err
		}

		if err != nil {
			continue
		}

		changed, err := cmp.HasChanged()
		if err != nil {
			if !errors.Is(err, compare.ErrSkip) {
				return nil, err
			}

			continue
		}

		if !changed {
			continue
		}

		html, err := cmp.HTMLDiff()
		if err != nil {
			return nil, err
		}

		changes = append(changes, Change{
			Name: cmp.Label(),
			HTML: html,
		})
	}

	return changes, nil
}
