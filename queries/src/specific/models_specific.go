package specific

import (
	"context"

	"github.com/Nigel2392/go-django/src/core/attrs"
	pkgembed "github.com/Nigel2392/go-django/src/core/attrs/embedder"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

type Model interface {
	attrs.Definer
	SpecificObject() Specific[attrs.Definer]
	//	PK() any
	//	SpecificPK() any
	//	TargetContentTypeField() attrs.FieldDefinition
	//	TargetPrimaryField() attrs.FieldDefinition
	// Specific(context.Context) (attrs.Definer, error)
}

type Specific[EMBEDDER attrs.Definer] struct {
	SpecificID          string
	SpecificContentType *contenttypes.BaseContentType[attrs.Definer]
	Specific            attrs.Definer `proxy:"-"`
	embedder            EMBEDDER      `proxy:"-"`
}

func (n *Specific[EMBEDDER]) SpecificObject() Specific[attrs.Definer] {
	return Specific[attrs.Definer]{
		SpecificContentType: n.SpecificContentType,
		Specific:            n.Specific,
		embedder:            n.embedder,
	}
}

func (n *Specific[EMBEDDER]) SetupEmbedded(ctx context.Context, embedder EMBEDDER) {
	r := pkgembed.RootContext(ctx).Root
	if r == nil {
		panic("root object is nil")
	}

	n.embedder = embedder
	n.Specific = r.(attrs.Definer)
	n.SpecificContentType = contenttypes.NewContentType(n.Specific)
}

func (n *Specific[EMBEDDER]) PK() any {
	return attrs.PrimaryKey(context.Background(), n.embedder)
}

func (n *Specific[EMBEDDER]) SpecificPK() any {
	return attrs.PrimaryKey(context.Background(), n.Specific)
}

func (n *Specific[EMBEDDER]) TargetContentTypeField() attrs.FieldDefinition {
	var defs = n.embedder.FieldDefs(
		attrs.ContextWithFlags(context.Background(), attrs.CtxFlagRegistering, true),
	)
	var f, _ = defs.Field("ContentType")
	return f
}

func (n *Specific[EMBEDDER]) TargetPrimaryField() attrs.FieldDefinition {
	var defs = n.embedder.FieldDefs(
		attrs.ContextWithFlags(context.Background(), attrs.CtxFlagRegistering, true),
	)
	return defs.Primary()
}
