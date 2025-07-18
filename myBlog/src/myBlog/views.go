package myblog

import (
	"net/http"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
)

func Index(w http.ResponseWriter, r *http.Request) {
	var context = ctx.RequestContext(r)

	context.Set("Title", "Home")
	context.Set("ProjectURL", "https://github.com/nigel2392/go-django")
	context.Set("ProjectName", "GO-Django")

	if err := tpl.FRender(w, context, "core", "myBlog/index.tmpl"); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

}
