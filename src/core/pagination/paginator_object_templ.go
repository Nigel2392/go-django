// Code generated by templ - DO NOT EDIT.

// templ: version: v0.2.778
package pagination

//lint:file-ignore SA4006 This context is only used if a nested component is present.

import "github.com/a-h/templ"
import templruntime "github.com/a-h/templ/runtime"

import "github.com/Nigel2392/go-django/src/forms/fields"
import "strconv"
import "net/url"
import "fmt"
import "context"
import "html/template"
import "strings"

type pageObject[T any] struct {
	num       int
	results   []T
	paginator Pagination[T]
}

func (p *pageObject[T]) HTML(queryParam string, numPageNumbers int, queryParams url.Values) template.HTML {
	var b = new(strings.Builder)
	var cmp = p.Component(queryParam, numPageNumbers, queryParams)
	var ctx = context.Background()
	cmp.Render(ctx, b)
	return template.HTML(b.String())
}

func (p *pageObject[T]) Component(queryParam string, numPageNumbers int, queryParams url.Values) templ.Component {
	return templruntime.GeneratedTemplate(func(templ_7745c5c3_Input templruntime.GeneratedComponentInput) (templ_7745c5c3_Err error) {
		templ_7745c5c3_W, ctx := templ_7745c5c3_Input.Writer, templ_7745c5c3_Input.Context
		if templ_7745c5c3_CtxErr := ctx.Err(); templ_7745c5c3_CtxErr != nil {
			return templ_7745c5c3_CtxErr
		}
		templ_7745c5c3_Buffer, templ_7745c5c3_IsBuffer := templruntime.GetBuffer(templ_7745c5c3_W)
		if !templ_7745c5c3_IsBuffer {
			defer func() {
				templ_7745c5c3_BufErr := templruntime.ReleaseBuffer(templ_7745c5c3_Buffer)
				if templ_7745c5c3_Err == nil {
					templ_7745c5c3_Err = templ_7745c5c3_BufErr
				}
			}()
		}
		ctx = templ.InitializeContext(ctx)
		templ_7745c5c3_Var1 := templ.GetChildren(ctx)
		if templ_7745c5c3_Var1 == nil {
			templ_7745c5c3_Var1 = templ.NopComponent
		}
		ctx = templ.ClearChildren(ctx)
		var count, err = p.Paginator().NumPages()
		if err != nil {
			count = 1
		}
		if count == 1 {
			return nil
		}
		var q string
		if queryParams.Has(queryParam) {
			queryParams.Del(queryParam)
		}
		if len(queryParams) > 0 {
			q = fmt.Sprintf("&%s", queryParams.Encode())
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"pagination\"><section class=\"pagination--paginator\"><div class=\"prev\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if p.HasPrev() {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<a href=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var2 templ.SafeURL = templ.SafeURL(fmt.Sprintf("?%s=%d%s", queryParam, p.Prev(), q))
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(string(templ_7745c5c3_Var2)))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var3 string
			templ_7745c5c3_Var3, templ_7745c5c3_Err = templ.JoinStringErrs(fields.T("Previous"))
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/core/pagination/paginator_object.templ`, Line: 47, Col: 118}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var3))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</a>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		var start = p.PageNum() - numPageNumbers
		if start < 1 {
			start = 1
		}
		var end = p.PageNum() + numPageNumbers
		if end > count {
			end = count
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"page-numbers\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		for i := start; i <= end; i++ {
			if i == p.PageNum() {
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"page-number active\"><a href=\"")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				var templ_7745c5c3_Var4 templ.SafeURL = templ.SafeURL(fmt.Sprintf("?%s=%d%s", queryParam, i, q))
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(string(templ_7745c5c3_Var4)))
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\" disabled>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				var templ_7745c5c3_Var5 string
				templ_7745c5c3_Var5, templ_7745c5c3_Err = templ.JoinStringErrs(strconv.Itoa(i))
				if templ_7745c5c3_Err != nil {
					return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/core/pagination/paginator_object.templ`, Line: 65, Col: 123}
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var5))
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</a></div>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
			} else {
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<div class=\"page-number\"><a href=\"")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				var templ_7745c5c3_Var6 templ.SafeURL = templ.SafeURL(fmt.Sprintf("?%s=%d%s", queryParam, i, q))
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(string(templ_7745c5c3_Var6)))
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\">")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				var templ_7745c5c3_Var7 string
				templ_7745c5c3_Var7, templ_7745c5c3_Err = templ.JoinStringErrs(strconv.Itoa(i))
				if templ_7745c5c3_Err != nil {
					return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/core/pagination/paginator_object.templ`, Line: 69, Col: 114}
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var7))
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
				_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</a></div>")
				if templ_7745c5c3_Err != nil {
					return templ_7745c5c3_Err
				}
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div><div class=\"next\">")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		if p.HasNext() {
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("<a href=\"")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var8 templ.SafeURL = templ.SafeURL(fmt.Sprintf("?%s=%d%s", queryParam, p.Next(), q))
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(string(templ_7745c5c3_Var8)))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("\">")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			var templ_7745c5c3_Var9 string
			templ_7745c5c3_Var9, templ_7745c5c3_Err = templ.JoinStringErrs(fields.T("Next"))
			if templ_7745c5c3_Err != nil {
				return templ.Error{Err: templ_7745c5c3_Err, FileName: `src/core/pagination/paginator_object.templ`, Line: 77, Col: 114}
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString(templ.EscapeString(templ_7745c5c3_Var9))
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
			_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</a>")
			if templ_7745c5c3_Err != nil {
				return templ_7745c5c3_Err
			}
		}
		_, templ_7745c5c3_Err = templ_7745c5c3_Buffer.WriteString("</div></section></div>")
		if templ_7745c5c3_Err != nil {
			return templ_7745c5c3_Err
		}
		return templ_7745c5c3_Err
	})
}

func (p *pageObject[T]) Count() int {
	return len(p.results)
}

func (p *pageObject[T]) Results() []T {
	return p.results
}

func (p *pageObject[T]) Paginator() Pagination[T] {
	return p.paginator
}

func (p *pageObject[T]) HasNext() bool {
	var numpages, err = p.Paginator().NumPages()
	if err != nil {
		return false
	}
	return p.PageNum() < numpages
}

func (p *pageObject[T]) HasPrev() bool {
	return p.num > 0
}

func (p *pageObject[T]) Next() int {
	if p.HasNext() {
		return p.PageNum() + 1
	}
	return -1
}

func (p *pageObject[T]) Prev() int {
	if p.HasPrev() {
		return p.PageNum() - 1
	}
	return -1
}

func (p *pageObject[T]) PageNum() int {
	return p.num + 1
}

var _ = templruntime.GeneratedTemplate
