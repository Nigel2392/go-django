package queries_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/djester/quest"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/contenttypes"
)

func init() {
	var tables = quest.Table[*testing.T](nil,
		&ProxyModel{},
		&ProxiedModel{},
		&LinkedToProxiedModel{},
		&OtherProxiedModel{},
		&OneToOneProxiedModel{},
	)

	tables.Create()
}

type ProxyModel struct {
	models.Model `ctype:"TargetCType" target:"TargetID"`
	ID           int64
	TargetID     int64
	TargetCType  string
	Title        string
	Description  string
}

func (b *ProxyModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return b.Model.Define(ctx, b,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		attrs.Unbound("TargetID"),
		attrs.Unbound("TargetCType"),
		attrs.Unbound("Title"),
		attrs.Unbound("Description"),
	)
}

type proxyValidationErrorKey struct{}

func (b *ProxyModel) Validate(ctx context.Context) error {
	if err, ok := ctx.Value(proxyValidationErrorKey{}).(error); ok {
		return err
	}
	return b.Model.Validate(ctx)
}

type ProxiedModel struct {
	models.Model
	*ProxyModel
	ID        int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *ProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		attrs.Unbound("CreatedAt"),
		attrs.Unbound("UpdatedAt"),
	)
}

type OtherProxiedModel struct {
	models.Model
	ID    int64
	Label string
	Base  *ProxyModel `proxy:"true" auto:"true" ctype:"TargetCType" target:"TargetID"`
}

func (p *OtherProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		attrs.Unbound("Label"),
	)
}

type OneToOneProxiedModel struct {
	models.Model
	ID   int64
	Base *ProxyModel `proxy:"-"`
}

func (p *OneToOneProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return p.Model.Define(ctx, p,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		fields.OneToOne[*ProxyModel]("Base", &fields.FieldConfig{
			IsProxy:    true,
			AllowEdit:  true,
			ColumnName: "base_id",
		}),
	)
}

type LinkedToProxiedModel struct {
	models.Model
	ID           int64
	ProxiedModel *ProxiedModel
}

func (l *LinkedToProxiedModel) FieldDefs(ctx context.Context) attrs.Definitions {
	return l.Model.Define(ctx, l,
		attrs.Unbound("ID", &attrs.Config{Primary: true}),
		fields.ForeignKey[*ProxiedModel]("ProxiedModel", "proxied_model_id"),
	)
}

func isolateProxyRows(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		for _, model := range []attrs.Definer{
			&LinkedToProxiedModel{}, &OneToOneProxiedModel{}, &OtherProxiedModel{}, &ProxiedModel{}, &ProxyModel{},
		} {
			if _, err := queries.GetQuerySet(model).Delete(); err != nil {
				t.Errorf("clean up %T: %v", model, err)
			}
		}
	})
}

func saveProxyModel(t *testing.T, title string) *ProxiedModel {
	t.Helper()
	obj := models.Setup(t.Context(), &ProxiedModel{
		ProxyModel: &ProxyModel{Title: title, Description: "Description for " + title},
		CreatedAt:  time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:  time.Date(2025, 1, 3, 3, 4, 5, 0, time.UTC),
	})
	if err := obj.Save(t.Context()); err != nil {
		t.Fatalf("save %q: %v", title, err)
	}
	if obj.ID == 0 || obj.ProxyModel.ID == 0 {
		t.Fatalf("save did not assign both primary keys: embedder=%d proxy=%d", obj.ID, obj.ProxyModel.ID)
	}
	return obj
}

func assertLoadedProxy(t *testing.T, got, want *ProxiedModel) {
	t.Helper()
	if got == nil || got.ProxyModel == nil {
		t.Fatalf("expected a loaded embedder and proxy, got %+v", got)
	}
	if got.ID != want.ID || !got.CreatedAt.Equal(want.CreatedAt) || !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("embedder = (%d, %s, %s), want (%d, %s, %s)",
			got.ID, got.CreatedAt, got.UpdatedAt, want.ID, want.CreatedAt, want.UpdatedAt)
	}
	if got.ProxyModel.ID != want.ProxyModel.ID || got.Title != want.Title || got.Description != want.Description {
		t.Errorf("proxy = (%d, %q, %q), want (%d, %q, %q)",
			got.ProxyModel.ID, got.Title, got.Description, want.ProxyModel.ID, want.Title, want.Description)
	}
	assertProxyTarget(t, got.ProxyModel, want, want.ID)
}

func assertProxyTarget(t *testing.T, proxy *ProxyModel, target attrs.Definer, id int64) {
	t.Helper()
	wantType := contenttypes.NewContentType[attrs.Definer](target).TypeName()
	if proxy.TargetID != id || proxy.TargetCType != wantType {
		t.Errorf("proxy target = (%d, %q), want (%d, %q)", proxy.TargetID, proxy.TargetCType, id, wantType)
	}
}

func TestProxyModelRoundTrip(t *testing.T) {
	isolateProxyRows(t)
	want := saveProxyModel(t, "Original")
	assertProxyTarget(t, want.ProxyModel, want, want.ID)

	row, err := queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{}).Filter("ID", want.ID).Get()
	if err != nil {
		t.Fatal(err)
	}
	assertLoadedProxy(t, row.Object, want)

	row.Object.Title = "Revised"
	row.Object.Description = "Revised description"
	row.Object.UpdatedAt = want.UpdatedAt.AddDate(0, 0, 1)
	// The loaded proxy pointer stays the same when its fields are edited in place.
	config := models.SaveConfig{IncludeFields: []string{"ProxyModel", "UpdatedAt"}}
	if err := row.Object.SaveObject(t.Context(), config); err != nil {
		t.Fatalf("save updated embedder and proxy: %v", err)
	}
	want.Title = "Revised"
	want.Description = "Revised description"
	want.UpdatedAt = want.UpdatedAt.AddDate(0, 0, 1)
	reloaded, err := queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{}).Filter("ID", want.ID).Get()
	if err != nil {
		t.Fatal(err)
	}
	assertLoadedProxy(t, reloaded.Object, want)

	proxies, err := queries.GetQuerySetWithContext(t.Context(), &ProxyModel{}).
		Filter("TargetID", want.ID).Filter("TargetCType", want.TargetCType).Count()
	if err != nil || proxies != 1 {
		t.Fatalf("proxy count after update = %d, error = %v, want 1", proxies, err)
	}
}

func TestProxyModelQueries(t *testing.T) {
	isolateProxyRows(t)
	alpha := saveProxyModel(t, "Alpha")
	zulu := saveProxyModel(t, "Zulu")

	for _, tc := range []struct {
		name  string
		query func(*queries.QuerySet[*ProxiedModel]) *queries.QuerySet[*ProxiedModel]
		want  []*ProxiedModel
	}{
		{"implicit selection", func(q *queries.QuerySet[*ProxiedModel]) *queries.QuerySet[*ProxiedModel] {
			return q.OrderBy("ID")
		}, []*ProxiedModel{alpha, zulu}},
		{"filter on proxy", func(q *queries.QuerySet[*ProxiedModel]) *queries.QuerySet[*ProxiedModel] {
			return q.Filter("ProxyModel.Title", zulu.Title)
		}, []*ProxiedModel{zulu}},
		{"filter on both models", func(q *queries.QuerySet[*ProxiedModel]) *queries.QuerySet[*ProxiedModel] {
			return q.Filter("ID", alpha.ID).Filter("ProxyModel.Description", zulu.Description)
		}, nil},
		{"sort on proxy", func(q *queries.QuerySet[*ProxiedModel]) *queries.QuerySet[*ProxiedModel] {
			return q.OrderBy("-ProxyModel.Title")
		}, []*ProxiedModel{zulu, alpha}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rows, err := tc.query(queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{})).All()
			if err != nil {
				t.Fatal(err)
			}
			if len(rows) != len(tc.want) {
				t.Fatalf("got %d rows, want %d", len(rows), len(tc.want))
			}
			for i, want := range tc.want {
				assertLoadedProxy(t, rows[i].Object, want)
			}
		})
	}
}

func TestProxyModelJoinUsesBothTargetKeys(t *testing.T) {
	isolateProxyRows(t)
	want := saveProxyModel(t, "Matching target")
	wrongType := contenttypes.NewContentType[attrs.Definer](&OtherProxiedModel{}).TypeName()
	for _, decoy := range []*ProxyModel{
		{TargetID: want.ID, TargetCType: wrongType, Title: "Wrong content type"},
		{TargetID: want.ID + 1, TargetCType: want.TargetCType, Title: "Wrong target ID"},
	} {
		if err := models.Setup(t.Context(), decoy).Save(t.Context()); err != nil {
			t.Fatalf("save decoy %q: %v", decoy.Title, err)
		}
	}

	row, err := queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{}).Filter("ID", want.ID).Get()
	if err != nil {
		t.Fatal(err)
	}
	assertLoadedProxy(t, row.Object, want)

	deleted, err := queries.GetQuerySetWithContext(t.Context(), &ProxyModel{}).Filter("ID", want.ProxyModel.ID).Delete()
	if err != nil || deleted != 1 {
		t.Fatalf("deleting matching proxy: affected=%d error=%v", deleted, err)
	}
	rows, err := queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{}).Filter("ID", want.ID).All()
	if err != nil || len(rows) != 0 {
		t.Fatalf("querying an embedder without a matching proxy: rows=%d error=%v", len(rows), err)
	}
}

func TestNamedAutoProxyRoundTrip(t *testing.T) {
	isolateProxyRows(t)
	want := models.Setup(t.Context(), &OtherProxiedModel{Label: "Named embedder"})
	if want.Base == nil {
		t.Fatal("setup did not allocate the named auto proxy")
	}
	want.Base.Title = "Named proxy"
	if err := want.Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	assertProxyTarget(t, want.Base, want, want.ID)
	row, err := queries.GetQuerySetWithContext(t.Context(), &OtherProxiedModel{}).
		Filter("Base.Title", want.Base.Title).Get()
	if err != nil {
		t.Fatal(err)
	}
	if got := row.Object; got.Base == nil || got.ID != want.ID || got.Label != want.Label || got.Base.ID != want.Base.ID || got.Base.Title != want.Base.Title {
		t.Fatalf("wrong named proxy loaded: embedder=(%d, %q) proxy=%+v", got.ID, got.Label, got.Base)
	}
	assertProxyTarget(t, row.Object.Base, want, want.ID)
}

func TestProxyModelQueriesFromSharedTable(t *testing.T) {
	isolateProxyRows(t)
	ctx := t.Context()
	cutoff := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	makeProxied := func(title string, createdAt time.Time) *ProxiedModel {
		t.Helper()
		obj := models.Setup(ctx, &ProxiedModel{
			ProxyModel: &ProxyModel{Title: title},
			CreatedAt:  createdAt,
			UpdatedAt:  createdAt,
		})
		if err := obj.Save(ctx); err != nil {
			t.Fatalf("save %q: %v", title, err)
		}
		return obj
	}
	makeProxied("A: too old", cutoff.AddDate(0, 0, -1))
	older := makeProxied("B: first match", cutoff)
	newer := makeProxied("C: newest match", cutoff.AddDate(0, 0, 1))
	other := models.Setup(ctx, &OtherProxiedModel{Label: "Featured"})
	other.Base.Title = "D: another type"
	if err := other.Save(ctx); err != nil {
		t.Fatal(err)
	}
	decoy := models.Setup(ctx, &ProxyModel{
		Title:       "E: same ID, wrong type",
		TargetID:    newer.ID,
		TargetCType: contenttypes.NewContentType[attrs.Definer](&LinkedToProxiedModel{}).TypeName(),
	})
	if err := decoy.Save(ctx); err != nil {
		t.Fatal(err)
	}

	base := queries.GetQuerySetWithContext(ctx, &ProxyModel{})
	proxiedIDs := queries.GetQuerySetWithContext(ctx, &ProxiedModel{}).
		Select("ID").ClearOrderBy().Limit(0).Filter("CreatedAt__gte", cutoff)
	proxiedMatch := expr.And(
		expr.Q("TargetCType", older.TargetCType),
		expr.Q("TargetID__in", queries.Subquery(proxiedIDs)),
	)

	t.Run("filter before counting and paginating", func(t *testing.T) {
		matching := base.Filter(proxiedMatch)
		count, err := matching.Count()
		if err != nil || count != 2 {
			t.Fatalf("matching count = %d, error = %v, want 2", count, err)
		}
		rows, err := matching.OrderBy("Title").Limit(1).Offset(1).All()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].Object.ID != newer.ProxyModel.ID {
			t.Fatalf("second page = %v, want the proxy for %q", rows, newer.Title)
		}
	})

	t.Run("combine different concrete types", func(t *testing.T) {
		otherIDs := queries.GetQuerySetWithContext(ctx, &OtherProxiedModel{}).
			Select("ID").ClearOrderBy().Limit(0).Filter("Label", "Featured")
		otherMatch := expr.And(
			expr.Q("TargetCType", other.Base.TargetCType),
			expr.Q("TargetID__in", queries.Subquery(otherIDs)),
		)
		rows, err := base.Filter(expr.Or(proxiedMatch, otherMatch)).OrderBy("Title").All()
		if err != nil {
			t.Fatal(err)
		}
		var got []int64
		for _, row := range rows {
			got = append(got, row.Object.ID)
		}
		want := []int64{older.ProxyModel.ID, newer.ProxyModel.ID, other.Base.ID}
		if !slices.Equal(got, want) {
			t.Errorf("proxy IDs = %v, want %v", got, want)
		}
	})

	t.Run("order by a concrete field", func(t *testing.T) {
		createdAt := queries.GetQuerySetWithContext(ctx, &ProxiedModel{}).
			Select("CreatedAt").Filter("ID", expr.OuterRef("TargetID")).Limit(1)
		rows, err := base.Filter(proxiedMatch).
			Select("*").
			Annotate("ConcreteCreatedAt", queries.Subquery(createdAt)).
			OrderBy("-ConcreteCreatedAt").Limit(1).All()
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].Object.ID != newer.ProxyModel.ID {
			t.Fatalf("newest match = %v, want the proxy for %q", rows, newer.Title)
		}
	})
}

func TestLinkedToProxiedModel(t *testing.T) {
	isolateProxyRows(t)
	want := saveProxyModel(t, "Linked")
	other := saveProxyModel(t, "Other link")
	var linkedID int64
	for _, parent := range []*ProxiedModel{want, other} {
		link := models.Setup(t.Context(), &LinkedToProxiedModel{ProxiedModel: parent})
		if err := link.Save(t.Context()); err != nil {
			t.Fatal(err)
		}
		if parent == want {
			linkedID = link.ID
		}
	}

	t.Run("wildcard selection", func(t *testing.T) {
		row, err := queries.GetQuerySetWithContext(t.Context(), &LinkedToProxiedModel{}).
			Select("*", "ProxiedModel.*").Filter("ID", linkedID).Get()
		if err != nil {
			t.Fatal(err)
		}
		if row.Object.ID != linkedID {
			t.Errorf("linked ID = %d, want %d", row.Object.ID, linkedID)
		}
		assertLoadedProxy(t, row.Object.ProxiedModel, want)
	})

	t.Run("select related and filter on proxy", func(t *testing.T) {
		row, err := queries.GetQuerySetWithContext(t.Context(), &LinkedToProxiedModel{}).
			SelectRelated("ProxiedModel").Filter("ProxiedModel.ProxyModel.Title", want.Title).Get()
		if err != nil {
			t.Fatal(err)
		}
		if row.Object.ID != linkedID {
			t.Errorf("linked ID = %d, want %d", row.Object.ID, linkedID)
		}
		assertLoadedProxy(t, row.Object.ProxiedModel, want)
	})
}

func TestExplicitOneToOneProxy(t *testing.T) {
	isolateProxyRows(t)
	base := models.Setup(t.Context(), &ProxyModel{Title: "Explicit base"})
	if err := base.Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := models.Setup(t.Context(), &OneToOneProxiedModel{Base: base})
	if err := want.Save(t.Context()); err != nil {
		t.Fatal(err)
	}
	row, err := queries.GetQuerySetWithContext(t.Context(), &OneToOneProxiedModel{}).
		Filter("Base.Title", base.Title).Get()
	if err != nil {
		t.Fatal(err)
	}
	got := row.Object
	if got.Base == nil || got.ID != want.ID || got.Base.ID != base.ID || got.Base.Title != base.Title {
		t.Fatalf("wrong one-to-one proxy loaded: embedder=%d proxy=%+v", got.ID, got.Base)
	}
}

func TestProxyModelSaveFailureRollsBackEmbedder(t *testing.T) {
	isolateProxyRows(t)
	rejected := errors.New("proxy validation rejected")
	now := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	obj := models.Setup(t.Context(), &ProxiedModel{
		ProxyModel: &ProxyModel{Title: "Rejected"},
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	ctx := context.WithValue(t.Context(), proxyValidationErrorKey{}, rejected)
	if err := obj.Save(ctx); !errors.Is(err, rejected) {
		t.Fatalf("save error = %v, want %v", err, rejected)
	}

	// Project only the embedder's primary key so an inner join cannot hide a
	// partially saved embedder from the assertion.
	embedders, err := queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{}).Select("ID").Count()
	if err != nil || embedders != 0 {
		t.Errorf("embedder count after rollback = %d, error = %v, want 0", embedders, err)
	}
	proxies, err := queries.GetQuerySetWithContext(t.Context(), &ProxyModel{}).Count()
	if err != nil || proxies != 0 {
		t.Errorf("proxy count after rollback = %d, error = %v, want 0", proxies, err)
	}
}

func TestProxyModelUpdateFailurePreservesDatabaseState(t *testing.T) {
	isolateProxyRows(t)
	want := saveProxyModel(t, "Before failed update")
	query := func() *queries.QuerySet[*ProxiedModel] {
		return queries.GetQuerySetWithContext(t.Context(), &ProxiedModel{}).Filter("ID", want.ID)
	}
	row, err := query().Get()
	if err != nil {
		t.Fatal(err)
	}
	row.Object.UpdatedAt = want.UpdatedAt.AddDate(0, 0, 1)
	row.Object.Title = "Rejected title"

	rejected := errors.New("proxy update rejected")
	ctx := context.WithValue(t.Context(), proxyValidationErrorKey{}, rejected)
	config := models.SaveConfig{IncludeFields: []string{"ProxyModel", "UpdatedAt"}}
	if err := row.Object.SaveObject(ctx, config); !errors.Is(err, rejected) {
		t.Fatalf("update error = %v, want %v", err, rejected)
	}
	reloaded, err := query().Get()
	if err != nil {
		t.Fatal(err)
	}
	assertLoadedProxy(t, reloaded.Object, want)
}

func TestProxyFields(t *testing.T) {
	obj := &ProxiedModel{ProxyModel: &ProxyModel{Title: "Bound instance"}}
	tree := queries.ProxyFields(obj)
	if tree.FieldsLen() != 1 {
		t.Fatalf("proxy count = %d, want 1", tree.FieldsLen())
	}
	field, ok := tree.Get("ProxyModel")
	if !ok || field.Name() != "ProxyModel" || !field.IsProxy() {
		t.Fatalf("invalid proxy field: %v", field)
	}
	if !tree.IsProxyField("ProxyModel") || !tree.IsProxyField(field) || tree.IsProxyField("ID") {
		t.Fatal("proxy metadata misclassified a field")
	}
	if field.Rel().Type() != attrs.RelOneToOne {
		t.Errorf("proxy relation type = %s, want one-to-one", field.Rel().Type())
	}
	if linked := queries.ProxyFields(&LinkedToProxiedModel{}); linked.FieldsLen() != 0 || linked.IsProxyField("ProxiedModel") {
		t.Fatal("ordinary foreign key was classified as a proxy")
	}

	var paths [][]string
	for item, err := range tree.WalkObjectProxies(obj, false) {
		if err != nil {
			t.Fatal(err)
		}
		paths = append(paths, item.Path)
		if item.Value != obj.ProxyModel || item.ParentField.Name() != "ProxyModel" || item.Field.Name() != field.Name() {
			t.Errorf("walk did not return the instance-bound proxy: %+v", item)
		}
	}
	if len(paths) != 1 || !slices.Equal(paths[0], []string{"ProxyModel"}) {
		t.Fatalf("walk paths = %v, want [[ProxyModel]]", paths)
	}
}
