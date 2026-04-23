//go:build testing_auth && test && sqlite

package pages

import (
"context"
"fmt"
"net/http"
"net/http/httptest"
"net/url"
"strings"
"testing"

django "github.com/Nigel2392/go-django/src"
"github.com/Nigel2392/go-django/src/forms"
"github.com/Nigel2392/mux/middleware/authentication"
)

func TestDebugInlineFormValidation(t *testing.T) {
resetPagesAdminData(t)
root := createRootNode(t, "Root")
child := createChildPageViaAddView(t, root)
img := createBlogImage(t, child)

data := withInlineManagementForms(url.Values{
"Title":   {blogPageAddData.Get("Title")},
"Slug":    {blogPageAddData.Get("Slug")},
"Summary": {blogPageAddData.Get("Summary")},
"TestBlogImageSet-0-ImageText": {img.ImageText},
"TestBlogImageSet-0-BlogPage":  {fmt.Sprintf("%d", child.ID())},
}, 1)

specific, err := Specific(context.Background(), child, false)
if err != nil {
t.Fatalf("load specific page: %v", err)
}

req := httptest.NewRequest(http.MethodPost, django.Reverse("admin:pages:edit", child.ID()), strings.NewReader(data.Encode()))
req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
req = req.WithContext(authentication.ContextWithUser(req.Context(), testAdminUser))

form := PageEditForm(req, specific)
form.Load()
form.WithData(data, nil, req)

valid := forms.IsValid(req.Context(), form)
t.Logf("Valid: %v", valid)

fs := form.FormSet()
if fs == nil {
t.Fatal("formset is nil")
}

printBoundErrors := func(label string, be interface{ Keys() []string; Get(string) ([]error, bool) }) {
if be == nil {
return
}
for _, k := range be.Keys() {
errs, _ := be.Get(k)
for _, e := range errs {
fmt.Printf("%s[%s]: %v\n", label, k, e)
}
}
}

printBoundErrors("Formset", fs.BoundErrors())

fList, _ := fs.Forms()
for i, f2 := range fList {
printBoundErrors(fmt.Sprintf("SubForm[%d]", i), f2.BoundErrors())
fmt.Printf("SubForm[%d] ErrorList: %v\n", i, f2.ErrorList())
}
}
