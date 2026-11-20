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
	"github.com/Nigel2392/go-django/queries/src/expr"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

func init() {
	tables := quest.Table[*testing.T](nil,
		&MergeCountry{},
		&MergeAddress{},
		&MergeCategory{},
		&MergeSupplier{},
		&MergeProduct{},
		&MergeTag{},
		&MergeOrder{},
		&MergeOrderTag{},
		&MergeOrderItem{},
	)
	_ = attrs.ResetDefinitions.Send(context.Background(), nil)
	tables.IfNotExists = true
	tables.Create()
}

type MergeCountry struct {
	models.Model
	ID   int64
	Name string
	Code string
}

func (m *MergeCountry) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Name", &attrs.Config{}),
		attrs.NewField(m, "Code", &attrs.Config{}),
	).WithTableName("merge_countries")
}

type MergeAddress struct {
	models.Model
	ID         int64
	City       string
	Street     string
	PostalCode string
	Country    *MergeCountry
}

func (m *MergeAddress) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "City", &attrs.Config{}),
		attrs.NewField(m, "Street", &attrs.Config{}),
		attrs.NewField(m, "PostalCode", &attrs.Config{}),
		attrs.NewField(m, "Country", &attrs.Config{
			Column:        "country_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeCountry{}, "", nil),
		}),
	).WithTableName("merge_addresses")
}

type MergeCategory struct {
	models.Model
	ID     int64
	Title  string
	Slug   string
	Parent *MergeCategory
}

func (m *MergeCategory) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Title", &attrs.Config{}),
		attrs.NewField(m, "Slug", &attrs.Config{}),
		attrs.NewField(m, "Parent", &attrs.Config{
			Column:        "parent_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeCategory{}, "", nil),
		}),
	).WithTableName("merge_categories")
}

type MergeSupplier struct {
	models.Model
	ID      int64
	Name    string
	Code    string
	Address *MergeAddress
}

func (m *MergeSupplier) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Name", &attrs.Config{}),
		attrs.NewField(m, "Code", &attrs.Config{}),
		attrs.NewField(m, "Address", &attrs.Config{
			Column:        "address_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeAddress{}, "", nil),
		}),
	).WithTableName("merge_suppliers")
}

type MergeProduct struct {
	models.Model
	ID       int64
	Title    string
	Slug     string
	Price    int64
	Stock    int64
	Category *MergeCategory
	Supplier *MergeSupplier
}

func (m *MergeProduct) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Title", &attrs.Config{}),
		attrs.NewField(m, "Slug", &attrs.Config{}),
		attrs.NewField(m, "Price", &attrs.Config{}),
		attrs.NewField(m, "Stock", &attrs.Config{}),
		attrs.NewField(m, "Category", &attrs.Config{
			Column:        "category_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeCategory{}, "", nil),
		}),
		attrs.NewField(m, "Supplier", &attrs.Config{
			Column:        "supplier_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeSupplier{}, "", nil),
		}),
	).WithTableName("merge_products")
}

type MergeTag struct {
	models.Model
	ID     int64
	Name   string
	Weight int64
}

func (m *MergeTag) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Name", &attrs.Config{}),
		attrs.NewField(m, "Weight", &attrs.Config{}),
	).WithTableName("merge_tags")
}

type MergeOrderTag struct {
	models.Model
	Order     *MergeOrder
	Tag       *MergeTag
	SortOrder int64
}

func (m *MergeOrderTag) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "Order", &attrs.Config{Column: "order_id", Null: false}),
		attrs.NewField(m, "Tag", &attrs.Config{Column: "tag_id", Null: false}),
		attrs.NewField(m, "SortOrder", &attrs.Config{}),
	).WithTableName("merge_order_tags")
}

type MergeOrder struct {
	models.Model
	ID              int64
	Reference       string
	Total           int64
	Active          bool
	BillingAddress  *MergeAddress
	ShippingAddress *MergeAddress
	Items           *queries.RelRevFK[*MergeOrderItem]
	Tags            *queries.RelM2M[*MergeTag, *MergeOrderTag]
}

func (m *MergeOrder) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Reference", &attrs.Config{}),
		attrs.NewField(m, "Total", &attrs.Config{}),
		attrs.NewField(m, "Active", &attrs.Config{}),
		attrs.NewField(m, "BillingAddress", &attrs.Config{
			Column:        "billing_address_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeAddress{}, "", nil),
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "BillingOrders",
			},
		}),
		attrs.NewField(m, "ShippingAddress", &attrs.Config{
			Column:        "shipping_address_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeAddress{}, "", nil),
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "ShippingOrders",
			},
		}),
		fields.NewForeignKeyReverseField[*queries.RelRevFK[*MergeOrderItem]](m, "Items", &fields.FieldConfig{
			ScanTo:    &m.Items,
			IsReverse: true,
		}),
		fields.NewManyToManyField[*queries.RelM2M[*MergeTag, *MergeOrderTag]](m, "Tags", &fields.FieldConfig{
			ScanTo:      &m.Tags,
			ReverseName: "TagOrders",
			Rel: attrs.Relate(&MergeTag{}, "", &attrs.ThroughModel{
				This:   &MergeOrderTag{},
				Source: "Order",
				Target: "Tag",
			}),
		}),
	).WithTableName("merge_orders")
}

type MergeOrderItem struct {
	models.Model
	ID               int64
	Order            *MergeOrder
	Product          *MergeProduct
	SecondaryProduct *MergeProduct
	Quantity         int64
	Note             string
	Discount         int64
}

func (m *MergeOrderItem) FieldDefs(ctx context.Context) attrs.Definitions {
	return m.Model.Define(ctx, m,
		attrs.NewField(m, "ID", &attrs.Config{Primary: true, ReadOnly: true}),
		attrs.NewField(m, "Order", &attrs.Config{
			Column:        "order_id",
			Null:          false,
			RelForeignKey: attrs.Relate(&MergeOrder{}, "", nil),
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "Items",
			},
		}),
		attrs.NewField(m, "Product", &attrs.Config{
			Column:        "product_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeProduct{}, "", nil),
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "PrimaryItems",
			},
		}),
		attrs.NewField(m, "SecondaryProduct", &attrs.Config{
			Column:        "secondary_product_id",
			Null:          true,
			RelForeignKey: attrs.Relate(&MergeProduct{}, "", nil),
			Attributes: map[string]any{
				attrs.AttrReverseAliasKey: "SecondaryItems",
			},
		}),
		attrs.NewField(m, "Quantity", &attrs.Config{}),
		attrs.NewField(m, "Note", &attrs.Config{}),
		attrs.NewField(m, "Discount", &attrs.Config{}),
	).WithTableName("merge_order_items")
}

type mergeFixtures struct {
	countryNL  *MergeCountry
	countryDE  *MergeCountry
	addrBill   *MergeAddress
	addrShip   *MergeAddress
	addrSup    *MergeAddress
	catRoot    *MergeCategory
	catChildA  *MergeCategory
	catChildB  *MergeCategory
	supAlpha   *MergeSupplier
	prodCheap  *MergeProduct
	prodMid    *MergeProduct
	prodExp    *MergeProduct
	tagVIP     *MergeTag
	tagExpress *MergeTag
	order1     *MergeOrder
	order2     *MergeOrder
	item1      *MergeOrderItem
	item2      *MergeOrderItem
	item3      *MergeOrderItem
}

func clearMergeTables(t *testing.T) {
	t.Helper()
	modelsToClear := []attrs.Definer{
		&MergeOrderItem{},
		&MergeOrderTag{},
		&MergeOrder{},
		&MergeTag{},
		&MergeProduct{},
		&MergeSupplier{},
	}
	for _, m := range modelsToClear {
		if _, err := queries.GetQuerySet(m).Delete(); err != nil {
			t.Fatalf("failed to clear %T: %v", m, err)
		}
	}
	if _, err := queries.GetQuerySet(&MergeCategory{}).Filter("Parent__isnull", false).Delete(); err != nil {
		t.Fatalf("failed to clear child *queries_test.MergeCategory: %v", err)
	}
	for _, m := range []attrs.Definer{&MergeCategory{}, &MergeAddress{}, &MergeCountry{}} {
		if _, err := queries.GetQuerySet(m).Delete(); err != nil {
			t.Fatalf("failed to clear %T: %v", m, err)
		}
	}
}

func mustCreate[T attrs.Definer](t *testing.T, v T) T {
	t.Helper()
	res, err := queries.GetQuerySet(v).Create(v)
	if err != nil {
		t.Fatalf("failed to create %T: %v", v, err)
	}
	return res
}

func seedMergeFixtures(t *testing.T) *mergeFixtures {
	t.Helper()
	clearMergeTables(t)

	f := &mergeFixtures{}
	f.countryNL = mustCreate(t, &MergeCountry{Name: "Netherlands", Code: "NL"})
	f.countryDE = mustCreate(t, &MergeCountry{Name: "Germany", Code: "DE"})

	f.addrBill = mustCreate(t, &MergeAddress{City: "Amsterdam", Street: "Damrak 1", PostalCode: "1012LG", Country: f.countryNL})
	f.addrShip = mustCreate(t, &MergeAddress{City: "Berlin", Street: "Torstr 99", PostalCode: "10119", Country: f.countryDE})
	f.addrSup = mustCreate(t, &MergeAddress{City: "Rotterdam", Street: "Coolsingel 5", PostalCode: "3012AA", Country: f.countryNL})

	f.catRoot = mustCreate(t, &MergeCategory{Title: "Electronics", Slug: "electronics"})
	f.catChildA = mustCreate(t, &MergeCategory{Title: "Keyboards", Slug: "keyboards", Parent: f.catRoot})
	f.catChildB = mustCreate(t, &MergeCategory{Title: "Monitors", Slug: "monitors", Parent: f.catRoot})

	f.supAlpha = mustCreate(t, &MergeSupplier{Name: "Alpha Supply", Code: "SUP-A", Address: f.addrSup})

	f.prodCheap = mustCreate(t, &MergeProduct{Title: "USB Cable", Slug: "usb-cable", Price: 15, Stock: 50, Category: f.catChildA, Supplier: f.supAlpha})
	f.prodMid = mustCreate(t, &MergeProduct{Title: "Mech Keyboard", Slug: "mech-keyboard", Price: 120, Stock: 10, Category: f.catChildA, Supplier: f.supAlpha})
	f.prodExp = mustCreate(t, &MergeProduct{Title: "4K Monitor", Slug: "4k-monitor", Price: 450, Stock: 5, Category: f.catChildB, Supplier: f.supAlpha})

	f.tagVIP = mustCreate(t, &MergeTag{Name: "vip", Weight: 10})
	f.tagExpress = mustCreate(t, &MergeTag{Name: "express", Weight: 5})

	f.order1 = mustCreate(t, &MergeOrder{
		Reference:       "ORD-1001",
		Total:           200,
		Active:          true,
		BillingAddress:  f.addrBill,
		ShippingAddress: f.addrShip,
	})
	f.order2 = mustCreate(t, &MergeOrder{
		Reference:       "ORD-1002",
		Total:           50,
		Active:          false,
		BillingAddress:  f.addrShip,
		ShippingAddress: f.addrBill,
	})

	_ = mustCreate(t, &MergeOrderTag{Order: f.order1, Tag: f.tagExpress, SortOrder: 1})
	_ = mustCreate(t, &MergeOrderTag{Order: f.order1, Tag: f.tagVIP, SortOrder: 2})

	f.item1 = mustCreate(t, &MergeOrderItem{
		Order:            f.order1,
		Product:          f.prodMid,
		SecondaryProduct: f.prodCheap,
		Quantity:         2,
		Note:             "urgent",
		Discount:         10,
	})
	f.item2 = mustCreate(t, &MergeOrderItem{
		Order:            f.order1,
		Product:          f.prodCheap,
		SecondaryProduct: f.prodExp,
		Quantity:         5,
		Note:             "standard",
		Discount:         2,
	})
	f.item3 = mustCreate(t, &MergeOrderItem{
		Order:            f.order1,
		Product:          f.prodExp,
		SecondaryProduct: f.prodMid,
		Quantity:         1,
		Note:             "fragile",
		Discount:         25,
	})

	return f
}

func getPreloadEntry[T attrs.Definer](t *testing.T, qs *queries.QuerySet[T], path string) *queries.Preload {
	t.Helper()
	internals := qs.Internals()
	if internals.Preload == nil {
		return nil
	}
	for _, p := range internals.Preload.Preloads {
		if p.Path == path {
			return p
		}
	}
	return nil
}

func countPreloadPaths[T attrs.Definer](qs *queries.QuerySet[T], path string) int {
	var cnt int
	internals := qs.Internals()
	if internals.Preload == nil {
		return 0
	}
	for _, p := range internals.Preload.Preloads {
		if p.Path == path {
			cnt++
		}
	}
	return cnt
}

func preloadInternals(t *testing.T, preloadQS *queries.QuerySet[attrs.Definer]) *queries.QuerySetInternals {
	t.Helper()
	return preloadQS.Internals()
}

func assertNoDuplicateSelectColumns(t *testing.T, preloadQS *queries.QuerySet[attrs.Definer]) {
	t.Helper()
	sqlStr := preloadQS.QueryAll().SQL()
	selectIdx := strings.Index(sqlStr, "SELECT ")
	fromIdx := strings.Index(sqlStr, " FROM ")
	if selectIdx < 0 || fromIdx < 0 || fromIdx <= selectIdx {
		t.Fatalf("unexpected SQL layout: %s", sqlStr)
	}
	colsPart := sqlStr[selectIdx+len("SELECT ") : fromIdx]
	colsPart = strings.TrimPrefix(colsPart, "DISTINCT ")
	cols := strings.Split(colsPart, ",")
	seen := make(map[string]int, len(cols))
	for _, col := range cols {
		norm := strings.TrimSpace(col)
		seen[norm]++
		if seen[norm] > 1 {
			t.Fatalf("duplicate SELECT column %q in SQL:\n%s", norm, sqlStr)
		}
	}
}

func assertNoDuplicateJoinAliases(t *testing.T, preloadQS *queries.QuerySet[attrs.Definer]) {
	t.Helper()
	internals := preloadInternals(t, preloadQS)
	seenAliases := make(map[string]string, len(internals.Joins))
	for _, j := range internals.Joins {
		cond := ""
		if j.JoinDefCondition != nil {
			cond = j.JoinDefCondition.String()
		}
		alias := j.Table.Alias
		if alias == "" {
			alias = j.Table.Name
		}
		if prevCond, exists := seenAliases[alias]; exists {
			t.Fatalf("duplicate join table/alias %q (prev=%q, cur=%q)", alias, prevCond, cond)
		}
		seenAliases[alias] = cond
	}
}

func TestPreloadQuerySetMerge(t *testing.T) {
	t.Run("01_StringPreloadThenPreloadWithQuerySet", func(t *testing.T) {
		f := seedMergeFixtures(t)
		subQS := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload("Items").
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: subQS})

		if cnt := countPreloadPaths(qs, "Items"); cnt != 1 {
			t.Fatalf("expected 1 preload entry for Items, got %d", cnt)
		}
		rows, err := qs.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows) != 1 || len(rows[0].Object.Items.AsList()) != 3 {
			t.Fatalf("expected 1 order with 3 items, got %+v", rows)
		}
		if rows[0].Object.Items.AsList()[0].Product.Title == "" {
			t.Fatal("expected preloaded Product.Title to be populated")
		}
	})

	t.Run("02_PreloadWithQuerySetThenStringPreloadPreservesQuerySet", func(t *testing.T) {
		f := seedMergeFixtures(t)
		subQS := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).
			Select("*", "Product.Title").
			Filter("Quantity__gte", 5)

		qs := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: subQS}).
			Preload("Items")

		rows, err := qs.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows[0].Object.Items.AsList()) != 1 || rows[0].Object.Items.AsList()[0].ID != f.item2.ID {
			t.Fatalf("expected string Preload not to overwrite filter QuerySet, got %d items", len(rows[0].Object.Items.AsList()))
		}
	})

	t.Run("03_NoJoinsPreloadThenCustomQuerySet", func(t *testing.T) {
		f := seedMergeFixtures(t)
		subQS := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")
		qs := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.NoJoins("Items")).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: subQS})

		rows, err := qs.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if rows[0].Object.Items.AsList()[0].Product.Price == 0 {
			t.Fatal("expected Product.Price to be loaded")
		}
	})

	t.Run("04_EmptySelectPlusExplicitField", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("Note")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows[0].Object.Items.AsList()) != 3 {
			t.Fatalf("expected 3 items, got %d", len(rows[0].Object.Items.AsList()))
		}
	})

	t.Run("05_DisjointRootFieldsMerged", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("ID", "Order", "Quantity")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("ID", "Order", "Note")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		var gotItem *MergeOrderItem
		for _, it := range rows[0].Object.Items.AsList() {
			if it.ID == f.item1.ID {
				gotItem = it
			}
		}
		if gotItem == nil || gotItem.Quantity != 2 || gotItem.Note != "urgent" {
			t.Fatalf("expected both Quantity=2 and Note='urgent', got %+v", gotItem)
		}
	})

	t.Run("06_IdenticalRootFieldsDeduplicated", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("ID", "Order", "Quantity", "Note")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("ID", "Order", "Quantity", "Note")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		sqlStr := strings.ReplaceAll(p.QuerySet.QueryAll().SQL(), `"`, "`")
		if strings.Count(sqlStr, "`quantity`") != 1 || strings.Count(sqlStr, "`note`") != 1 {
			t.Fatalf("expected each root column once, got SQL: %s", sqlStr)
		}
	})

	t.Run("07_DisjointRelatedProductFieldsMerged", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title", "Product.Slug")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Price", "Product.Stock")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		var item1 *MergeOrderItem
		for _, it := range rows[0].Object.Items.AsList() {
			if it.ID == f.item1.ID {
				item1 = it
			}
		}
		if item1 == nil || item1.Product.Title != "Mech Keyboard" || item1.Product.Slug != "mech-keyboard" ||
			item1.Product.Price != 120 || item1.Product.Stock != 10 {
			t.Fatalf("expected all Product fields from both QuerySets, got %+v", item1.Product)
		}
	})

	t.Run("08_OverlappingRelatedProductFieldsDeduplicated", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows[0].Object.Items.AsList()) != 3 {
			t.Fatalf("expected 3 items, got %d", len(rows[0].Object.Items.AsList()))
		}
	})

	t.Run("09_RelatedWildcardThenSpecificRelatedField", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price", "Product.Title")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if rows[0].Object.Items.AsList()[0].Product.Slug == "" {
			t.Fatal("expected wildcard Product fields preserved")
		}
	})

	t.Run("10_SpecificRelatedFieldThenRelatedWildcard", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.*")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if rows[0].Object.Items.AsList()[0].Product.Title == "" || rows[0].Object.Items.AsList()[0].Product.Price == 0 {
			t.Fatal("expected both Price and remaining wildcard Product fields loaded")
		}
	})

	t.Run("11_TwoRelatedWildcardsSameRelation", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.*")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
	})

	t.Run("12_RootWildcardThenSingleRootField", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("Note")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
	})

	t.Run("13_SingleRootFieldThenRootWildcard", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("ID", "Order", "Note")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if rows[0].Object.Items.AsList()[0].Discount == 0 {
			t.Fatal("expected wildcard root fields like Discount to be loaded")
		}
	})

	t.Run("14_ThreeWayPreloadMergeAccumulatesAllFields", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Slug")
		qs3 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price", "Product.Stock")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs3})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		prod := rows[0].Object.Items.AsList()[0].Product
		if prod.Title == "" || prod.Slug == "" || prod.Price == 0 || prod.Stock == 0 {
			t.Fatalf("expected all 3-way merged Product fields, got %+v", prod)
		}
	})

	t.Run("15_FourWayMergeMixedWildcardAndNested", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title", "Product.Category.Title")
		qs3 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Category.Slug", "Product.Price")
		qs4 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Stock").OrderBy("Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs3}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs4})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		items := rows[0].Object.Items.AsList()
		if len(items) != 3 || items[0].Product.Price != 15 || items[1].Product.Price != 120 || items[2].Product.Price != 450 {
			t.Fatalf("expected price ascending items (15,120,450), got %+v", items)
		}
		if items[0].Product.Category.Title != "Keyboards" || items[0].Product.Category.Slug != "keyboards" {
			t.Fatalf("expected nested Category Title & Slug merged, got %+v", items[0].Product.Category)
		}
	})

	t.Run("16_SameRelationJoinMergedWithoutDuplicateJoin", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if len(preloadInternals(t, p.QuerySet).Joins) != 1 {
			t.Fatalf("expected 1 join for shared Product relation, got %d", len(preloadInternals(t, p.QuerySet).Joins))
		}
	})

	t.Run("17_DifferentOrderOfJoinsRemapsAliasNoDuplicateJoin", func(t *testing.T) {
		f := seedMergeFixtures(t)
		// qs1 walks SecondaryProduct first (T_merge_products) then Product (T1_merge_products)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "SecondaryProduct.Title", "Product.Title")
		// qs2 walks Product alone (T_merge_products) — alias differs from qs1's Product (T1_merge_products)!
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		if joins := preloadInternals(t, p.QuerySet).Joins; len(joins) != 2 {
			t.Fatalf("expected exactly 2 joins (SecondaryProduct + Product), got %d", len(joins))
		}

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		var item1 *MergeOrderItem
		for _, it := range rows[0].Object.Items.AsList() {
			if it.ID == f.item1.ID {
				item1 = it
			}
		}
		if item1.Product.Title != "Mech Keyboard" || item1.Product.Price != 120 || item1.SecondaryProduct.Title != "USB Cable" {
			t.Fatalf("expected remapped Product fields and SecondaryProduct intact, got prod=%+v sec=%+v", item1.Product, item1.SecondaryProduct)
		}
	})

	t.Run("18_DisjointJoinsToSameTargetModelGetUniqueAliases", func(t *testing.T) {
		f := seedMergeFixtures(t)
		// Both qs1 and qs2 independently allocate T_merge_products at counter=0!
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "SecondaryProduct.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		if len(preloadInternals(t, p.QuerySet).Joins) != 2 {
			t.Fatalf("expected 2 disjoint joins with distinct aliases, got %d", len(preloadInternals(t, p.QuerySet).Joins))
		}

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		var item1 *MergeOrderItem
		for _, it := range rows[0].Object.Items.AsList() {
			if it.ID == f.item1.ID {
				item1 = it
			}
		}
		if item1.Product.Title != "Mech Keyboard" || item1.SecondaryProduct.Price != 15 {
			t.Fatalf("unexpected disjoint join values: prod=%+v sec=%+v", item1.Product, item1.SecondaryProduct)
		}
	})

	t.Run("19_SelfReferentialNestedChainMerge", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Category.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Category.ID", "Product.Category.Parent.Title")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		if len(preloadInternals(t, p.QuerySet).Joins) != 3 {
			t.Fatalf("expected 3 joins (Product, Category, Parent), got %d", len(preloadInternals(t, p.QuerySet).Joins))
		}

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		cat := rows[0].Object.Items.AsList()[0].Product.Category
		if cat.Title == "" || cat.Parent == nil || cat.Parent.Title != "Electronics" {
			t.Fatalf("expected self-referential Category.Parent merged, got %+v", cat)
		}
	})

	t.Run("20_DeepMultiHopNestedMerge", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Supplier.Name")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Supplier.ID", "Product.Supplier.Address.ID", "Product.Supplier.Address.Country.Code")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		sup := rows[0].Object.Items.AsList()[0].Product.Supplier
		if sup.Name != "Alpha Supply" || sup.Address.Country.Code != "NL" {
			t.Fatalf("expected 4-hop nested Supplier+Country fields, got %+v", sup)
		}
	})

	t.Run("21_DisjointMultiHopChainsToSameLeafModel", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.ID", "Product.Category.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "SecondaryProduct.ID", "SecondaryProduct.Category.Slug")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		if len(preloadInternals(t, p.QuerySet).Joins) != 4 {
			t.Fatalf("expected 4 joins across two Category chains, got %d", len(preloadInternals(t, p.QuerySet).Joins))
		}

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		var item2 *MergeOrderItem
		for _, it := range rows[0].Object.Items.AsList() {
			if it.ID == f.item2.ID {
				item2 = it
			}
		}
		if item2.Product.Category.Title != "Keyboards" || item2.SecondaryProduct.Category.Slug != "monitors" {
			t.Fatalf("unexpected leaf values: prodCat=%+v secCat=%+v", item2.Product.Category, item2.SecondaryProduct.Category)
		}
	})

	t.Run("22_WhereClausesCombinedWithAND", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Filter("Discount__gte", 10)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Filter("Quantity__gt", 1)

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows[0].Object.Items.AsList()) != 1 || rows[0].Object.Items.AsList()[0].ID != f.item1.ID {
			t.Fatalf("expected only item1 (Discount>=10 AND Quantity>1), got %d items", len(rows[0].Object.Items.AsList()))
		}
	})

	t.Run("23_WhereWithJoinCombinedWithSelect", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Filter("Product.Price__gte", 100)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows[0].Object.Items.AsList()) != 2 {
			t.Fatalf("expected 2 items with Product.Price >= 100, got %d", len(rows[0].Object.Items.AsList()))
		}
		for _, it := range rows[0].Object.Items.AsList() {
			if it.Product.Title == "" {
				t.Fatal("expected Product.Title selected from qs2")
			}
		}
	})

	t.Run("24_BothSidesFilterDifferentRelatedModels", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title").Filter("Product.Price__gte", 100)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "SecondaryProduct.Title").Filter("SecondaryProduct.Price__lte", 20)

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		// Only item1 has Product.Price=120 (>=100) AND SecondaryProduct.Price=15 (<=20)
		if len(rows[0].Object.Items.AsList()) != 1 || rows[0].Object.Items.AsList()[0].ID != f.item1.ID {
			t.Fatalf("expected only item1 matching both filters, got %d items", len(rows[0].Object.Items.AsList()))
		}
	})

	t.Run("25_OrderByAddedToUnorderedExisting", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price").OrderBy("-Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		items := rows[0].Object.Items.AsList()
		if items[0].Product.Price != 450 || items[1].Product.Price != 120 || items[2].Product.Price != 15 {
			t.Fatalf("expected descending Price (450, 120, 15), got %d,%d,%d", items[0].Product.Price, items[1].Product.Price, items[2].Product.Price)
		}
	})

	t.Run("26_MergedQuerySetOrderByOverridesExisting", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").OrderBy("Quantity")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").OrderBy("-Discount")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		orderBy := p.QuerySet.Internals().OrderBy
		if len(orderBy) != 1 || !orderBy[0].Desc || orderBy[0].Column.FieldColumn.Name() != "Discount" {
			t.Fatalf("expected merged qs OrderBy (-Discount) to override existing, got %+v", orderBy)
		}
	})

	t.Run("27_SameOrderByFromMergedQuerySetNotDuplicated", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price").OrderBy("Product.Price")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price").OrderBy("Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if len(p.QuerySet.Internals().OrderBy) != 1 {
			t.Fatalf("expected 1 OrderBy clause, got %d (%s)", len(p.QuerySet.Internals().OrderBy), p.QuerySet.QueryAll().SQL())
		}
	})

	t.Run("28_SameColumnMergedQuerySetDescOverridesAsc", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").OrderBy("Quantity")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").OrderBy("-Quantity")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		orderBy := p.QuerySet.Internals().OrderBy
		if len(orderBy) != 1 || !orderBy[0].Desc || orderBy[0].Column.FieldColumn.Name() != "Quantity" {
			t.Fatalf("expected merged qs -Quantity to override Quantity, got %+v", orderBy)
		}
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		items := rows[0].Object.Items.AsList()
		if items[0].Quantity != 5 || items[1].Quantity != 2 || items[2].Quantity != 1 {
			t.Fatalf("expected descending Quantity (5,2,1), got %+v", items)
		}
	})

	t.Run("29_OrderByWithDifferentAliasNumberingRemapped", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "SecondaryProduct.Title", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price").OrderBy("Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		items := rows[0].Object.Items.AsList()
		if items[0].Product.Price != 15 || items[2].Product.Price != 450 {
			t.Fatalf("expected remapped OrderBy(Product.Price) asc, got %+v", items)
		}
	})

	t.Run("30_DistinctMergedFromExisting", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Distinct()
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if !p.QuerySet.Peek().Distinct {
			t.Fatal("expected Distinct=true preserved from existing QuerySet")
		}
	})

	t.Run("31_DistinctMergedFromIncoming", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title").Distinct()

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if !p.QuerySet.Peek().Distinct {
			t.Fatal("expected Distinct=true merged from incoming QuerySet")
		}
	})

	t.Run("32_ForUpdateMerged", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").ForUpdate()

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if !p.QuerySet.Peek().ForUpdate {
			t.Fatal("expected ForUpdate=true merged from incoming QuerySet")
		}
	})

	t.Run("33_LimitOverriddenByCustomLimit", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Limit(5)

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if p.QuerySet.Peek().Limit != 5 {
			t.Fatalf("expected Limit=5, got %d", p.QuerySet.Peek().Limit)
		}
	})

	t.Run("34_ExistingCustomLimitUpdatedByIncomingCustomLimit", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Limit(10)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Limit(3)

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if p.QuerySet.Peek().Limit != 3 {
			t.Fatalf("expected Limit=3, got %d", p.QuerySet.Peek().Limit)
		}
	})

	t.Run("35_DefaultIncomingLimitPreservesExistingCustomLimit", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Limit(7)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if p.QuerySet.Peek().Limit != 7 {
			t.Fatalf("expected existing Limit=7 preserved, got %d", p.QuerySet.Peek().Limit)
		}
	})

	t.Run("36_OffsetPropagatedFromIncoming", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Offset(2)

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		if p.QuerySet.Peek().Offset != 2 {
			t.Fatalf("expected Offset=2, got %d", p.QuerySet.Peek().Offset)
		}
	})

	t.Run("37_SelectRelatedInBothPreloadQuerySets", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).SelectRelated("Product")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).SelectRelated("SecondaryProduct")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		it := rows[0].Object.Items.AsList()[0]
		if it.Product.Title == "" || it.SecondaryProduct.Title == "" {
			t.Fatalf("expected both SelectRelated models loaded, got prod=%+v sec=%+v", it.Product, it.SecondaryProduct)
		}
	})

	t.Run("38_SelectRelatedThenSelectFieldsNoDuplicates", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).SelectRelated("Product")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title", "Note")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if rows[0].Object.Items.AsList()[0].Product.Slug == "" {
			t.Fatal("expected full SelectRelated Product fields retained")
		}
	})

	t.Run("39_SelectFieldsThenSelectRelatedNoDuplicates", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("ID", "Order", "Product", "Product.Title", "Note")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).SelectRelated("Product")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if rows[0].Object.Items.AsList()[0].Product.Slug == "" {
			t.Fatal("expected missing Product fields from SelectRelated added cleanly")
		}
	})

	t.Run("40_NestedPreloadsInsidePreloadQuerySetMergedRecursively", func(t *testing.T) {
		f := seedMergeFixtures(t)
		nested1 := queries.GetQuerySet[attrs.Definer](&MergeProduct{}).Select("ID", "Title", "Category")
		nested2 := queries.GetQuerySet[attrs.Definer](&MergeProduct{}).Select("ID", "Price")

		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").
			Preload(queries.Preload{Path: "Product", NoJoins: true, QuerySet: nested1})
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").
			Preload(queries.Preload{Path: "Product", NoJoins: true, QuerySet: nested2})

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		subP := getPreloadEntry(t, p.QuerySet, "Product")
		if subP == nil || subP.QuerySet == nil {
			t.Fatal("expected merged nested Product preload")
		}
		assertNoDuplicateSelectColumns(t, subP.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		prod := rows[0].Object.Items.AsList()[0].Product
		if prod.Title == "" || prod.Price == 0 {
			t.Fatalf("expected nested merged Product.Title & Product.Price, got %+v", prod)
		}
	})

	t.Run("41_DistinctNestedPreloadsPreserved", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Preload("Product")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*").Preload("SecondaryProduct")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		it := rows[0].Object.Items.AsList()[0]
		if it.Product.Title == "" || it.SecondaryProduct.Title == "" {
			t.Fatalf("expected both nested Preloads preserved, got prod=%+v sec=%+v", it.Product, it.SecondaryProduct)
		}
	})

	t.Run("42_ManyToManyThroughPreloadMerge", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeTag{}).Select("*").Filter("Weight__gte", 5)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeTag{}).Annotate("tag_bonus", expr.Value(99)).Select("*").OrderBy("-Weight")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Tags", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Tags", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Tags")
		assertNoDuplicateSelectColumns(t, p.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		tags := rows[0].Object.Tags.AsList()
		if len(tags) != 2 {
			t.Fatalf("expected 2 tags, got %d", len(tags))
		}
		if tags[0].Object.Name != "vip" || tags[0].Object.Weight != 10 {
			t.Fatalf("expected tags ordered by -Weight with Name & Weight populated, got %+v", tags[0].Object)
		}
		if through, ok := tags[0].Through().(*MergeOrderTag); !ok || through.SortOrder != 2 {
			t.Fatalf("expected Through SortOrder=2 on vip tag, got %#v", tags[0].Through())
		}
	})

	t.Run("43_AnnotationsMergedWithoutOverwrite", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).
			Annotate("ann_qty", expr.Value(42)).
			Select("*")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).
			Annotate("ann_qty", expr.Value(999)).
			Annotate("ann_tag", expr.Value("merged")).
			Select("*")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		item0 := rows[0].Object.Items.AsList()[0]
		if v, ok := item0.DataStore().GetValue("ann_qty"); !ok || v == nil {
			t.Fatalf("expected ann_qty annotation present, got (%v, %v)", v, ok)
		}
		if v, ok := item0.DataStore().GetValue("ann_tag"); !ok || v == nil {
			t.Fatalf("expected ann_tag annotation present, got (%v, %v)", v, ok)
		}
	})

	t.Run("44_GroupByMergedWithoutDuplicateGroupCols", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("Order", "Product.Category.Title").
			GroupBy("Order", "Product.Category.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("Order", "Product.Category.Slug").
			GroupBy("Product.Category.Title", "Product.Category.Slug")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)
		sqlStr := p.QuerySet.QueryAll().SQL()
		if gbIdx := strings.Index(sqlStr, "GROUP BY "); gbIdx < 0 {
			t.Fatalf("expected GROUP BY in SQL: %s", sqlStr)
		} else {
			gbPart := strings.ReplaceAll(sqlStr[gbIdx:], `"`, "`")
			if strings.Count(gbPart, "`title`") != 1 || strings.Count(gbPart, "`slug`") != 1 {
				t.Fatalf("expected deduplicated GROUP BY columns, got: %s", gbPart)
			}
		}
	})

	t.Run("45_OuterJoinedPreloadPlusMergedChildQuerySet", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: false, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: false, QuerySet: qs2})

		assertNoDuplicateSelectColumns(t, getPreloadEntry(t, mainQS, "Items").QuerySet)
		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows) != 1 || len(rows[0].Object.Items.AsList()) != 3 {
			t.Fatalf("expected 1 order with 3 items, got %d", len(rows))
		}
	})

	t.Run("46_OriginalQuerySetsRemainUnmutatedAfterMerge", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price", "SecondaryProduct.Title").OrderBy("Product.Price")
		beforeSQL1 := qs1.QueryAll().SQL()
		beforeSQL2 := qs2.QueryAll().SQL()

		_ = queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		if after1 := qs1.QueryAll().SQL(); after1 != beforeSQL1 {
			t.Fatalf("qs1 mutated:\nbefore: %s\nafter:  %s", beforeSQL1, after1)
		}
		if after2 := qs2.QueryAll().SQL(); after2 != beforeSQL2 {
			t.Fatalf("qs2 mutated:\nbefore: %s\nafter:  %s", beforeSQL2, after2)
		}
	})

	t.Run("47_BranchingCloneIsolationBetweenForkedQuerySets", func(t *testing.T) {
		f := seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")
		qs3 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Stock")

		base := queries.GetQuerySet(&MergeOrder{}).Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1})

		branchA := base.Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})
		branchB := base.Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs3})

		sqlA := getPreloadEntry(t, branchA, "Items").QuerySet.QueryAll().SQL()
		sqlB := getPreloadEntry(t, branchB, "Items").QuerySet.QueryAll().SQL()
		if strings.Contains(sqlA, "`stock`") || strings.Contains(sqlB, "`price`") {
			t.Fatalf("branch mutation leaked across forked QuerySets:\nsqlA=%s\nsqlB=%s", sqlA, sqlB)
		}
	})

	t.Run("48_MultiSegmentPreloadPathDirectMerge", func(t *testing.T) {
		f := seedMergeFixtures(t)
		pQS1 := queries.GetQuerySet[attrs.Definer](&MergeProduct{}).Select("ID", "Title")
		pQS2 := queries.GetQuerySet[attrs.Definer](&MergeProduct{}).Select("ID", "Price", "Category.Title")

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items.Product", NoJoins: true, QuerySet: pQS1}).
			Preload(queries.Preload{Path: "Items.Product", NoJoins: true, QuerySet: pQS2})

		p := getPreloadEntry(t, mainQS, "Items.Product")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		assertNoDuplicateJoinAliases(t, p.QuerySet)

		rows, err := mainQS.All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		prod := rows[0].Object.Items.AsList()[0].Product
		if prod.Title == "" || prod.Price == 0 || prod.Category.Title == "" {
			t.Fatalf("expected multi-segment Items.Product merge fields, got %+v", prod)
		}
	})

	t.Run("49_HavingClausesCombinedAndExecuted", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("Order", "Product.Title").
			GroupBy("Order", "Product.Title").
			Having("Quantity__gte", 1)
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("Order", "Product.Price").
			GroupBy("Order", "Product.Price").
			Having("Discount__gte", 0)

		mainQS := queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2})

		p := getPreloadEntry(t, mainQS, "Items")
		assertNoDuplicateSelectColumns(t, p.QuerySet)
		if len(p.QuerySet.Peek().Having) != 2 {
			t.Fatalf("expected 2 Having clauses merged, got %d", len(p.QuerySet.Peek().Having))
		}
	})

	t.Run("50_EmptyParentResultSetSkipsMergedPreloadCleanly", func(t *testing.T) {
		_ = seedMergeFixtures(t)
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title")
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price")

		rows, err := queries.GetQuerySet(&MergeOrder{}).
			Filter("Reference", "NON-EXISTENT").
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2}).
			All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(rows) != 0 {
			t.Fatalf("expected 0 orders, got %d", len(rows))
		}
	})

	t.Run("51_ForEachRowHooksCombinedInOrder", func(t *testing.T) {
		f := seedMergeFixtures(t)
		var callOrder []string
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Title").
			ForEachRow(func(q *queries.QuerySet[attrs.Definer], row *queries.Row[attrs.Definer]) error {
				callOrder = append(callOrder, fmt.Sprintf("1:%d", row.Object.(*MergeOrderItem).ID))
				return nil
			})
		qs2 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*", "Product.Price").
			ForEachRow(func(q *queries.QuerySet[attrs.Definer], row *queries.Row[attrs.Definer]) error {
				callOrder = append(callOrder, fmt.Sprintf("2:%d", row.Object.(*MergeOrderItem).ID))
				return nil
			})

		_, err := queries.GetQuerySet(&MergeOrder{}).
			Filter("ID", f.order1.ID).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs2}).
			All()
		if err != nil {
			t.Fatalf("All() failed: %v", err)
		}
		if len(callOrder) != 6 || !strings.HasPrefix(callOrder[0], "1:") || !strings.HasPrefix(callOrder[1], "2:") {
			t.Fatalf("expected both ForEachRow callbacks called per row in order, got %v", callOrder)
		}
	})

	t.Run("52_MismatchedModelPreloadQuerySetPanics", func(t *testing.T) {
		qs1 := queries.GetQuerySet[attrs.Definer](&MergeOrderItem{}).Select("*")
		qsWrong := queries.GetQuerySet[attrs.Definer](&MergeProduct{}).Select("*")

		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic when merging preload QuerySets of different models")
			}
		}()

		_ = queries.GetQuerySet(&MergeOrder{}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qs1}).
			Preload(queries.Preload{Path: "Items", NoJoins: true, QuerySet: qsWrong})
	})
}
