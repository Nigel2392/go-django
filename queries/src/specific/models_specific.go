package specific

import (
	"context"
	"encoding/json"
	"reflect"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	pkgembed "github.com/Nigel2392/go-django/src/core/attrs/embedder"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/forms/fields"
	"github.com/Nigel2392/versatile"
)

type Model interface {
	attrs.Definer
	SpecificObject() *Specific[attrs.Definer]
}

type TModel[T any] interface {
	attrs.Definer
	Embedder() T

	SpecificObject() *Specific[attrs.Definer]
}

type Specific[EMBEDDER attrs.Definer] struct {
	SpecificID          string
	SpecificContentType *contenttypes.BaseContentType[attrs.Definer]
	Instance            attrs.Definer `proxy:"-"`
	embedder            attrs.Definer `proxy:"-"`
}

func (n *Specific[E]) SpecificObject() *Specific[attrs.Definer] {
	return (*Specific[attrs.Definer])(n)
}

func (n *Specific[E]) PK() any {
	return attrs.PrimaryKey(context.Background(), n.embedder)
}

func (n *Specific[E]) SpecificPK() any {
	if n.Instance == nil {
		if n.SpecificID == "" {
			return nil
		}

		var meta = attrs.GetModelMeta(n.SpecificContentType.New())
		var primary = meta.Primary()
		var pkVal = reflect.New(primary.Type())
		var err = json.Unmarshal([]byte(n.SpecificID), pkVal.Interface())
		if err != nil {
			panic("could not unmarshal pk")
		}

		return attrs.PrimaryKey(context.Background(), pkVal.Elem().Interface())
	}

	return attrs.PrimaryKey(context.Background(), n.Instance)
}

func (n *Specific[E]) Embedder() E {
	return n.embedder.(E)
}

func (n *Specific[E]) Fields(_ context.Context) []attrs.Field {
	return []attrs.Field{
		fattrs.Field(n.embedder.(E), "SpecificID", &n.SpecificID, func() fattrs.PtrFieldConfig[E, string] {
			return fattrs.PtrFieldConfig[E, string]{Config: attrs.FieldConfig{
				ReadOnly:  true,
				FormField: func(opts ...func(fields.Field)) fields.Field { return nil },
			}}
		}),
		fattrs.Field(n.embedder.(E), "SpecificContentType", &n.SpecificContentType, func() fattrs.PtrFieldConfig[E, *contenttypes.BaseContentType[attrs.Definer]] {
			return fattrs.PtrFieldConfig[E, *contenttypes.BaseContentType[attrs.Definer]]{Config: attrs.FieldConfig{
				ReadOnly:  true,
				FormField: func(opts ...func(fields.Field)) fields.Field { return nil },
			}}
		}),
	}
}

func resolveSpecificInstance(r any, embedder attrs.Definer, self *Specific[attrs.Definer]) attrs.Definer {
	if r == nil {
		return nil
	}
	rModel, ok := r.(Model)
	if !ok {
		return nil
	}
	if reflect.TypeOf(r) == reflect.TypeOf(embedder) {
		return nil
	}
	if rModel.SpecificObject() != self {
		return nil
	}
	return rModel
}

func (n *Specific[E]) SetupEmbedded(ctx context.Context, embedder attrs.Definer) error {
	r := pkgembed.RootContext(ctx).Root
	if r == nil && n.Instance == nil {
		panic("root object is nil")
	}

	n.embedder = embedder
	if inst := resolveSpecificInstance(r, embedder, n.SpecificObject()); inst != nil {
		n.Instance = inst
	} else if n.Instance == nil {
		return nil
	}

	if n.SpecificContentType == nil {
		n.SpecificContentType = contenttypes.NewContentType(n.Instance)
	}

	if n.SpecificID == "" && !attrs.ContextHasFlag(ctx, attrs.CtxFlagRegistering) {
		pk := attrs.PrimaryKey(ctx, attrs.Define(ctx, n.Instance).Primary())

		if versatile.IsZero(pk) {
			return nil
		}

		idBytes, err := json.Marshal(pk)
		if err != nil {
			return err
		}

		n.SpecificID = string(idBytes)
	}
	return nil
}

func (n *Specific[E]) PreSaveEmbedded(ctx context.Context, embedder attrs.Definer) error {
	r := pkgembed.RootContext(ctx).Root
	if r == nil && n.Instance == nil {
		panic("root object is nil")
	}

	n.embedder = embedder
	if inst := resolveSpecificInstance(r, embedder, n.SpecificObject()); inst != nil {
		n.Instance = inst
	} else if n.Instance == nil {
		return nil
	}

	if n.SpecificContentType == nil {
		n.SpecificContentType = contenttypes.NewContentType(n.Instance)
	}

	if n.SpecificID == "" {
		pk := attrs.PrimaryKey(ctx, attrs.Define(ctx, n.Instance).Primary())

		if versatile.IsZero(pk) {
			return nil
		}

		idBytes, err := json.Marshal(pk)
		if err != nil {
			return err
		}

		n.SpecificID = string(idBytes)
	}

	return nil
}

func (n *Specific[E]) Specific(ctx context.Context, refresh bool) (attrs.Definer, error) {
	if !refresh && n.Instance != nil {
		return n.Instance, nil
	}

	var (
		newObject = n.SpecificContentType.New().(Model)
		meta      = attrs.GetModelMeta(n.SpecificContentType.New())
		defs      = meta.Definitions()
		primary   = defs.Primary()
		reVal     = reflect.New(primary.Type())
		err       = json.Unmarshal([]byte(n.SpecificID), reVal.Interface())
	)

	if err != nil {
		return nil, errors.Wrapf(err, "Specific: %T -> %T: %v", n.embedder, newObject, n.SpecificID)
	}

	var querySet = queries.GetQuerySet(newObject).
		WithContext(ctx).
		Filter(primary.Name(), reVal.Elem().Interface())

	//	if withRelated {
	//		if len(definition.Prefetch.SelectRelated) > 0 {
	//			querySet = querySet.SelectRelated(definition.Prefetch.SelectRelated...)
	//		}
	//
	//		if len(definition.Prefetch.PrefetchRelated) > 0 {
	//			querySet = querySet.Preload(definition.Prefetch.PrefetchRelated...)
	//		}
	//	}

	row, err := querySet.Get()
	if err != nil {
		return nil, errors.NoResults.Wrapf("no results found in queryset for %T", newObject)
	}

	var ref = row.Object.SpecificObject()
	if ref == nil {
		// If the reference is nil, we cannot really do anything with it,
		// log a warning and return the row object.
		logger.Warnf(
			"specific: Specific cannot set reference for model %s with ID %v, reference pointer is nil",
			n.SpecificContentType.ShortTypeName(), reVal.Elem().Interface(),
		)
		return row.Object, nil
	}

	if ref.embedder != nil && n.embedder != nil {
		reflect.ValueOf(ref.embedder).Elem().Set(reflect.ValueOf(n.embedder).Elem())
		n.Instance = row.Object
	}

	*ref = *(*Specific[attrs.Definer])(n)
	return row.Object, nil
}
