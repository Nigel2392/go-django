package queries_test

import (
	"context"
	"testing"
	"time"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	var tables = quest.Table[*testing.T](nil,
		&ProxyModel{},
		&ProxiedModel{},
		&LinkedToProxiedModel{},
	)

	tables.Create()
}

type ProxyModel struct {
	models.Model
	ID          int64
	TargetID    int64
	TargetCType string
	Title       string
	Description string
}

func (b *ProxyModel) FieldDefs() attrs.Definitions {
	return b.Model.Define(b,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
		attrs.Unbound("TargetID"),
		attrs.Unbound("TargetCType"),
		attrs.Unbound("Title"),
		attrs.Unbound("Description"),
	)
}

type ProxiedModel struct {
	models.Model
	*ProxyModel
	// ProxyModel     *ProxyModel
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *ProxiedModel) FieldDefs() attrs.Definitions {
	return p.Model.Define(p,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
		// 	fields.OneToOne[*ProxyModel]("ProxyModel", &fields.FieldConfig{
		// 		IsProxy:  true,
		// 		Nullable: false,
		// 	}),
		attrs.Unbound("CreatedAt"),
		attrs.Unbound("UpdatedAt"),
	)
}

type LinkedToProxiedModel struct {
	models.Model
	ID           int64
	ProxiedModel *ProxiedModel
}

func (l *LinkedToProxiedModel) FieldDefs() attrs.Definitions {
	return l.Model.Define(l,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
		fields.ForeignKey[*ProxiedModel]("ProxiedModel", "proxied_model_id"),
	)
}

func mustDeleteAll(t *testing.T, modelTypes ...attrs.Definer) {
	for _, modelType := range modelTypes {
		var _, err = queries.GetQuerySet(modelType).
			WithContext(context.Background()).
			Delete()
		if err != nil {
			t.Fatalf("Failed to delete all records for model %T: %v", modelType, err)
		}
	}
}

func TestProxyModel(t *testing.T) {
	attrs.RegisterModel(&ProxyModel{})

	var proxyModel = models.Setup(&ProxiedModel{
		ProxyModel: &ProxyModel{
			Title:       "Test Proxy",
			Description: "This is a test proxy model",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	var ctx = context.Background()
	if err := proxyModel.Save(ctx); err != nil {
		t.Fatalf("Failed to save proxy model: %v", err)
	}

	var loadedModel, err = queries.GetQuerySet(&ProxiedModel{}).
		WithContext(ctx).
		Filter("ID", proxyModel.ID).
		First()
	if err != nil {
		t.Fatalf("Failed to load proxy model: %v", err)
	}
	if loadedModel == nil {
		t.Fatal("Expected to load a proxy model, but got nil")
	}
	if loadedModel.Object.ID != proxyModel.ID {
		t.Fatalf("Expected loaded model ID to be %d, but got %d", proxyModel.ID, loadedModel.Object.ID)
	}
	if loadedModel.Object.CreatedAt.IsZero() || loadedModel.Object.UpdatedAt.IsZero() {
		t.Fatal("Expected CreatedAt and UpdatedAt to be set, but they are zero values")
	}
	if loadedModel.Object.ProxyModel == nil {
		t.Fatal("Expected ProxyModel to be initialized, but it is nil")
	}
	//if loadedModel.Object.ProxyModel.ID != 1 {
	//	t.Fatalf("Expected TargetID to be %d, but got %d", 1, loadedModel.Object.ProxyModel.ID)
	//}
	if loadedModel.Object.ProxyModel.Title != "Test Proxy" {
		t.Fatalf("Expected ProxyModel Title to be 'Test Proxy', but got '%s'", loadedModel.Object.ProxyModel.Title)
	}
	if loadedModel.Object.ProxyModel.Description != "This is a test proxy model" {
		t.Fatalf("Expected ProxyModel Description to be 'This is a test proxy model', but got '%s'", loadedModel.Object.ProxyModel.Description)
	}
	//if loadedModel.Object.ProxyModel.TargetCType != contenttypes.NewContentType[attrs.Definer](loadedModel.Object).TypeName() {
	//	t.Fatalf("Expected TargetCType to be '%s', but got '%s'", contenttypes.NewContentType[attrs.Definer](loadedModel.Object).TypeName(), loadedModel.Object.ProxyModel.TargetCType)
	//}
	//if loadedModel.Object.ProxyModel.TargetID != loadedModel.Object.ID {
	//	t.Fatalf("Expected TargetID to be %d, but got %d", loadedModel.Object.ID, loadedModel.Object.ProxyModel.TargetID)
	//}

	mustDeleteAll(t, &LinkedToProxiedModel{}, &ProxiedModel{}, &ProxyModel{})
}

func TestProxyModelFilter(t *testing.T) {
	attrs.RegisterModel(&ProxyModel{})

	var proxyModel = models.Setup(&ProxiedModel{
		ProxyModel: &ProxyModel{
			Title:       "Test Proxy",
			Description: "This is a test proxy model",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	var ctx = context.Background()
	if err := proxyModel.Save(ctx); err != nil {
		t.Fatalf("Failed to save proxy model: %v", err)
	}

	var loadedModel, err = queries.GetQuerySet(&ProxiedModel{}).
		WithContext(ctx).
		Filter("ProxyModel.Title", "Test Proxy").
		First()
	if err != nil {
		t.Fatalf("Failed to load proxy model: %v", err)
	}
	if loadedModel == nil {
		t.Fatal("Expected to load a proxy model, but got nil")
	}
	if loadedModel.Object.ID != proxyModel.ID {
		t.Fatalf("Expected loaded model ID to be %d, but got %d", loadedModel.Object.ID, proxyModel.ID)
	}
	if loadedModel.Object.CreatedAt.IsZero() || loadedModel.Object.UpdatedAt.IsZero() {
		t.Fatal("Expected CreatedAt and UpdatedAt to be set, but they are zero values")
	}
	if loadedModel.Object.ProxyModel == nil {
		t.Fatal("Expected ProxyModel to be initialized, but it is nil")
	}
	//if loadedModel.Object.ProxyModel.ID != 1 {
	//	t.Fatalf("Expected TargetID to be %d, but got %d", 1, loadedModel.Object.ProxyModel.ID)
	//}
	if loadedModel.Object.ProxyModel.Title != "Test Proxy" {
		t.Fatalf("Expected ProxyModel Title to be 'Test Proxy', but got '%s'", loadedModel.Object.ProxyModel.Title)
	}
	if loadedModel.Object.ProxyModel.Description != "This is a test proxy model" {
		t.Fatalf("Expected ProxyModel Description to be 'This is a test proxy model', but got '%s'", loadedModel.Object.ProxyModel.Description)
	}
	//if loadedModel.Object.ProxyModel.TargetCType != contenttypes.NewContentType[attrs.Definer](loadedModel.Object).TypeName() {
	//	t.Fatalf("Expected TargetCType to be '%s', but got '%s'", contenttypes.NewContentType[attrs.Definer](loadedModel.Object).TypeName(), loadedModel.Object.ProxyModel.TargetCType)
	//}
	//if loadedModel.Object.ProxyModel.TargetID != loadedModel.Object.ID {
	//	t.Fatalf("Expected TargetID to be %d, but got %d", loadedModel.Object.ID, loadedModel.Object.ProxyModel.TargetID)
	//}

	mustDeleteAll(t, &LinkedToProxiedModel{}, &ProxiedModel{}, &ProxyModel{})
}

func TestLinkedToProxiedModel(t *testing.T) {

	var proxyModel = models.Setup(&ProxiedModel{
		ProxyModel: &ProxyModel{
			Title:       "Test Proxy",
			Description: "This is a test proxy model",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	var ctx = context.Background()
	if err := proxyModel.Save(ctx); err != nil {
		t.Fatalf("Failed to save proxy model: %v", err)
	}

	var linkedModel = models.Setup(&LinkedToProxiedModel{
		ProxiedModel: proxyModel,
	})
	if err := linkedModel.Save(ctx); err != nil {
		t.Fatalf("Failed to save linked model: %v", err)
	}

	var loadedLinkedModel, err = queries.GetQuerySet(&LinkedToProxiedModel{}).
		WithContext(ctx).
		Select("*", "ProxiedModel.*").
		Filter("ID", linkedModel.ID).
		First()
	if err != nil {
		t.Fatalf("Failed to load linked model: %v", err)
	}

	if loadedLinkedModel == nil {
		t.Fatal("Expected to load a linked model, but got nil")
	}

	if loadedLinkedModel.Object.ID != linkedModel.ID {
		t.Fatalf("Expected loaded linked model ID to be %d, but got %d", linkedModel.ID, loadedLinkedModel.Object.ID)
	}

	if loadedLinkedModel.Object.ProxiedModel == nil {
		t.Fatal("Expected ProxiedModel to be initialized in linked model, but it is nil")
	}

	if loadedLinkedModel.Object.ProxiedModel.ID != proxyModel.ID {
		t.Fatalf("Expected ProxiedModel ID to be %d, but got %d", proxyModel.ID, loadedLinkedModel.Object.ProxiedModel.ID)
	}

	if loadedLinkedModel.Object.ProxiedModel.ProxyModel == nil {
		t.Fatal("Expected ProxyModel to be initialized in ProxiedModel, but it is nil")
	}

	if loadedLinkedModel.Object.ProxiedModel.ProxyModel.Title != "Test Proxy" {
		t.Fatalf("Expected ProxyModel Title to be 'Test Proxy', but got '%s'", loadedLinkedModel.Object.ProxiedModel.ProxyModel.Title)
	}

	if loadedLinkedModel.Object.ProxiedModel.ProxyModel.Description != "This is a test proxy model" {
		t.Fatalf("Expected ProxyModel Description to be 'This is a test proxy model', but got '%s'", loadedLinkedModel.Object.ProxiedModel.ProxyModel.Description)
	}

	mustDeleteAll(t, &LinkedToProxiedModel{}, &ProxiedModel{}, &ProxyModel{})
}

func TestProxyFields(t *testing.T) {
	attrs.RegisterModel(&ProxyModel{})

	var proxiedModel = models.Setup(&ProxiedModel{
		ProxyModel: &ProxyModel{
			Title:       "Test Proxy",
			Description: "This is a test proxy model",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	var proxyFields = queries.ProxyFields(proxiedModel)
	if proxyFields == nil {
		t.Fatal("Expected proxy fields to be initialized, but got nil")
	}

	if proxyFields.FieldsLen() != 1 {
		t.Fatalf("Expected 1 proxy field, but got %d", proxyFields.FieldsLen())
	}

	var proxyField, ok = proxyFields.Get("ProxyModel")
	if !ok {
		t.Fatal("Expected to find proxy field with name 'ProxyModel'")
	}

	if proxyField.Name() != "ProxyModel" {
		t.Fatalf("Expected proxy field name to be 'ProxyModel', but got '%s'", proxyField.Name())
	}
}
