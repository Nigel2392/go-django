//go:build test
// +build test

package queries_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/djester/quest"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/queries/src/migrator"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

func init() {
	tables := quest.Table[*testing.T](nil,
		&ThroughRegCoupon{},
		&ThroughRegProduct{},
		&ThroughRegCouponProduct{},
		&ThroughRegDiffPKSource{},
		&ThroughRegDiffPKTarget{},
		&ThroughRegDiffPKLink{},
		&ThroughRegO2OSource{},
		&ThroughRegO2OTarget{},
		&ThroughRegO2OLink{},
	)
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)
	tables.IfNotExists = true
	tables.Create()
}

// --- Model Set A: fattrs.Field mirror of contrib/shop Coupon <-> Product ---

type ThroughRegCoupon struct {
	ID       uint64
	Title    string
	Products *queries.RelM2M[*ThroughRegProduct, *ThroughRegCouponProduct]
}

func (c *ThroughRegCoupon) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*ThroughRegCoupon, uint64] {
			return fattrs.PtrFieldConfig[*ThroughRegCoupon, uint64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(c, "Title", &c.Title, func() fattrs.PtrFieldConfig[*ThroughRegCoupon, string] {
			return fattrs.PtrFieldConfig[*ThroughRegCoupon, string]{}
		}),
		fattrs.Field(c, "Products", &c.Products, func() fattrs.PtrFieldConfig[*ThroughRegCoupon, *queries.RelM2M[*ThroughRegProduct, *ThroughRegCouponProduct]] {
			return fattrs.PtrFieldConfig[*ThroughRegCoupon, *queries.RelM2M[*ThroughRegProduct, *ThroughRegCouponProduct]]{
				Config: attrs.FieldConfig{
					RelManyToMany: attrs.Relate(
						&ThroughRegProduct{},
						"",
						&attrs.ThroughModel{
							This:   &ThroughRegCouponProduct{},
							Source: "Coupon",
							Target: "Product",
						},
					),
					Attributes: map[string]any{
						attrs.AttrReverseAliasKey: "Coupons",
					},
				},
			}
		}),
	).WithTableName("through_reg_coupon")
}

type ThroughRegProduct struct {
	ID      uint64
	Title   string
	Coupons *queries.RelM2M[attrs.Definer, attrs.Definer]
}

func (p *ThroughRegProduct) String() string {
	return p.Title
}

func (p *ThroughRegProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		fattrs.Field(p, "ID", &p.ID, func() fattrs.PtrFieldConfig[*ThroughRegProduct, uint64] {
			return fattrs.PtrFieldConfig[*ThroughRegProduct, uint64]{
				Config: attrs.FieldConfig{
					Primary:  true,
					ReadOnly: true,
				},
			}
		}),
		fattrs.Field(p, "Title", &p.Title, func() fattrs.PtrFieldConfig[*ThroughRegProduct, string] {
			return fattrs.PtrFieldConfig[*ThroughRegProduct, string]{}
		}),
		fattrs.Field(p, "Coupons", &p.Coupons, func() fattrs.PtrFieldConfig[*ThroughRegProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]] {
			return fattrs.PtrFieldConfig[*ThroughRegProduct, *queries.RelM2M[attrs.Definer, attrs.Definer]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.FieldConfig{
					Attributes: map[string]any{
						migrator.AttrUseInDBKey: false,
					},
				},
			}
		}),
	).WithTableName("through_reg_product")
}

type ThroughRegCouponProduct struct {
	Coupon  *ThroughRegCoupon
	Product *ThroughRegProduct
}

func (m *ThroughRegCouponProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, m,
		fattrs.Field(m, "Coupon", &m.Coupon, func() fattrs.PtrFieldConfig[*ThroughRegCouponProduct, *ThroughRegCoupon] {
			return fattrs.PtrFieldConfig[*ThroughRegCouponProduct, *ThroughRegCoupon]{
				Config: attrs.FieldConfig{
					Column: "coupon_id",
				},
			}
		}),
		fattrs.Field(m, "Product", &m.Product, func() fattrs.PtrFieldConfig[*ThroughRegCouponProduct, *ThroughRegProduct] {
			return fattrs.PtrFieldConfig[*ThroughRegCouponProduct, *ThroughRegProduct]{
				Config: attrs.FieldConfig{
					Column: "product_id",
				},
			}
		}),
	).WithTableName("through_reg_coupon_product")
}

// --- Model Set B: Mismatched PK column names for M2M ---

type ThroughRegDiffPKSource struct {
	models.Model
	ID    uint64
	Title string
	Items *queries.RelM2M[*ThroughRegDiffPKTarget, *ThroughRegDiffPKLink]
}

func (m *ThroughRegDiffPKSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Column:  "source_pk",
			Primary: true,
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Column: "title",
		}),
		fields.NewManyToManyField[*queries.RelM2M[*ThroughRegDiffPKTarget, *ThroughRegDiffPKLink]](m, "Items", &fields.FieldConfig{
			ScanTo:      &m.Items,
			ReverseName: "Sources",
			Rel: attrs.Relate(
				&ThroughRegDiffPKTarget{},
				"",
				&attrs.ThroughModel{
					This:   &ThroughRegDiffPKLink{},
					Source: "Source",
					Target: "Target",
				},
			),
		}),
	).WithTableName("through_reg_diffpk_source")
}

type ThroughRegDiffPKTarget struct {
	models.Model
	ID    uint64
	Title string
}

func (m *ThroughRegDiffPKTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Column:  "target_pk",
			Primary: true,
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Column: "title",
		}),
	).WithTableName("through_reg_diffpk_target")
}

type ThroughRegDiffPKLink struct {
	models.Model
	Source *ThroughRegDiffPKSource
	Target *ThroughRegDiffPKTarget
}

func (m *ThroughRegDiffPKLink) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "Source", &attrs.FieldConfig{
			Column: "source_id",
		}),
		attrs.NewField(m, "Target", &attrs.FieldConfig{
			Column: "target_id",
		}),
	).WithTableName("through_reg_diffpk_link")
}

// --- Model Set C: Mismatched PK column names for O2O with through ---

type ThroughRegO2OSource struct {
	models.Model
	ID     uint64
	Title  string
	Target *queries.RelO2O[*ThroughRegO2OTarget, *ThroughRegO2OLink]
}

func (m *ThroughRegO2OSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Column:  "source_pk",
			Primary: true,
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Column: "title",
		}),
		fields.NewOneToOneField[*queries.RelO2O[*ThroughRegO2OTarget, *ThroughRegO2OLink]](m, "Target", &fields.FieldConfig{
			ScanTo:      &m.Target,
			ReverseName: "SourceReverse",
			Rel: attrs.Relate(
				&ThroughRegO2OTarget{},
				"",
				&attrs.ThroughModel{
					This:   &ThroughRegO2OLink{},
					Source: "Source",
					Target: "Target",
				},
			),
		}),
	).WithTableName("through_reg_o2o_source")
}

type ThroughRegO2OTarget struct {
	models.Model
	ID    uint64
	Title string
}

func (m *ThroughRegO2OTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.FieldConfig{
			Column:  "target_pk",
			Primary: true,
		}),
		attrs.NewField(m, "Title", &attrs.FieldConfig{
			Column: "title",
		}),
	).WithTableName("through_reg_o2o_target")
}

type ThroughRegO2OLink struct {
	models.Model
	Source *ThroughRegO2OSource
	Target *ThroughRegO2OTarget
}

func (m *ThroughRegO2OLink) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "Source", &attrs.FieldConfig{
			Column: "source_id",
		}),
		attrs.NewField(m, "Target", &attrs.FieldConfig{
			Column: "target_id",
		}),
	).WithTableName("through_reg_o2o_link")
}

func clearThroughRegTables(t *testing.T) {
	t.Helper()
	modelsToClear := []attrs.Definer{
		&ThroughRegCouponProduct{},
		&ThroughRegCoupon{},
		&ThroughRegProduct{},
		&ThroughRegDiffPKLink{},
		&ThroughRegDiffPKSource{},
		&ThroughRegDiffPKTarget{},
		&ThroughRegO2OLink{},
		&ThroughRegO2OSource{},
		&ThroughRegO2OTarget{},
	}
	for _, m := range modelsToClear {
		if _, err := queries.GetQuerySet(m).Delete(); err != nil {
			t.Fatalf("failed to clear %T: %v", m, err)
		}
	}
}

func mustCreateThroughReg[T attrs.Definer](t *testing.T, v T) T {
	t.Helper()
	res, err := queries.GetQuerySet(v).Create(v)
	if err != nil {
		t.Fatalf("failed to create %T: %v", v, err)
	}
	return res
}

// --- Granular Regression Tests ---

func TestThroughRelatedQuerySet_JoinCondition_TargetColumn(t *testing.T) {
	t.Run("AttrsFieldDef_M2M", func(t *testing.T) {
		coupon := &ThroughRegCoupon{ID: 7, Title: "C7"}
		attrsField := attrs.NewField(coupon, "Products", &attrs.FieldConfig{
			RelManyToMany: attrs.Relate(
				&ThroughRegProduct{},
				"",
				&attrs.ThroughModel{
					This:   &ThroughRegCouponProduct{},
					Source: "Coupon",
					Target: "Product",
				},
			),
		})
		backRef := &queries.RelM2M[attrs.Definer, attrs.Definer]{
			Parent: &queries.ParentInfo{
				Object: coupon,
				Field:  attrsField,
			},
		}

		qs := backRef.Objects().WithContext(t.Context())
		if qs.JoinDefCondition == nil {
			t.Fatalf("expected non-nil JoinDefCondition")
		}
		if qs.JoinDefCondition.ConditionA.TableOrAlias != "through_reg_product" {
			t.Errorf("ConditionA.TableOrAlias = %q, want %q", qs.JoinDefCondition.ConditionA.TableOrAlias, "through_reg_product")
		}
		if qs.JoinDefCondition.ConditionA.FieldColumn == nil {
			t.Fatalf("expected non-nil ConditionA.FieldColumn")
		}
		if got := qs.JoinDefCondition.ConditionA.FieldColumn.ColumnName(); got != "id" {
			t.Errorf("ConditionA.FieldColumn.ColumnName() = %q, want %q", got, "id")
		}
		if got := qs.JoinDefCondition.ConditionB.FieldColumn.ColumnName(); got != "product_id" {
			t.Errorf("ConditionB.FieldColumn.ColumnName() = %q, want %q", got, "product_id")
		}
		if qs.JoinDefCondition.Next == nil {
			t.Fatalf("expected non-nil JoinDefCondition.Next")
		}
		if got := qs.JoinDefCondition.Next.ConditionA.FieldColumn.ColumnName(); got != "coupon_id" {
			t.Errorf("Next.ConditionA.FieldColumn.ColumnName() = %q, want %q", got, "coupon_id")
		}
		if len(qs.JoinDefCondition.Next.ConditionB.Values) == 0 || qs.JoinDefCondition.Next.ConditionB.Values[0] != uint64(7) {
			t.Errorf("Next.ConditionB.Values = %v, want [7]", qs.JoinDefCondition.Next.ConditionB.Values)
		}
	})

	t.Run("RelationField_M2M_DifferentPKColumns", func(t *testing.T) {
		src := &ThroughRegDiffPKSource{ID: 9, Title: "S9"}
		itemsField, ok := attrs.Define(t.Context(), src).Field("Items")
		if !ok {
			t.Fatalf("expected Items field on ThroughRegDiffPKSource")
		}
		backRef := &queries.RelM2M[*ThroughRegDiffPKTarget, *ThroughRegDiffPKLink]{
			Parent: &queries.ParentInfo{
				Object: src,
				Field:  itemsField,
			},
		}

		qs := backRef.Objects().WithContext(t.Context())
		if qs.JoinDefCondition == nil {
			t.Fatalf("expected non-nil JoinDefCondition")
		}
		if qs.JoinDefCondition.ConditionA.TableOrAlias != "through_reg_diffpk_target" {
			t.Errorf("ConditionA.TableOrAlias = %q, want %q", qs.JoinDefCondition.ConditionA.TableOrAlias, "through_reg_diffpk_target")
		}
		if qs.JoinDefCondition.ConditionA.FieldColumn == nil {
			t.Fatalf("expected non-nil ConditionA.FieldColumn")
		}
		if got := qs.JoinDefCondition.ConditionA.FieldColumn.ColumnName(); got != "target_pk" {
			t.Errorf("ConditionA.FieldColumn.ColumnName() = %q, want %q", got, "target_pk")
		}
	})

	t.Run("RelationField_O2O_WithThrough_DifferentPKColumns", func(t *testing.T) {
		src := &ThroughRegO2OSource{ID: 5, Title: "O5"}
		targetField, ok := attrs.Define(t.Context(), src).Field("Target")
		if !ok {
			t.Fatalf("expected Target field on ThroughRegO2OSource")
		}
		backRef := &queries.RelO2O[*ThroughRegO2OTarget, *ThroughRegO2OLink]{
			Parent: &queries.ParentInfo{
				Object: src,
				Field:  targetField,
			},
		}

		qs := backRef.Objects().WithContext(t.Context())
		if qs.JoinDefCondition == nil {
			t.Fatalf("expected non-nil JoinDefCondition")
		}
		if qs.JoinDefCondition.ConditionA.TableOrAlias != "through_reg_o2o_target" {
			t.Errorf("ConditionA.TableOrAlias = %q, want %q", qs.JoinDefCondition.ConditionA.TableOrAlias, "through_reg_o2o_target")
		}
		if qs.JoinDefCondition.ConditionA.FieldColumn == nil {
			t.Fatalf("expected non-nil ConditionA.FieldColumn")
		}
		if got := qs.JoinDefCondition.ConditionA.FieldColumn.ColumnName(); got != "target_pk" {
			t.Errorf("ConditionA.FieldColumn.ColumnName() = %q, want %q", got, "target_pk")
		}
	})
}

func TestThroughRelatedQuerySet_SQL_Execution(t *testing.T) {
	t.Run("AttrsFieldDef_M2M_All", func(t *testing.T) {
		clearThroughRegTables(t)

		coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "C1"})
		p1 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "P1"})
		_ = mustCreateThroughReg(t, &ThroughRegProduct{Title: "P2"})
		_ = mustCreateThroughReg(t, &ThroughRegCouponProduct{Coupon: coupon, Product: p1})

		attrsField := attrs.NewField(coupon, "Products", &attrs.FieldConfig{
			RelManyToMany: attrs.Relate(
				&ThroughRegProduct{},
				"",
				&attrs.ThroughModel{
					This:   &ThroughRegCouponProduct{},
					Source: "Coupon",
					Target: "Product",
				},
			),
		})
		backRef := &queries.RelM2M[attrs.Definer, attrs.Definer]{
			Parent: &queries.ParentInfo{
				Object: coupon,
				Field:  attrsField,
			},
		}

		rows, err := backRef.Objects().WithContext(t.Context()).All()
		if err != nil {
			t.Fatalf("backRef.Objects().All() failed: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 linked product, got %d", len(rows))
		}
		if pk := attrs.PrimaryKey(t.Context(), rows[0].Object); pk != p1.ID {
			t.Errorf("linked product PK = %v, want %d", pk, p1.ID)
		}
	})

	t.Run("RelationField_M2M_DifferentPKColumns_All", func(t *testing.T) {
		clearThroughRegTables(t)

		src := mustCreateThroughReg(t, &ThroughRegDiffPKSource{Title: "S1"})
		t1 := mustCreateThroughReg(t, &ThroughRegDiffPKTarget{Title: "T1"})
		_ = mustCreateThroughReg(t, &ThroughRegDiffPKTarget{Title: "T2"})
		_ = mustCreateThroughReg(t, &ThroughRegDiffPKLink{Source: src, Target: t1})

		loadedRow, err := queries.GetQuerySet(&ThroughRegDiffPKSource{}).Filter("ID", src.ID).First()
		if err != nil {
			t.Fatalf("failed to load ThroughRegDiffPKSource: %v", err)
		}

		rows, err := loadedRow.Object.Items.Objects().WithContext(t.Context()).All()
		if err != nil {
			t.Fatalf("Items.Objects().All() failed: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 linked item, got %d", len(rows))
		}
		if rows[0].Object.ID != t1.ID {
			t.Errorf("linked item ID = %d, want %d", rows[0].Object.ID, t1.ID)
		}
	})

	t.Run("RelationField_O2O_WithThrough_DifferentPKColumns_All", func(t *testing.T) {
		clearThroughRegTables(t)

		src := mustCreateThroughReg(t, &ThroughRegO2OSource{Title: "O2O_S1"})
		tgt := mustCreateThroughReg(t, &ThroughRegO2OTarget{Title: "O2O_T1"})
		_ = mustCreateThroughReg(t, &ThroughRegO2OLink{Source: src, Target: tgt})

		loadedRow, err := queries.GetQuerySet(&ThroughRegO2OSource{}).Filter("ID", src.ID).First()
		if err != nil {
			t.Fatalf("failed to load ThroughRegO2OSource: %v", err)
		}

		rows, err := loadedRow.Object.Target.Objects().WithContext(t.Context()).All()
		if err != nil {
			t.Fatalf("Target.Objects().All() failed: %v", err)
		}
		if len(rows) != 1 {
			t.Fatalf("expected 1 linked O2O target, got %d", len(rows))
		}
		if rows[0].Object.ID != tgt.ID {
			t.Errorf("linked target ID = %d, want %d", rows[0].Object.ID, tgt.ID)
		}
	})
}

func TestThroughQuerySet_Preload_DifferentPKColumns(t *testing.T) {
	clearThroughRegTables(t)

	src := mustCreateThroughReg(t, &ThroughRegDiffPKSource{Title: "Preload_S1"})
	t1 := mustCreateThroughReg(t, &ThroughRegDiffPKTarget{Title: "Preload_T1"})
	_ = mustCreateThroughReg(t, &ThroughRegDiffPKTarget{Title: "Preload_T2"})
	_ = mustCreateThroughReg(t, &ThroughRegDiffPKLink{Source: src, Target: t1})

	rows, err := queries.GetQuerySet(&ThroughRegDiffPKSource{}).
		Filter("ID", src.ID).
		Preload("Items").
		All()
	if err != nil {
		t.Fatalf("Preload(\"Items\").All() failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 source row, got %d", len(rows))
	}

	items := rows[0].Object.Items.AsList()
	if len(items) != 1 {
		t.Fatalf("expected 1 preloaded item, got %d", len(items))
	}
	if items[0].Object.ID != t1.ID {
		t.Errorf("preloaded item ID = %d, want %d", items[0].Object.ID, t1.ID)
	}
}

func TestMultiSelectWidget_FattrsM2M_EditInstance(t *testing.T) {
	clearThroughRegTables(t)

	coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "SAVE10"})
	pLinked := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Linked Product"})
	pUnlinked := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Unlinked Product"})
	_ = mustCreateThroughReg(t, &ThroughRegCouponProduct{Coupon: coupon, Product: pLinked})

	loadedRow, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
	if err != nil {
		t.Fatalf("failed to fetch ThroughRegCoupon: %v", err)
	}
	loadedCoupon := loadedRow.Object

	t.Run("FormFieldAndWidget_BoundToLoadedCouponInstance", func(t *testing.T) {
		prodField, ok := attrs.Define(t.Context(), loadedCoupon).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on ThroughRegCoupon")
		}

		ff := prodField.FormField()
		if ff == nil {
			t.Fatalf("expected non-nil FormField for Products")
		}
		m2mFF, ok := ff.(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("FormField type = %T, want *formfields.ManyToManyFormField", ff)
		}

		if gotPK := attrs.PrimaryKey(t.Context(), m2mFF.Field.Instance()); gotPK != loadedCoupon.ID {
			t.Errorf("m2mFF.Field.Instance() PK = %v, want %d", gotPK, loadedCoupon.ID)
		}

		w := m2mFF.Widget()
		m2mWidget, ok := w.(*formfields.MultiSelectWidget[attrs.Definer])
		if !ok {
			t.Fatalf("Widget type = %T, want *formfields.MultiSelectWidget[attrs.Definer]", w)
		}

		if gotPK := attrs.PrimaryKey(t.Context(), m2mWidget.FieldDef.Instance()); gotPK != loadedCoupon.ID {
			t.Errorf("m2mWidget.FieldDef.Instance() PK = %v, want %d", gotPK, loadedCoupon.ID)
		}
	})

	t.Run("MultiSelectWidget_Choices_NoPanicAndSelectsLinkedProduct", func(t *testing.T) {
		prodField, ok := attrs.Define(t.Context(), loadedCoupon).Field("Products")
		if !ok {
			t.Fatalf("expected Products field on ThroughRegCoupon")
		}
		m2mFF, ok := prodField.FormField().(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("FormField type = %T, want *formfields.ManyToManyFormField", prodField.FormField())
		}
		m2mWidget, ok := m2mFF.Widget().(*formfields.MultiSelectWidget[attrs.Definer])
		if !ok {
			t.Fatalf("Widget type = %T, want *formfields.MultiSelectWidget[attrs.Definer]", m2mFF.Widget())
		}

		var choices []widgets.Option
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("MultiSelectWidget.Choices panicked: %v", r)
				}
			}()
			choices = m2mWidget.Choices(t.Context())
		}()

		if len(choices) != 2 {
			t.Fatalf("expected 2 choices, got %d", len(choices))
		}

		selectedByValue := make(map[string]bool, len(choices))
		for _, ch := range choices {
			wrapped, ok := ch.(*widgets.WrappedOption)
			if !ok {
				t.Fatalf("choice type = %T, want *widgets.WrappedOption", ch)
			}
			selectedByValue[wrapped.Value()] = wrapped.Selected
		}

		linkedVal := fmt.Sprint(pLinked.ID)
		unlinkedVal := fmt.Sprint(pUnlinked.ID)
		if sel, ok := selectedByValue[linkedVal]; !ok || !sel {
			t.Errorf("expected linked product %q to be selected (present=%v, selected=%v)", linkedVal, ok, sel)
		}
		if sel, ok := selectedByValue[unlinkedVal]; !ok || sel {
			t.Errorf("expected unlinked product %q to be unselected (present=%v, selected=%v)", unlinkedVal, ok, sel)
		}
	})

	t.Run("ModelForm_BoundFormField_Render_NoPanicAndSelectedOption", func(t *testing.T) {
		form := modelforms.NewBaseModelForm(t.Context(), loadedCoupon)
		form.Load()

		bf, ok := form.BoundFields().Get("Products")
		if !ok {
			t.Fatalf("expected Products in form.BoundFields()")
		}

		rendered := string(bf.Field())
		if rendered == "" {
			t.Fatalf("expected non-empty rendered HTML for Products field (empty string indicates panic recovery in BoundFormField.Field)")
		}

		wantSelected := fmt.Sprintf("<option value=%q selected>%s</option>", fmt.Sprint(pLinked.ID), pLinked.Title)
		wantUnselected := fmt.Sprintf("<option value=%q>%s</option>", fmt.Sprint(pUnlinked.ID), pUnlinked.Title)

		if !strings.Contains(rendered, wantSelected) {
			t.Errorf("rendered HTML missing selected option %q:\n%s", wantSelected, rendered)
		}
		if !strings.Contains(rendered, wantUnselected) {
			t.Errorf("rendered HTML missing unselected option %q:\n%s", wantUnselected, rendered)
		}
	})
}
