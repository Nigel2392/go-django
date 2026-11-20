//go:build test
// +build test

package queries_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"testing"

	"github.com/Nigel2392/go-django/djester/quest"
	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields/formfields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/attrs/fattrs"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/forms"
	"github.com/Nigel2392/go-django/src/forms/modelforms"
	"github.com/Nigel2392/go-django/src/forms/widgets"
)

type MSRelParent struct {
	ID       uint64
	Title    string
	Children *queries.RelRevFK[*MSRelChild]
}

func (p *MSRelParent) String() string { return p.Title }

func (p *MSRelParent) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, p,
		fattrs.Field(p, "ID", &p.ID, func() fattrs.PtrFieldConfig[*MSRelParent, uint64] {
			return fattrs.PtrFieldConfig[*MSRelParent, uint64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(p, "Title", &p.Title, func() fattrs.PtrFieldConfig[*MSRelParent, string] {
			return fattrs.PtrFieldConfig[*MSRelParent, string]{}
		}),
		fattrs.Field(p, "Children", &p.Children, func() fattrs.PtrFieldConfig[*MSRelParent, *queries.RelRevFK[*MSRelChild]] {
			return fattrs.PtrFieldConfig[*MSRelParent, *queries.RelRevFK[*MSRelChild]]{
				Flags: fattrs.F_IS_REVERSE_FLD,
				Config: attrs.Config{
					ReadOnly:             true,
					RelForeignKeyReverse: attrs.Relate(&MSRelChild{}, "Parent", nil),
					ReverseAlias:         "Parent",
				},
			}
		}),
	).WithTableName("ms_rel_parent")
}

type MSRelChild struct {
	ID     uint64
	Title  string
	Parent *MSRelParent
}

func (c *MSRelChild) String() string { return c.Title }

func (c *MSRelChild) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, c,
		fattrs.Field(c, "ID", &c.ID, func() fattrs.PtrFieldConfig[*MSRelChild, uint64] {
			return fattrs.PtrFieldConfig[*MSRelChild, uint64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(c, "Title", &c.Title, func() fattrs.PtrFieldConfig[*MSRelChild, string] {
			return fattrs.PtrFieldConfig[*MSRelChild, string]{}
		}),
		fattrs.Field(c, "Parent", &c.Parent, func() fattrs.PtrFieldConfig[*MSRelChild, *MSRelParent] {
			return fattrs.PtrFieldConfig[*MSRelChild, *MSRelParent]{
				Config: attrs.Config{
					Null:          true,
					Blank:         true,
					Column:        "parent_id",
					RelForeignKey: attrs.Relate(&MSRelParent{}, "", nil),
					ReverseAlias:  "Children",
				},
			}
		}),
	).WithTableName("ms_rel_child")
}

type MSDirectO2OTarget struct {
	ID    uint64
	Label string
}

func (t *MSDirectO2OTarget) String() string { return t.Label }

func (t *MSDirectO2OTarget) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, t,
		fattrs.Field(t, "ID", &t.ID, func() fattrs.PtrFieldConfig[*MSDirectO2OTarget, uint64] {
			return fattrs.PtrFieldConfig[*MSDirectO2OTarget, uint64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(t, "Label", &t.Label, func() fattrs.PtrFieldConfig[*MSDirectO2OTarget, string] {
			return fattrs.PtrFieldConfig[*MSDirectO2OTarget, string]{}
		}),
	).WithTableName("ms_direct_o2o_target")
}

type MSDirectO2OSource struct {
	ID     uint64
	Title  string
	Target *MSDirectO2OTarget
}

func (s *MSDirectO2OSource) FieldDefs(ctx context.Context) attrs.Definitions {
	return attrs.Make(ctx, s,
		fattrs.Field(s, "ID", &s.ID, func() fattrs.PtrFieldConfig[*MSDirectO2OSource, uint64] {
			return fattrs.PtrFieldConfig[*MSDirectO2OSource, uint64]{
				Config: attrs.Config{Primary: true, ReadOnly: true},
			}
		}),
		fattrs.Field(s, "Title", &s.Title, func() fattrs.PtrFieldConfig[*MSDirectO2OSource, string] {
			return fattrs.PtrFieldConfig[*MSDirectO2OSource, string]{}
		}),
		fattrs.Field(s, "Target", &s.Target, func() fattrs.PtrFieldConfig[*MSDirectO2OSource, *MSDirectO2OTarget] {
			return fattrs.PtrFieldConfig[*MSDirectO2OSource, *MSDirectO2OTarget]{
				Config: attrs.Config{
					Null:        true,
					Blank:       true,
					Column:      "target_id",
					RelOneToOne: attrs.Relate(&MSDirectO2OTarget{}, "", nil),
				},
			}
		}),
	).WithTableName("ms_direct_o2o_source")
}

func init() {
	tables := quest.Table[*testing.T](nil,
		&MSRelParent{},
		&MSRelChild{},
		&MSDirectO2OTarget{},
		&MSDirectO2OSource{},
	)
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)
	tables.IfNotExists = true
	tables.Create()
}

func clearMSRelTables(t *testing.T) {
	t.Helper()
	clearThroughRegTables(t)
	_, _ = queries.GetQuerySet(&MSRelChild{}).Delete()
	_, _ = queries.GetQuerySet(&MSRelParent{}).Delete()
	_, _ = queries.GetQuerySet(&MSDirectO2OSource{}).Delete()
	_, _ = queries.GetQuerySet(&MSDirectO2OTarget{}).Delete()
}

func TestMultiSelectWidget_Validate(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "VALTEST"})
	p1 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Prod 1"})
	p2 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Prod 2"})

	prodField, ok := attrs.Define(ctx, coupon).Field("Products")
	if !ok {
		t.Fatalf("expected Products field")
	}

	makeWidget := func(includeBlank bool) *formfields.MultiSelectWidget[attrs.Definer] {
		return &formfields.MultiSelectWidget[attrs.Definer]{
			BaseWidget:   widgets.NewBaseWidget("model-multiple-select", "forms/widgets/model-multiple-select.html", nil),
			IncludeBlank: includeBlank,
			Relation:     prodField.Rel(),
			FieldDef:     prodField,
		}
	}

	t.Run("ValidSubset_DefinerSlice_IncludeBlankFalse", func(t *testing.T) {
		w := makeWidget(false)
		if valErrs := w.Validate(ctx, []attrs.Definer{p1}); len(valErrs) > 0 {
			t.Errorf("expected 0 validation errors for valid []attrs.Definer subset (IncludeBlank=false), got %v", valErrs)
		}
	})

	t.Run("ValidSubset_StringSlice_IncludeBlankFalse", func(t *testing.T) {
		w := makeWidget(false)
		if valErrs := w.Validate(ctx, []string{strconv.FormatUint(p2.ID, 10)}); len(valErrs) > 0 {
			t.Errorf("expected 0 validation errors for valid []string subset (IncludeBlank=false), got %v", valErrs)
		}
	})

	t.Run("InvalidChoice_StringSlice_IncludeBlankTrue", func(t *testing.T) {
		w := makeWidget(true)
		valErrs := w.Validate(ctx, []string{"999999"})
		if len(valErrs) == 0 {
			t.Fatalf("expected validation error for unknown choice ID, got none")
		}
		if !errors.Is(valErrs[0], errs.ErrInvalidValue) {
			t.Errorf("expected ErrInvalidValue, got %v", valErrs[0])
		}
	})

	t.Run("InvalidChoice_StringSlice_IncludeBlankFalse", func(t *testing.T) {
		w := makeWidget(false)
		valErrs := w.Validate(ctx, []string{"999999"})
		if len(valErrs) != 1 || !errors.Is(valErrs[0], errs.ErrInvalidValue) {
			t.Errorf("expected 1 ErrInvalidValue for unknown choice ID, got %v", valErrs)
		}
	})

	t.Run("InvalidChoice_DefinerSlice", func(t *testing.T) {
		w := makeWidget(false)
		bogus := &ThroughRegProduct{ID: 999999, Title: "Missing"}
		valErrs := w.Validate(ctx, []attrs.Definer{bogus})
		if len(valErrs) != 1 || !errors.Is(valErrs[0], errs.ErrInvalidValue) {
			t.Errorf("expected 1 ErrInvalidValue for nonexistent model in []attrs.Definer, got %v", valErrs)
		}
	})

	t.Run("EmptySelection_BlankBehavior", func(t *testing.T) {
		if valErrs := makeWidget(true).Validate(ctx, []string{}); len(valErrs) > 0 {
			t.Errorf("IncludeBlank=true with []string{} should succeed, got %v", valErrs)
		}
		if valErrs := makeWidget(true).Validate(ctx, []string{""}); len(valErrs) > 0 {
			t.Errorf("IncludeBlank=true with []string{\"\"} should succeed, got %v", valErrs)
		}
		if valErrs := makeWidget(false).Validate(ctx, []string{}); len(valErrs) == 0 {
			t.Errorf("IncludeBlank=false with []string{} should fail validation")
		}
	})
}

func TestManyToManyFormField_MethodsAndSaveField(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "M2MFF"})
	p1 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Alpha"})
	p2 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Beta"})

	t.Run("ValueToGoAndValidateBeforeWidget", func(t *testing.T) {
		loaded, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
		if err != nil {
			t.Fatalf("fetch coupon: %v", err)
		}
		prodFld, ok := attrs.Define(ctx, loaded.Object).Field("Products")
		if !ok {
			t.Fatalf("missing Products field")
		}
		m2mFF, ok := prodFld.FormField().(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ManyToManyFormField, got %T", prodFld.FormField())
		}

		// Must work without calling m2mFF.Widget() first.
		goVal, err := m2mFF.ValueToGo([]string{strconv.FormatUint(p1.ID, 10)})
		if err != nil {
			t.Fatalf("ValueToGo([]string) failed before Widget(): %v", err)
		}
		objs, ok := goVal.([]attrs.Definer)
		if !ok || len(objs) != 1 || attrs.PrimaryKey(ctx, objs[0]) != p1.ID {
			t.Fatalf("ValueToGo returned %#v, want [p1]", goVal)
		}

		if valErrs := m2mFF.Validate(ctx, objs); len(valErrs) > 0 {
			t.Fatalf("Validate failed for valid selection: %v", valErrs)
		}
	})

	t.Run("ModelForm_FullFlow_BlankFalse_SelectedSubset", func(t *testing.T) {
		loaded, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
		if err != nil {
			t.Fatalf("fetch coupon: %v", err)
		}
		mf := modelforms.NewBaseModelForm(ctx, loaded.Object)
		mf.SetFields("Title", "Products")
		mf.Load()
		mf.WithData(url.Values{
			"Title":    {"M2MFF"},
			"Products": {strconv.FormatUint(p1.ID, 10)},
		}, nil, nil)

		if !forms.IsValid(ctx, mf) {
			t.Fatalf("expected ModelForm valid for 1 of 2 products, errList=%v boundErrs=%v",
				mf.ErrorList(), mf.BoundErrors())
		}

		// Calling Instance() before Save() must not error on RelM2M.
		_ = mf.Instance()
		if mf.BoundErrors() != nil && mf.BoundErrors().Len() > 0 {
			t.Fatalf("Instance() added errors: %v", mf.BoundErrors())
		}

		if _, err := mf.Save(); err != nil {
			t.Fatalf("Save() failed: %v", err)
		}

		rows, err := loaded.Object.Products.Objects().WithContext(ctx).All()
		if err != nil {
			t.Fatalf("load Products after Save: %v", err)
		}
		if len(rows) != 1 || rows[0].Object.ID != p1.ID {
			t.Fatalf("expected [p1] persisted, got %d rows", len(rows))
		}
		if inMem := loaded.Object.Products.AsList(); len(inMem) != 1 || inMem[0].Object.ID != p1.ID {
			t.Errorf("expected in-memory coupon.Products synced with [p1], got %v", inMem)
		}
	})

	t.Run("SaveField_ClearWithNilAndEmptySlice", func(t *testing.T) {
		loadCoupon := func(t *testing.T) (*ThroughRegCoupon, attrs.Field) {
			t.Helper()
			row, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
			if err != nil {
				t.Fatalf("fetch coupon: %v", err)
			}
			fld, ok := attrs.Define(ctx, row.Object).Field("Products")
			if !ok {
				t.Fatalf("missing Products field")
			}
			_ = fld.GetValue()
			return row.Object, fld
		}

		loaded, _ := loadCoupon(t)

		// Seed both products
		if _, err := loaded.Products.Objects().WithContext(ctx).SetTargets([]*ThroughRegProduct{p1, p2}); err != nil {
			t.Fatalf("seed targets: %v", err)
		}

		loaded, prodFld := loadCoupon(t)
		m2mFF := prodFld.FormField().(*formfields.ManyToManyFormField)

		if err := m2mFF.SaveField(ctx, prodFld, nil, loaded); err != nil {
			t.Fatalf("SaveField(nil) failed: %v", err)
		}
		rows, err := loaded.Products.Objects().WithContext(ctx).All()
		if err != nil || len(rows) != 0 {
			t.Fatalf("expected 0 relations after SaveField(nil), count=%d err=%v", len(rows), err)
		}

		// Re-seed and clear via non-nil empty []attrs.Definer{}
		if _, err := loaded.Products.Objects().WithContext(ctx).SetTargets([]*ThroughRegProduct{p1}); err != nil {
			t.Fatalf("re-seed target: %v", err)
		}
		if err := m2mFF.SaveField(ctx, prodFld, []attrs.Definer{}, loaded); err != nil {
			t.Fatalf("SaveField([]attrs.Definer{}) failed: %v", err)
		}
		rows, err = loaded.Products.Objects().WithContext(ctx).All()
		if err != nil || len(rows) != 0 {
			t.Fatalf("expected 0 relations after SaveField([]attrs.Definer{}), count=%d err=%v", len(rows), err)
		}
	})
}

func TestMultiSelectWidget_Choices_UnsavedAndRelRevFK(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	t.Run("UnsavedParent_DoesNotQueryWithZeroPK", func(t *testing.T) {
		_ = mustCreateThroughReg(t, &ThroughRegProduct{Title: "Choice A"})
		_ = mustCreateThroughReg(t, &ThroughRegProduct{Title: "Choice B"})

		unsaved := &ThroughRegCoupon{}
		prodFld, _ := attrs.Define(ctx, unsaved).Field("Products")
		m2mFF := prodFld.FormField().(*formfields.ManyToManyFormField)
		w := m2mFF.Widget().(*formfields.MultiSelectWidget[attrs.Definer])

		choices := w.Choices(ctx)
		if len(choices) != 2 {
			t.Fatalf("expected 2 choices on unsaved parent, got %d", len(choices))
		}
		for _, ch := range choices {
			if wrapped := ch.(*widgets.WrappedOption); wrapped.Selected {
				t.Errorf("expected option %q to be unselected for unsaved parent", wrapped.Label())
			}
		}
	})

	t.Run("RelOneToMany_RelRevFK_SelectsLinkedChildren", func(t *testing.T) {
		parent := mustCreateThroughReg(t, &MSRelParent{Title: "Parent"})
		cLinked := mustCreateThroughReg(t, &MSRelChild{Title: "Linked", Parent: parent})
		cUnlinked := mustCreateThroughReg(t, &MSRelChild{Title: "Unlinked"})

		loaded, err := queries.GetQuerySet(&MSRelParent{}).Filter("ID", parent.ID).First()
		if err != nil {
			t.Fatalf("fetch MSRelParent: %v", err)
		}
		childrenFld, ok := attrs.Define(ctx, loaded.Object).Field("Children")
		if !ok {
			t.Fatalf("expected Children field on MSRelParent")
		}
		if ff := childrenFld.FormField(); ff != nil {
			t.Errorf("expected fattrs RelOneToMany Children.FormField() = nil, got %T", ff)
		}

		w := &formfields.MultiSelectWidget[attrs.Definer]{
			BaseWidget:   widgets.NewBaseWidget("model-multiple-select", "forms/widgets/model-multiple-select.html", nil),
			IncludeBlank: true,
			Relation:     childrenFld.Rel(),
			FieldDef:     childrenFld,
		}

		choices := w.Choices(ctx)
		if len(choices) != 2 {
			t.Fatalf("expected 2 child choices, got %d", len(choices))
		}
		selected := make(map[string]bool, len(choices))
		for _, ch := range choices {
			wrapped := ch.(*widgets.WrappedOption)
			selected[wrapped.Value()] = wrapped.Selected
		}
		if !selected[fmt.Sprint(cLinked.ID)] {
			t.Errorf("expected linked child %d selected", cLinked.ID)
		}
		if selected[fmt.Sprint(cUnlinked.ID)] {
			t.Errorf("expected unlinked child %d unselected", cUnlinked.ID)
		}
	})
}

func TestRelQuerySet_DirectMutationWithoutWithContext(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	t.Run("RelM2M_SetTargets_RemoveTargets_ClearTargets_NoPanic", func(t *testing.T) {
		coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "DirectM2M"})
		p1 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Item 1"})
		p2 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "Item 2"})

		reload := func(t *testing.T) *ThroughRegCoupon {
			t.Helper()
			row, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
			if err != nil {
				t.Fatalf("reload coupon: %v", err)
			}
			fld, _ := attrs.Define(ctx, row.Object).Field("Products")
			_ = fld.GetValue()
			return row.Object
		}

		c := reload(t)
		if _, err := c.Products.Objects().SetTargets([]*ThroughRegProduct{p1, p2}); err != nil {
			t.Fatalf("SetTargets without WithContext failed: %v", err)
		}

		c = reload(t)
		if _, err := c.Products.Objects().RemoveTargets(p1); err != nil {
			t.Fatalf("RemoveTargets without WithContext failed: %v", err)
		}
		rows, err := c.Products.Objects().WithContext(ctx).All()
		if err != nil || len(rows) != 1 || rows[0].Object.ID != p2.ID {
			t.Fatalf("after RemoveTargets(p1): got %d rows err=%v, want [p2]", len(rows), err)
		}

		c = reload(t)
		if added, err := c.Products.Objects().SetTargets([]*ThroughRegProduct{}); err != nil || added != 0 {
			t.Fatalf("SetTargets(empty) failed: added=%d err=%v", added, err)
		}
		if afterEmpty, err := reload(t).Products.Objects().WithContext(ctx).All(); err != nil || len(afterEmpty) != 0 {
			t.Fatalf("expected 0 targets after SetTargets(empty), got %d err=%v", len(afterEmpty), err)
		}

		c = reload(t)
		_, _ = c.Products.Objects().WithContext(ctx).AddTargets(p1)
		c = reload(t)
		if _, err := c.Products.Objects().ClearTargets(); err != nil {
			t.Fatalf("ClearTargets without WithContext failed: %v", err)
		}
	})

	t.Run("RelO2O_Clear_And_OneToOneFormField_SaveField_NoPanic", func(t *testing.T) {
		src := mustCreateThroughReg(t, &ThroughRegO2OSource{Title: "O2OSrc"})
		tgt := mustCreateThroughReg(t, &ThroughRegO2OTarget{Title: "O2OTgt"})
		models.Setup(ctx, src)
		models.Setup(ctx, tgt)

		srcFld, ok := attrs.Define(ctx, src).Field("Target")
		if !ok {
			t.Fatalf("missing Target field on ThroughRegO2OSource")
		}
		o2oFF, ok := srcFld.FormField().(*formfields.OneToOneFormField)
		if !ok {
			t.Fatalf("expected *formfields.OneToOneFormField, got %T", srcFld.FormField())
		}

		if err := o2oFF.SaveField(ctx, srcFld, tgt, src); err != nil {
			t.Fatalf("OneToOneFormField.SaveField(tgt) failed: %v", err)
		}
		if src.Target == nil || src.Target.Object == nil || src.Target.Object.ID != tgt.ID {
			t.Errorf("expected in-memory src.Target updated to tgt ID=%d, got %+v", tgt.ID, src.Target)
		}

		reloadedRow, err := queries.GetQuerySet(&ThroughRegO2OSource{}).Filter("ID", src.ID).First()
		if err != nil {
			t.Fatalf("reload O2OSource: %v", err)
		}
		deleted, err := reloadedRow.Object.Target.Objects().Clear(ctx)
		if err != nil {
			t.Fatalf("RelO2O.Objects().Clear(ctx) failed: %v", err)
		}
		if !deleted {
			t.Errorf("expected Clear(ctx) to delete 1 link")
		}

		// Re-link and test OneToOneFormField.SaveField(nil)
		if err := o2oFF.SaveField(ctx, srcFld, tgt, reloadedRow.Object); err != nil {
			t.Fatalf("re-SaveField(tgt) failed: %v", err)
		}
		reloadedRow, err = queries.GetQuerySet(&ThroughRegO2OSource{}).Filter("ID", src.ID).First()
		if err != nil {
			t.Fatalf("reload O2OSource 2: %v", err)
		}
		if err := o2oFF.SaveField(ctx, srcFld, nil, reloadedRow.Object); err != nil {
			t.Fatalf("OneToOneFormField.SaveField(nil) failed: %v", err)
		}
		cnt, err := queries.GetQuerySet(&ThroughRegO2OLink{}).Count()
		if err != nil || cnt != 0 {
			t.Fatalf("expected 0 O2O links after SaveField(nil), got %d err=%v", cnt, err)
		}
	})
}

func TestOneToOneFormField_DirectVsThrough(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	target := mustCreateThroughReg(t, &MSDirectO2OTarget{Label: "Target 1"})
	src := &MSDirectO2OSource{Title: "Direct O2O"}

	fld, ok := attrs.Define(ctx, src).Field("Target")
	if !ok {
		t.Fatalf("missing Target field on MSDirectO2OSource")
	}
	ff := fld.FormField()
	if _, ok := ff.(*formfields.ForeignKeyFormField); !ok {
		t.Fatalf("direct RelOneToOne FormField() = %T, want *formfields.ForeignKeyFormField", ff)
	}

	mf := modelforms.NewBaseModelForm(ctx, src)
	mf.SetFields("Title", "Target")
	mf.Load()
	mf.WithData(url.Values{
		"Title":  {"Direct O2O"},
		"Target": {strconv.FormatUint(target.ID, 10)},
	}, nil, nil)

	if !forms.IsValid(ctx, mf) {
		t.Fatalf("expected direct O2O form valid, errList=%v boundErrs=%v", mf.ErrorList(), mf.BoundErrors())
	}
	if _, err := mf.Save(); err != nil {
		t.Fatalf("direct O2O Save() failed: %v", err)
	}

	saved, err := queries.GetQuerySet(&MSDirectO2OSource{}).Select("*", "Target.*").Filter("ID", src.ID).First()
	if err != nil {
		t.Fatalf("reload direct O2O source: %v", err)
	}
	if saved.Object.Target == nil || saved.Object.Target.ID != target.ID {
		t.Fatalf("saved direct O2O Target = %+v, want ID=%d", saved.Object.Target, target.ID)
	}
}

func TestFormField_BeforeWidgetDispatch(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "PreWidget"})
	p1 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "P1"})
	_ = p1
	parent := mustCreateThroughReg(t, &MSRelParent{Title: "ValidParent"})
	_ = parent
	child := mustCreateThroughReg(t, &MSRelChild{Title: "Child"})

	t.Run("ManyToManyFormField_ValidateRejectsInvalidBeforeWidget", func(t *testing.T) {
		loaded, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
		if err != nil {
			t.Fatalf("fetch coupon: %v", err)
		}
		prodFld, ok := attrs.Define(ctx, loaded.Object).Field("Products")
		if !ok {
			t.Fatalf("missing Products field")
		}
		m2mFF, ok := prodFld.FormField().(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ManyToManyFormField, got %T", prodFld.FormField())
		}

		// Calling Validate before Widget() must dispatch to MultiSelectWidget.Validate,
		// not BaseField -> TextInput.Validate.
		valErrs := m2mFF.Validate(ctx, []string{"999999"})
		if len(valErrs) == 0 || !errors.Is(valErrs[0], errs.ErrInvalidValue) {
			t.Errorf("Validate([]string{\"999999\"}) before Widget() = %v, want ErrInvalidValue", valErrs)
		}

		// Use a fresh field instance (still without calling Widget()) for []attrs.Definer.
		loaded2, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
		if err != nil {
			t.Fatalf("fetch coupon 2: %v", err)
		}
		prodFld2, ok := attrs.Define(ctx, loaded2.Object).Field("Products")
		if !ok {
			t.Fatalf("missing Products field 2")
		}
		m2mFF2, ok := prodFld2.FormField().(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ManyToManyFormField, got %T", prodFld2.FormField())
		}

		valErrs2 := m2mFF2.Validate(ctx, []attrs.Definer{&ThroughRegProduct{ID: 999999}})
		if len(valErrs2) == 0 || !errors.Is(valErrs2[0], errs.ErrInvalidValue) {
			t.Errorf("Validate([]attrs.Definer{999999}) before Widget() = %v, want ErrInvalidValue", valErrs2)
		}
	})

	t.Run("ManyToManyFormField_HideBeforeWidgetPreservesMultiSelect", func(t *testing.T) {
		loaded, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
		if err != nil {
			t.Fatalf("fetch coupon: %v", err)
		}
		prodFld, ok := attrs.Define(ctx, loaded.Object).Field("Products")
		if !ok {
			t.Fatalf("missing Products field")
		}
		m2mFF, ok := prodFld.FormField().(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ManyToManyFormField, got %T", prodFld.FormField())
		}

		m2mFF.Hide(true)
		w := m2mFF.Widget()
		msw, ok := w.(*formfields.MultiSelectWidget[attrs.Definer])
		if !ok {
			t.Fatalf("Widget() after Hide(true) = %T, want *formfields.MultiSelectWidget[attrs.Definer]", w)
		}
		if !msw.IsHidden() {
			t.Errorf("expected MultiSelectWidget.IsHidden() = true after Hide(true)")
		}
	})

	t.Run("ForeignKeyFormField_ValidateRejectsInvalidBeforeWidget", func(t *testing.T) {
		parentFld, ok := attrs.Define(ctx, child).Field("Parent")
		if !ok {
			t.Fatalf("missing Parent field")
		}
		fkFF, ok := parentFld.FormField().(*formfields.ForeignKeyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ForeignKeyFormField, got %T", parentFld.FormField())
		}

		// Calling Validate before Widget() must dispatch to ModelSelect.Validate,
		// not BaseField -> TextInput.Validate.
		valErrs := fkFF.Validate(ctx, &MSRelParent{ID: 999999})
		if len(valErrs) == 0 {
			t.Fatalf("Validate(&MSRelParent{ID: 999999}) before Widget() = nil, want validation error")
		}
	})

	t.Run("ForeignKeyFormField_HideBeforeWidgetPreservesModelSelect", func(t *testing.T) {
		loadedChild, err := queries.GetQuerySet(&MSRelChild{}).Filter("ID", child.ID).First()
		if err != nil {
			t.Fatalf("fetch child: %v", err)
		}
		parentFld, ok := attrs.Define(ctx, loadedChild.Object).Field("Parent")
		if !ok {
			t.Fatalf("missing Parent field")
		}
		fkFF, ok := parentFld.FormField().(*formfields.ForeignKeyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ForeignKeyFormField, got %T", parentFld.FormField())
		}

		fkFF.Hide(true)
		w := fkFF.Widget()
		ms, ok := w.(*formfields.ModelSelect)
		if !ok {
			t.Fatalf("Widget() after Hide(true) = %T, want *formfields.ModelSelect", w)
		}
		if !ms.IsHidden() {
			t.Errorf("expected ModelSelect.IsHidden() = true after Hide(true)")
		}
	})
}

func TestRelationFormField_InMemoryCacheSyncOnSaveField(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	t.Run("ManyToManyFormField_SaveField_SyncsReplaceAndClearInMemory", func(t *testing.T) {
		coupon := mustCreateThroughReg(t, &ThroughRegCoupon{Title: "CacheSyncM2M"})
		p1 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "SyncP1"})
		p2 := mustCreateThroughReg(t, &ThroughRegProduct{Title: "SyncP2"})

		row, err := queries.GetQuerySet(&ThroughRegCoupon{}).Filter("ID", coupon.ID).First()
		if err != nil {
			t.Fatalf("fetch coupon: %v", err)
		}
		loaded := row.Object
		prodFld, ok := attrs.Define(ctx, loaded).Field("Products")
		if !ok {
			t.Fatalf("missing Products field")
		}
		_ = prodFld.GetValue()

		if _, err := loaded.Products.Objects().WithContext(ctx).SetTargets([]*ThroughRegProduct{p1, p2}); err != nil {
			t.Fatalf("seed [p1, p2]: %v", err)
		}
		if list := loaded.Products.AsList(); len(list) != 2 {
			t.Fatalf("expected 2 in-memory products after seed, got %d", len(list))
		}

		m2mFF, ok := prodFld.FormField().(*formfields.ManyToManyFormField)
		if !ok {
			t.Fatalf("expected *formfields.ManyToManyFormField, got %T", prodFld.FormField())
		}

		// 1. Replace [p1, p2] -> [p2]
		if err := m2mFF.SaveField(ctx, prodFld, []attrs.Definer{p2}, loaded); err != nil {
			t.Fatalf("SaveField([p2]) failed: %v", err)
		}
		list := loaded.Products.AsList()
		if len(list) != 1 || list[0].Object.ID != p2.ID {
			t.Errorf("after SaveField([p2]): in-memory Products = %v, want [p2 (ID=%d)]", list, p2.ID)
		}

		// 2. Clear via empty slice
		if err := m2mFF.SaveField(ctx, prodFld, []attrs.Definer{}, loaded); err != nil {
			t.Fatalf("SaveField([]) failed: %v", err)
		}
		if list := loaded.Products.AsList(); len(list) != 0 {
			t.Errorf("after SaveField([]): in-memory Products len = %d, want 0 (%v)", len(list), list)
		}

		// 3. Re-seed [p1], then clear via nil
		if _, err := loaded.Products.Objects().WithContext(ctx).SetTargets([]*ThroughRegProduct{p1}); err != nil {
			t.Fatalf("re-seed [p1]: %v", err)
		}
		if list := loaded.Products.AsList(); len(list) != 1 {
			t.Fatalf("expected 1 in-memory product after re-seed, got %d", len(list))
		}
		if err := m2mFF.SaveField(ctx, prodFld, nil, loaded); err != nil {
			t.Fatalf("SaveField(nil) failed: %v", err)
		}
		if list := loaded.Products.AsList(); len(list) != 0 {
			t.Errorf("after SaveField(nil): in-memory Products len = %d, want 0 (%v)", len(list), list)
		}
	})

	t.Run("OneToOneFormField_SaveField_SyncsSetReplaceAndClearInMemory", func(t *testing.T) {
		src := mustCreateThroughReg(t, &ThroughRegO2OSource{Title: "CacheSyncO2O"})
		tgt1 := mustCreateThroughReg(t, &ThroughRegO2OTarget{Title: "O2OTgt1"})
		tgt2 := mustCreateThroughReg(t, &ThroughRegO2OTarget{Title: "O2OTgt2"})
		models.Setup(ctx, src)
		models.Setup(ctx, tgt1)
		models.Setup(ctx, tgt2)

		srcFld, ok := attrs.Define(ctx, src).Field("Target")
		if !ok {
			t.Fatalf("missing Target field")
		}
		o2oFF, ok := srcFld.FormField().(*formfields.OneToOneFormField)
		if !ok {
			t.Fatalf("expected *formfields.OneToOneFormField, got %T", srcFld.FormField())
		}

		// 1. Set tgt1
		if err := o2oFF.SaveField(ctx, srcFld, tgt1, src); err != nil {
			t.Fatalf("SaveField(tgt1) failed: %v", err)
		}
		if src.Target == nil || src.Target.Object == nil || src.Target.Object.ID != tgt1.ID {
			t.Errorf("after SaveField(tgt1): src.Target = %+v, want Object.ID=%d", src.Target, tgt1.ID)
		}

		// 2. Replace tgt1 -> tgt2
		if err := o2oFF.SaveField(ctx, srcFld, tgt2, src); err != nil {
			t.Fatalf("SaveField(tgt2) failed: %v", err)
		}
		if src.Target == nil || src.Target.Object == nil || src.Target.Object.ID != tgt2.ID {
			t.Errorf("after SaveField(tgt2): src.Target = %+v, want Object.ID=%d", src.Target, tgt2.ID)
		}

		// 3. Clear via nil
		if err := o2oFF.SaveField(ctx, srcFld, nil, src); err != nil {
			t.Fatalf("SaveField(nil) failed: %v", err)
		}
		if src.Target != nil && (src.Target.Object != nil || src.Target.ThroughObject != nil) {
			t.Errorf("after SaveField(nil): expected Object=nil and ThroughObject=nil, got Object=%+v ThroughObject=%+v",
				src.Target.Object, src.Target.ThroughObject)
		}
	})
}

func TestRelO2O_ClearResetsInMemoryObjectAndThrough(t *testing.T) {
	clearMSRelTables(t)
	ctx := t.Context()

	src := mustCreateThroughReg(t, &ThroughRegO2OSource{Title: "O2OClearSrc"})
	tgt := mustCreateThroughReg(t, &ThroughRegO2OTarget{Title: "O2OClearTgt"})
	models.Setup(ctx, src)
	models.Setup(ctx, tgt)

	if _, _, err := src.Target.Objects().WithContext(ctx).SetTarget(tgt); err != nil {
		t.Fatalf("SetTarget failed: %v", err)
	}
	if src.Target == nil || src.Target.Object == nil || src.Target.Object.ID != tgt.ID || src.Target.ThroughObject == nil {
		t.Fatalf("expected populated Object and ThroughObject after SetTarget, got %+v", src.Target)
	}

	deleted, err := src.Target.Objects().Clear(ctx)
	if err != nil {
		t.Fatalf("Clear(ctx) failed: %v", err)
	}
	if !deleted {
		t.Fatalf("expected Clear(ctx) deleted = true")
	}

	if src.Target.Object != nil {
		t.Errorf("after Clear(ctx): src.Target.Object = %+v, want nil", src.Target.Object)
	}
	if src.Target.ThroughObject != nil {
		t.Errorf("after Clear(ctx): src.Target.ThroughObject = %+v, want nil", src.Target.ThroughObject)
	}
}
