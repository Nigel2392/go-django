package adminviews_test

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/contrib/admin/chooser"
	"github.com/Nigel2392/go-django/contrib/shop/coupons"
	"github.com/Nigel2392/go-django/contrib/shop/models"
	"github.com/Nigel2392/go-django/contrib/shop/products"
	"github.com/Nigel2392/go-django/djester"
	queries "github.com/Nigel2392/go-django/queries/src"
	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/forms/formsets"
	"github.com/PuerkitoBio/goquery"
)

func extractAdminFormValues(doc *goquery.Document) map[string]any {
	values := make(map[string]any)
	form := doc.Find("main form").First()
	if form.Length() == 0 {
		form = doc.Find("form").Last()
	}
	form.Find("input[name], select[name], textarea[name]").Each(func(_ int, s *goquery.Selection) {
		name, ok := s.Attr("name")
		if !ok || name == "" || strings.Contains(name, "__INDEX__") {
			return
		}
		if s.Is("input") {
			inputType := strings.ToLower(s.AttrOr("type", "text"))
			if inputType == "submit" || inputType == "button" {
				return
			}
			if inputType == "checkbox" || inputType == "radio" {
				if _, checked := s.Attr("checked"); checked {
					values[name] = s.AttrOr("value", "on")
				}
				return
			}
			values[name] = s.AttrOr("value", "")
			return
		}
		if s.Is("textarea") {
			values[name] = s.Text()
			return
		}
		if s.Is("select") {
			selected := s.Find("option[selected]").First()
			if selected.Length() > 0 {
				values[name] = selected.AttrOr("value", "")
			} else if first := s.Find("option").First(); first.Length() > 0 {
				values[name] = first.AttrOr("value", "")
			} else {
				values[name] = ""
			}
		}
	})
	return values
}

func adminCouponUITests() []djester.Test {
	var p1, p2, p3 *products.Product

	couponProductIDs := func(t *testing.T, couponID uint64) []string {
		t.Helper()
		rows, err := queries.GetQuerySet(&coupons.CouponProductThrough{}).
			Filter("Coupon", couponID).
			All()
		if err != nil {
			t.Fatalf("failed to query CouponProductThrough: %v", err)
		}
		ids := make([]string, 0, len(rows))
		for _, r := range rows {
			if r.Object.Product != nil {
				ids = append(ids, r.Object.Product.ID.String())
			}
		}
		slices.Sort(ids)
		return ids
	}

	return []djester.Test{
		&djester.BasicTest{
			Label: "Coupon Seed products for coupon admin UI tests",
			Function: func(d *djester.Tester, t *testing.T) {
				cat := &models.Category{Name: "Coupon Apparel", Slug: "coupon-apparel"}
				if _, err := queries.GetQuerySet(&models.Category{}).Create(cat); err != nil {
					t.Fatalf("failed to seed category: %v", err)
				}
				_, p1 = seedAdminProduct(t, cat, "alpha_supplier", "Merino Shirt", "merino-shirt", "SKU-SHIRT-1", "49.99", "0.00")
				_, p2 = seedAdminProduct(t, cat, "alpha_supplier", "Canvas Backpack", "canvas-backpack", "SKU-PACK-2", "89.99", "0.00")
				_, p3 = seedAdminProduct(t, cat, "alpha_supplier", "Running Shoes", "running-shoes", "SKU-SHOE-3", "119.99", "0.00")
			},
		},
		&djester.BasicTest{
			Label: "Coupon Add/Edit UI end-to-end chooser, error rendering, duplicate rejection, and updates",
			Function: func(d *djester.Tester, t *testing.T) {
				asserter := djester.Asserter(t, true)
				loginResp, err := d.Login("admin_user")
				asserter.Assert(err == nil, "Login failed: %v", err)
				loginResp.Body.Close()
				defer d.Logout()

				addURL := django.Reverse("admin:apps:model:add", "shop", "coupons")
				listURL := django.Reverse("admin:apps:model", "shop", "coupons")
				expectedChooserURL := django.Reverse("admin:apps:model:chooser:list", "shop", "products", chooser.DEFAULT_KEY)

				// 1. GET Add view renders inline formset, management inputs, empty template, and product chooser widget.
				getAddResp, err := d.Get(addURL, nil, nil)
				asserter.Assert(err == nil, "GET coupon add failed: %v", err)
				asserter.AssertEqual(http.StatusOK, getAddResp.StatusCode)
				addDoc, err := getAddResp.DOM()
				asserter.Assert(err == nil, "Failed to parse Add HTML: %v", err)
				getAddResp.Body.Close()

				asserter.AssertEqual("1", addDoc.Find(`input[name="Products-management-TOTAL_FORMS"]`).AttrOr("value", ""))
				asserter.AssertEqual("1", addDoc.Find(`input[name="Products-management-INITIAL_FORMS"]`).AttrOr("value", ""))
				asserter.AssertEqual("1", addDoc.Find(`input[name="Products-management-MIN_NUM_FORMS"]`).AttrOr("value", ""))
				asserter.Assert(addDoc.Find(`template.modelform-panel__empty-form`).Length() > 0, "expected inline empty-form template")

				chooserEl := addDoc.Find(`[data-controller="chooser"]`).First()
				asserter.Assert(chooserEl.Length() == 1, "expected chooser widget on Products-0")
				listChooserURL := chooserEl.AttrOr("data-chooser-listurl-value", "")
				asserter.AssertEqual(expectedChooserURL, listChooserURL)
				asserter.AssertEqual(1, addDoc.Find(`input[data-chooser-target="input"][name="Products-0-Product"]`).Length())

				// 2. Open product chooser list modal via DOM URL (JSON response with .HTML) and verify seeded products and search work.
				chooserResp, err := d.Get(listChooserURL, nil, nil)
				asserter.Assert(err == nil, "GET chooser list failed: %v", err)
				asserter.AssertEqual(http.StatusOK, chooserResp.StatusCode)
				var chooserPayload chooser.ChooserResponse
				asserter.Assert(chooserResp.JSON(&chooserPayload) == nil, "Decode chooser list JSON failed")
				chooserResp.Body.Close()
				chooserDoc, err := goquery.NewDocumentFromReader(strings.NewReader(chooserPayload.HTML))
				asserter.Assert(err == nil, "Failed to parse chooser HTML: %v", err)

				for _, prod := range []*products.Product{p1, p2, p3} {
					row := chooserDoc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%s"]`, prod.ID.String()))
					asserter.Assert(row.Length() == 1, "expected chooser row for product %s (%s)", prod.Title, prod.ID.String())
					asserter.AssertEqual(prod.Title, row.AttrOr("data-chooser-preview", ""))
				}

				if searchInput := chooserDoc.Find(`input.godjango-chooser-list-search-input`); searchInput.Length() > 0 {
					searchParam := searchInput.AttrOr("name", "")
					if searchParam != "" {
						searchResp, err := d.Get(listChooserURL, nil, url.Values{searchParam: {"Canvas"}})
						asserter.Assert(err == nil, "GET chooser search failed: %v", err)
						asserter.AssertEqual(http.StatusOK, searchResp.StatusCode)
						var searchPayload chooser.ChooserResponse
						asserter.Assert(searchResp.JSON(&searchPayload) == nil, "Decode chooser search JSON failed")
						searchResp.Body.Close()
						searchDoc, err := goquery.NewDocumentFromReader(strings.NewReader(searchPayload.HTML))
						asserter.Assert(err == nil, "Failed to parse chooser search HTML: %v", err)
						asserter.AssertEqual(1, searchDoc.Find(fmt.Sprintf(`tr.godjango-chooser-list-group[data-chooser-value="%s"]`, p2.ID.String())).Length())
					}
				}

				// 3. POST Add missing required fields -> 200 with error-list in HTML, 0 rows persisted.
				missingData := extractAdminFormValues(addDoc)
				missingData["Title"] = ""
				missingData["Code"] = ""
				missingData["MaxDiscount"] = ""
				missingData["Products-0-Product"] = ""
				errResp, err := d.PostForm(addURL, nil, nil, missingData)
				asserter.Assert(err == nil, "POST Add missing fields failed: %v", err)
				asserter.AssertEqual(http.StatusOK, errResp.StatusCode)
				asserter.AssertEqual(addURL, errResp.Request.URL.Path)
				errDoc, err := errResp.DOM()
				asserter.Assert(err == nil, "Parse missing-field error HTML failed: %v", err)
				errResp.Body.Close()
				asserter.Assert(errDoc.Find(`ul.error-list`).Length() > 0, "expected error-list rendered in add.tmpl")
				if ct, _ := queries.GetQuerySet(&coupons.Coupon{}).Count(); ct != 0 {
					t.Fatalf("expected 0 coupons on validation error, got %d", ct)
				}

				// 4. POST Add duplicate product (p1 selected in both rows) -> 200 with unpacked duplicate error rendered in HTML.
				dupAddData := extractAdminFormValues(addDoc)
				dupAddData["Title"] = "Duplicate Add Promo"
				dupAddData["Code"] = "DUPADD2026"
				dupAddData["MaxDiscount"] = "15.00"
				dupAddData["Products-management-TOTAL_FORMS"] = "2"
				dupAddData["Products-0-Product"] = p1.ID.String()
				dupAddData["Products-0-"+formsets.ORDERING_FIELD_NAME] = "0"
				dupAddData["Products-0-"+formsets.DELETION_FIELD_NAME] = "false"
				dupAddData["Products-1-Product"] = p1.ID.String()
				dupAddData["Products-1-"+formsets.ORDERING_FIELD_NAME] = "1"
				dupAddData["Products-1-"+formsets.DELETION_FIELD_NAME] = "false"

				dupAddResp, err := d.PostForm(addURL, nil, nil, dupAddData)
				asserter.Assert(err == nil, "POST Add duplicate products failed: %v", err)
				asserter.AssertEqual(http.StatusOK, dupAddResp.StatusCode)
				asserter.AssertEqual(addURL, dupAddResp.Request.URL.Path)
				dupAddDoc, err := dupAddResp.DOM()
				asserter.Assert(err == nil, "Parse duplicate Add error HTML failed: %v", err)
				dupAddResp.Body.Close()

				errText := strings.ToLower(dupAddDoc.Find(`ul.error-list`).Text())
				asserter.Assert(dupAddDoc.Find(`ul.error-list`).Length() > 0, "expected error-list on duplicate product add")
				asserter.Assert(strings.Contains(errText, "duplicate"), "expected duplicate error in error-list, got %q", errText)
				if ct, _ := queries.GetQuerySet(&coupons.Coupon{}).Count(); ct != 0 {
					t.Fatalf("expected 0 coupons on duplicate add rejection, got %d", ct)
				}
				if ct, _ := queries.GetQuerySet(&coupons.CouponProductThrough{}).Count(); ct != 0 {
					t.Fatalf("expected 0 through rows on duplicate add rejection, got %d", ct)
				}

				// 5. POST Add with 2 distinct products (leaving blank optional decimal & date inputs) -> redirects and persists.
				beforeProdsCount, _ := queries.GetQuerySet(&products.Product{}).Count()
				goodAddData := extractAdminFormValues(addDoc)
				goodAddData["Title"] = "Winter Promo"
				goodAddData["Code"] = "WINTER2026"
				goodAddData["ShortDescription"] = "Seasonal discount"
				goodAddData["MaxDiscount"] = "25.00"
				goodAddData["MinOrderAmount"] = ""
				goodAddData["ValidFrom"] = ""
				goodAddData["ValidUntil"] = ""
				goodAddData["Products-management-TOTAL_FORMS"] = "2"
				goodAddData["Products-0-Product"] = p1.ID.String()
				goodAddData["Products-0-"+formsets.ORDERING_FIELD_NAME] = "0"
				goodAddData["Products-0-"+formsets.DELETION_FIELD_NAME] = "false"
				goodAddData["Products-1-Product"] = p2.ID.String()
				goodAddData["Products-1-"+formsets.ORDERING_FIELD_NAME] = "1"
				goodAddData["Products-1-"+formsets.DELETION_FIELD_NAME] = "false"

				okAddResp, err := d.PostForm(addURL, nil, nil, goodAddData)
				asserter.Assert(err == nil, "POST valid coupon failed: %v", err)
				asserter.AssertEqual(http.StatusOK, okAddResp.StatusCode)
				asserter.AssertEqual(listURL, okAddResp.Request.URL.Path)
				okAddResp.Body.Close()

				couponRow, err := queries.GetQuerySet(&coupons.Coupon{}).Filter("Code", "WINTER2026").Get()
				asserter.Assert(err == nil, "failed to query saved coupon: %v", err)
				savedCoupon := couponRow.Object

				wantTwo := []string{p1.ID.String(), p2.ID.String()}
				slices.Sort(wantTwo)
				if got := couponProductIDs(t, savedCoupon.ID); !slices.Equal(got, wantTwo) {
					t.Fatalf("expected linked product IDs %v, got %v", wantTwo, got)
				}
				if afterProdsCount, _ := queries.GetQuerySet(&products.Product{}).Count(); afterProdsCount != beforeProdsCount {
					t.Fatalf("expected chooser add not to create new products (want %d, got %d)", beforeProdsCount, afterProdsCount)
				}

				// 6. GET Edit view renders both products pre-selected in chooser widgets and blank dates cleanly.
				editURL := django.Reverse("admin:apps:model:edit", "shop", "coupons", savedCoupon.ID)
				getEditResp, err := d.Get(editURL, nil, nil)
				asserter.Assert(err == nil, "GET coupon edit failed: %v", err)
				asserter.AssertEqual(http.StatusOK, getEditResp.StatusCode)
				editDoc, err := getEditResp.DOM()
				asserter.Assert(err == nil, "Parse Edit HTML failed: %v", err)
				getEditResp.Body.Close()

				asserter.AssertEqual("2", editDoc.Find(`input[name="Products-management-TOTAL_FORMS"]`).AttrOr("value", ""))
				asserter.AssertEqual("2", editDoc.Find(`input[name="Products-management-INITIAL_FORMS"]`).AttrOr("value", ""))
				row0Val := editDoc.Find(`input[data-chooser-target="input"][name="Products-0-Product"]`).AttrOr("value", "")
				row1Val := editDoc.Find(`input[data-chooser-target="input"][name="Products-1-Product"]`).AttrOr("value", "")
				gotDOMIDs := []string{row0Val, row1Val}
				slices.Sort(gotDOMIDs)
				if !slices.Equal(gotDOMIDs, wantTwo) {
					t.Fatalf("expected Edit DOM chooser inputs %v, got %v", wantTwo, gotDOMIDs)
				}
				editPreviewText := editDoc.Find(`.chooser-widget-preview`).Text()
				asserter.Assert(strings.Contains(editPreviewText, p1.Title), "expected %q in chooser preview, got %q", p1.Title, editPreviewText)
				asserter.Assert(strings.Contains(editPreviewText, p2.Title), "expected %q in chooser preview, got %q", p2.Title, editPreviewText)
				asserter.AssertEqual("", editDoc.Find(`input[name="ValidFrom"]`).AttrOr("value", ""))
				asserter.AssertEqual("", editDoc.Find(`input[name="ValidUntil"]`).AttrOr("value", ""))

				// 7. POST Edit with duplicate product -> rejected with HTML error-list and DB preserved.
				dupEditData := extractAdminFormValues(editDoc)
				dupEditData["Products-1-Product"] = row0Val
				dupEditResp, err := d.PostForm(editURL, nil, nil, dupEditData)
				asserter.Assert(err == nil, "POST Edit duplicate failed: %v", err)
				asserter.AssertEqual(http.StatusOK, dupEditResp.StatusCode)
				asserter.AssertEqual(editURL, dupEditResp.Request.URL.Path)
				dupEditDoc, err := dupEditResp.DOM()
				asserter.Assert(err == nil, "Parse Edit duplicate HTML failed: %v", err)
				dupEditResp.Body.Close()

				dupEditErrText := strings.ToLower(dupEditDoc.Find(`ul.error-list`).Text())
				asserter.Assert(strings.Contains(dupEditErrText, "duplicate"), "expected duplicate error on Edit HTML, got %q", dupEditErrText)
				if got := couponProductIDs(t, savedCoupon.ID); !slices.Equal(got, wantTwo) {
					t.Fatalf("expected DB links unchanged %v after duplicate Edit error, got %v", wantTwo, got)
				}

				// 8. POST Edit unchanged -> succeeds cleanly without losing links.
				unchangedData := extractAdminFormValues(editDoc)
				unchangedResp, err := d.PostForm(editURL, nil, nil, unchangedData)
				asserter.Assert(err == nil, "POST Edit unchanged failed: %v", err)
				asserter.AssertEqual(http.StatusOK, unchangedResp.StatusCode)
				asserter.AssertEqual(listURL, unchangedResp.Request.URL.Path)
				unchangedResp.Body.Close()
				if got := couponProductIDs(t, savedCoupon.ID); !slices.Equal(got, wantTwo) {
					t.Fatalf("expected DB links %v after unchanged save, got %v", wantTwo, got)
				}

				// 9. POST Edit chain-shift (row0: c0 -> c1, row1: c1 -> p3) -> succeeds and persists {c1, p3}.
				shiftRespGet, err := d.Get(editURL, nil, nil)
				asserter.Assert(err == nil, "GET Edit before shift failed: %v", err)
				shiftGetDoc, err := shiftRespGet.DOM()
				asserter.Assert(err == nil, "Parse Edit before shift failed: %v", err)
				shiftRespGet.Body.Close()

				c0 := shiftGetDoc.Find(`input[data-chooser-target="input"][name="Products-0-Product"]`).AttrOr("value", "")
				c1 := shiftGetDoc.Find(`input[data-chooser-target="input"][name="Products-1-Product"]`).AttrOr("value", "")
				asserter.Assert(c0 != "" && c1 != "" && c0 != c1, "expected 2 distinct pre-populated chooser IDs, got %q and %q", c0, c1)
				shiftData := extractAdminFormValues(shiftGetDoc)
				shiftData["Products-0-Product"] = c1
				shiftData["Products-1-Product"] = p3.ID.String()

				shiftPostResp, err := d.PostForm(editURL, nil, nil, shiftData)
				asserter.Assert(err == nil, "POST Edit chain-shift failed: %v", err)
				asserter.AssertEqual(http.StatusOK, shiftPostResp.StatusCode)
				asserter.AssertEqual(listURL, shiftPostResp.Request.URL.Path)
				shiftPostResp.Body.Close()

				wantShifted := []string{c1, p3.ID.String()}
				slices.Sort(wantShifted)
				if got := couponProductIDs(t, savedCoupon.ID); !slices.Equal(got, wantShifted) {
					t.Fatalf("expected %v after UI chain-shift, got %v", wantShifted, got)
				}

				// 10. Reload GET Edit after shift, then delete row 0 and move row 1 -> row0's old product -> persists single link.
				afterShiftResp, err := d.Get(editURL, nil, nil)
				asserter.Assert(err == nil, "GET Edit after shift failed: %v", err)
				afterShiftDoc, err := afterShiftResp.DOM()
				asserter.Assert(err == nil, "Parse Edit after shift failed: %v", err)
				afterShiftResp.Body.Close()

				r0 := afterShiftDoc.Find(`input[data-chooser-target="input"][name="Products-0-Product"]`).AttrOr("value", "")
				r1 := afterShiftDoc.Find(`input[data-chooser-target="input"][name="Products-1-Product"]`).AttrOr("value", "")
				gotAfterShiftDOM := []string{r0, r1}
				slices.Sort(gotAfterShiftDOM)
				if !slices.Equal(gotAfterShiftDOM, wantShifted) {
					t.Fatalf("expected Edit DOM chooser inputs %v after shift, got %v", wantShifted, gotAfterShiftDOM)
				}

				delReAddData := extractAdminFormValues(afterShiftDoc)
				delReAddData["Products-0-"+formsets.DELETION_FIELD_NAME] = "true"
				delReAddData["Products-1-Product"] = r0

				delPostResp, err := d.PostForm(editURL, nil, nil, delReAddData)
				asserter.Assert(err == nil, "POST Edit delete+re-add failed: %v", err)
				asserter.AssertEqual(http.StatusOK, delPostResp.StatusCode)
				asserter.AssertEqual(listURL, delPostResp.Request.URL.Path)
				delPostResp.Body.Close()

				if got := couponProductIDs(t, savedCoupon.ID); !slices.Equal(got, []string{r0}) {
					t.Fatalf("expected [%s] preserved after UI delete+re-add, got %v", r0, got)
				}

				finalEditResp, err := d.Get(editURL, nil, nil)
				asserter.Assert(err == nil, "GET final Edit failed: %v", err)
				finalEditDoc, err := finalEditResp.DOM()
				asserter.Assert(err == nil, "Parse final Edit HTML failed: %v", err)
				finalEditResp.Body.Close()
				asserter.AssertEqual("1", finalEditDoc.Find(`input[name="Products-management-TOTAL_FORMS"]`).AttrOr("value", ""))
				asserter.AssertEqual(r0, finalEditDoc.Find(`input[data-chooser-target="input"][name="Products-0-Product"]`).AttrOr("value", ""))
			},
		},
	}
}
