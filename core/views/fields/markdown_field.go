package fields

import (
	"embed"
	"html/template"
	"io/fs"
	"strings"

	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/tags"
)

type MarkdownField string

func (i *MarkdownField) FormValues(v []string) error {
	if len(v) > 0 {
		*i = MarkdownField(v[0])
	}
	return nil
}

func (i MarkdownField) LabelHTML(_ *request.Request, name string, text string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	b.WriteString(`<label for="`)
	var id, ok = tags["id"]
	if ok && len(id) > 0 {
		b.WriteString(id[0])
	} else {
		b.WriteString(`markdown-input-`)
		b.WriteString(name)
	}
	b.WriteString(`">`)
	b.WriteString(text)
	b.WriteString(`</label>`)
	return ElementType(b.String())
}

func (i MarkdownField) InputHTML(_ *request.Request, name string, tags tags.TagMap) interfaces.Element {
	var b strings.Builder
	if tags.Exists("before") {
		var outputid, ok2 = tags["outputid"]
		b.WriteString(`<div id="`)
		if ok2 && len(outputid) > 0 {
			b.WriteString(outputid[0])
		} else {
			b.WriteString(`markdown-output-`)
			b.WriteString(name)
		}
		b.WriteString(`" class="markdown-output"></div>`)
	}
	b.WriteString(`<textarea style="width:100%;" class="markdown-input" name="`)
	b.WriteString(name)
	b.WriteString(`" id="`)
	var id, ok = tags["id"]
	if ok && len(id) > 0 {
		b.WriteString(id[0])
	} else {
		b.WriteString(`markdown-input-`)
		b.WriteString(name)
	}
	b.WriteString(`" rows="10">`)
	b.WriteString(string(i))
	b.WriteString(`</textarea>`)

	if !tags.Exists("before") {
		var outputid, ok2 = tags["outputid"]
		b.WriteString(`<div id="`)
		if ok2 && len(outputid) > 0 {
			b.WriteString(outputid[0])
		} else {
			b.WriteString(`markdown-output-`)
			b.WriteString(name)
		}
		b.WriteString(`" class="markdown-output"></div>`)
	}
	return ElementType(b.String())
}

var _ = interfaces.Scripter(MarkdownField(""))

func (m MarkdownField) Script() (key string, src template.HTML) {
	key = "markdown"
	var srcBuf = strings.Builder{}
	srcBuf.WriteString(`<script src="/field-static-files/wasm_exec_go.js" type="text/javascript"></script>
	<script type="text/javascript">
		window.WASMLoaded.On(function(){
			let parser = markdownMaker
			let mdInput = document.querySelector(".markdown-input")
			let mdOutput = document.querySelector(".markdown-output")
			parser.markdownifyInput(mdInput, mdOutput)
			mdOutput.innerHTML = parser.markdownify(mdInput.value)
		})
	
	</script>
	`)
	src = template.HTML(srcBuf.String())
	return
}

//go:embed staticfiles/*
var staticFS embed.FS

var StaticHandler = router.NewFSRoute("/field-static-files", "field-static-files", fixStaticFS(staticFS))

func fixStaticFS(f embed.FS) fs.FS {
	var fsys, err = fs.Sub(f, "staticfiles")
	if err != nil {
		panic(err)
	}
	return fsys
}
