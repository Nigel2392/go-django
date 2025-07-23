package formsapp

import (
	"fmt"
	"net/http"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

func Index(w http.ResponseWriter, r *http.Request) {
	var context = ctx.RequestContext(r)

	context.Set("Title", "Home")
	context.Set("ProjectURL", "https://github.com/nigel2392/go-django")
	context.Set("ContactURL", django.Reverse("contact"))
	context.Set("ProjectName", "GO-Django")

	if err := tpl.FRender(w, context, "core", "formsapp/index.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func Contact(w http.ResponseWriter, r *http.Request) {
	var context = ctx.RequestContext(r)

	context.Set("Title", "Contact")
	context.Set("ProjectURL", "https://github.com/nigel2392/go-django")
	context.Set("ContactURL", django.Reverse("contact"))
	context.Set("ProjectName", "GO-Django")

	var form = NewContactForm(r)
	if r.Method == http.MethodPost {
		fmt.Println()

		if form.IsValid() {
			var data = form.CleanedData()
			fmt.Println("Form is valid!")
			fmt.Printf("Name (%T): %s\n", data["name"], data["name"])
			fmt.Printf("Email (%T): %s\n", data["email"], data["email"])
			fmt.Printf("Subject (%T): %s\n", data["subject"], data["subject"])
			fmt.Printf("Message (%T): %s\n", data["message"], data["message"])

			http.Redirect(w, r, django.Reverse("index"), http.StatusSeeOther)
			return
		}

		fmt.Println("Form is invalid!")
		for _, err := range form.ErrorList() {
			fmt.Printf("Error: %s\n", err)
		}

		var errs = form.BoundErrors()
		if errs != nil {
			for head := errs.Front(); head != nil; head = head.Next() {
				fmt.Printf("Error [%s]: %s\n", head.Key, head.Value)
			}
		}
	}

	context.Set("Form", form)
	if err := tpl.FRender(w, context, "core", "formsapp/contact.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}
