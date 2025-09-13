package contenttypes

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/text"
)

// ContentTypeDefinition is a struct that holds information about a model.
//
// This struct is used to register models with the ContentTypeRegistry.
//
// It is used to store information about a model, such as its human-readable name,
// description, and aliases.
//
// It allows for more flexibility to work with models that are not directly
// created by the framework developers, such as models from third-party apps.
type ContentTypeDefinition struct {
	// The model itself.
	//
	// This must be either a struct, or a pointer to a struct.
	ContentObject any

	// A function that returns the human-readable name of the model.
	//
	// This can be used to provide a custom name for the model.
	GetLabel func(ctx context.Context) string

	// A function to return a pluralized version of the model's name.
	//
	// This can be used to provide a custom plural name for the model.
	GetPluralLabel func(ctx context.Context) string

	// A function that returns a description of the model.
	//
	// This should return an accurate description of the model and what it represents.
	GetDescription func(ctx context.Context) string

	// A function which returns the label for an instance of the content type.
	//
	// This is used to get the human-readable name of an instance of the model.
	GetInstanceLabel func(any) string

	// A function that returns a new instance of the model.
	//
	// This should return a new instance of the model that can be safely typecast to the
	// correct model type.
	GetObject func() any

	// A function to retrieve an instance of the model by its ID.
	GetInstance func(context.Context, interface{}) (interface{}, error)

	// A function to get a list of instances of the model.
	GetInstances func(ctx context.Context, amount, offset uint) ([]interface{}, error)

	// A function to get a list of instances of the model by a list of IDs.
	//
	// Falls back to calling Instance for each ID if GetInstancesByID is not implemented.
	GetInstancesByIDs func(context.Context, []interface{}) ([]interface{}, error)

	// A list of aliases for the model.
	//
	// This can be used to provide additional names for the model and make it easier to
	// reference the model in code from the registry.
	//
	// For example, after a big refactor or renaming of a model, you can add the old name
	// as an alias to make it easier to reference the model in code.
	//
	// This should be the full type name of the model, including the package path.
	Aliases []string

	// The ContentType instance for the model.
	//
	// This is automatically generated from the ContentObject field.
	_cType ContentType
}

// Returns the ContentType instance for this model.
func (p *ContentTypeDefinition) ContentType() ContentType {
	if p._cType == nil {
		p._cType = NewContentType(p.ContentObject)
	}
	return p._cType
}

func (c *ContentTypeDefinition) Name() string {
	var rTyp = reflect.TypeOf(c.ContentObject)
	if rTyp.Kind() == reflect.Ptr {
		return rTyp.Elem().Name()
	}
	return rTyp.Name()
}

// Returns the model's human-readable name.
func (p *ContentTypeDefinition) Label(ctx context.Context) string {
	if p.GetLabel != nil {
		return p.GetLabel(ctx)
	}
	return trans.T(ctx, p.Name())
}

// Returns a description of the model and what it represents.
func (p *ContentTypeDefinition) Description(ctx context.Context) string {
	if p.GetDescription != nil {
		return p.GetDescription(ctx)
	}
	return ""
}

// Returns the pluralized version of the model's name.
func (p *ContentTypeDefinition) PluralLabel(ctx context.Context) string {
	if p.GetPluralLabel != nil {
		return p.GetPluralLabel(ctx)
	}
	return text.Pluralize(p.Label(ctx))
}

// Returns the human-readable name of an instance of the model.
func (p *ContentTypeDefinition) InstanceLabel(instance any) string {
	if p.GetInstanceLabel != nil {
		return p.GetInstanceLabel(instance)
	}

	if s, ok := instance.(fmt.Stringer); ok {
		var s = s.String()
		if s != "" {
			return s
		}
	}

	return fmt.Sprintf(
		"<Object %q>",
		p.ContentType().Model(),
	)
}

// Returns a new instance of the model.
//
// This method returns an object of type Any which can safely be typecast to the
// correct model type.
func (p *ContentTypeDefinition) Object() any {
	if p.GetObject != nil {
		return p.GetObject()
	}
	return p.ContentType().New()
}

// Returns an instance of the model by its ID.
func (p *ContentTypeDefinition) Instance(ctx context.Context, id interface{}) (interface{}, error) {
	if p.GetInstance != nil {
		return p.GetInstance(ctx, id)
	}
	assert.Fail("GetInstance not implemented for model %s", p.ContentType().TypeName())
	return nil, nil
}

// Returns a list of instances of the model.
func (p *ContentTypeDefinition) Instances(ctx context.Context, amount, offset uint) ([]interface{}, error) {
	if p.GetInstances != nil {
		return p.GetInstances(ctx, amount, offset)
	}
	assert.Fail("GetInstances not implemented for model %s", p.ContentType().TypeName())
	return nil, nil
}

// Returns a list of instances of the model by a list of IDs.
//
// Falls back to calling Instance for each ID if GetInstancesByID is not implemented.
func (p *ContentTypeDefinition) InstancesByIDs(ctx context.Context, ids []interface{}) ([]interface{}, error) {
	if p.GetInstancesByIDs != nil {
		return p.GetInstancesByIDs(ctx, ids)
	}

	var instancesCh = make(chan interface{}, len(ids))
	var errorsCh = make(chan error, len(ids))
	for _, id := range ids {
		var id = id
		go func(id interface{}) {
			var instance, err = p.Instance(ctx, id)
			if err != nil {
				errorsCh <- err
				return
			}
			instancesCh <- instance
		}(id)
	}

	var instances = make([]interface{}, 0, len(ids))
	var errs = make([]error, 0, len(ids))
	for i := 0; i < len(ids); i++ {
		select {
		case instance := <-instancesCh:
			instances = append(instances, instance)
		case err := <-errorsCh:
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return instances, nil
}
