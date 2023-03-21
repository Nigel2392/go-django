package debug

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Nigel2392/go-django/core/tracer"
	"github.com/Nigel2392/router/v3/middleware/csrf"
	"github.com/Nigel2392/router/v3/request"
)

var CSS = `
body, html {
	margin: 0;
	padding: 0;
}
body {
	font-family: sans-serif;
	background-color: #f2f0e1;
}
h1, h2, h3, h4, h5, h6 {
	margin: 0;
	padding: 0;
}
.container {
	margin: 40px 80px;
}
.container:last-child {
	margin-bottom: 80px;
}
.tracer-spacer {
	margin-top: 30px;
	margin-bottom: 30px;
}
.tracer-spacer-xs {
	margin-top: 10px;
	margin-bottom: 10px;
}
.tracer-header {
	font-size: 1.8em;
	font-weight: bold;
	margin-bottom: 0.5em;
	margin-top: 0;
	padding: 0;
	color: #bb0000;
}
.tracer-subheader {
	font-size: 1.2em;
	font-weight: bold;
	margin-top: 0;
	margin-bottom: 0.2em;
	padding: 0;
}
pre .tracer-subheader {
	color: #aa0000;
}
.tracer-paragraph {
	font-size: 1em;
	font-weight: normal;
	margin: 0;
}
.tracer-medheader {
	font-size: 1.4em;
	margin-top: 5px;
	margin-bottom: 5px;
	color: #aa0000;
}
.tracer-pre {
	font: monospace;
	font-size: 1em;
	font-weight: normal;
	margin: 0;
}
.tracer-code {
	font: monospace;
	font-size: 1em;
	font-weight: normal;
	margin: 0;
	white-space: pre-wrap;
}
.tracer-error {
	font: monospace;
	font-size: 1em;
	font-weight: normal;
	margin: 0;
}
.tracer-error-message {
	padding: 0.5em;
	font-size: 1.2em;
	margin: 0;
	text-align: right;
}
.tracer-error-stack {
	padding: 0.5em;
	font-size: 1em;
	border: 1px solid #bb0000;
	margin: 0;
	border-radius: 0.5em;
	background-color: #f5f5f5;
}
.tracer-error-stack-item {
	padding: 0.5em;
	font-size: 1em;
	margin: 0;
	white-space: pre-wrap;
}
.tracer-bold {
	font-weight: bold;
}

.tracer-settings {
	border: 1px solid #000000;
	border-radius: 0.5em;
	padding: 0.5em;
	margin: 0;
	border-style: inset;
	background-color: #f5f5f5;
}
`

var HTML_CLASS_H1 = "tracer-header"
var HTML_CLASS_H2 = "tracer-subheader"
var HTML_CLASS_H3 = "tracer-medheader"
var HTML_CLASS_P = "tracer-paragraph"
var HTML_CLASS_PRE = "tracer-pre"
var HTML_CLASS_CODE = "tracer-code"
var HTML_CLASS_ERROR = "tracer-error"
var HTML_CLASS_ERROR_MESSAGE = "tracer-error-message"
var HTML_CLASS_ERROR_STACK = "tracer-error-stack"
var HTML_CLASS_ERROR_STACK_ITEM = "tracer-error-stack-item"
var HTML_CLASS_SPACER = "tracer-spacer"
var HTML_CLASS_SPACER_XS = "tracer-spacer-xs"
var HTML_CLASS_BOLD = "tracer-bold"
var HTML_CLASS_SETTINGS = "tracer-settings"

type requestWriter interface {
	io.Writer
	WriteString(string) (int, error)
}

func RenderStdInfo(r requestWriter, s tracer.ErrorType) {
	r.WriteString("<div class=\"container\">")
	Header(r, fmt.Sprintf("Error: %s", s.Error()))
	HorizontalRuleXS(r)
	Paragraph(r, `To disable this page, please set `+
		makeBold("app.Application.DEBUG")+` to `+
		makeBold(false)+` or (`+
		makeBold("tracer.STACKLOGGER_UNSAFE")+` to `+
		makeBold(false)+` after running the app.)`)
	Paragraph(r, `Remember to always disable this page in production!`)
	r.WriteString("</div>")
}

func RenderStackTrace(s tracer.ErrorType, r *request.Request) {
	r.WriteString("<div class=\"container\">")
	MedHeader(r, "Stack Trace")
	SubHeader(r, `An error has occurred, please consider looking at the following stacktrace.`)
	HorizontalRuleXS(r)
	var trace = s.Trace()
	for i, item := range trace {
		var errb = Error(r)
		errb.Message(fmt.Sprintf("Error at line %s in file %s", makeBold(item.Line), makeBold(item.File)))
		var stackItem = errb.Stack()
		if tracer.STACKLOGGER_UNSAFE {
			stackItem.Item(makeBold(fmt.Sprintf("In function %s", item.FunctionName)))
			var amountOfLines int = 2
			var totalLen = len(trace) - 1
			if i >= totalLen {
				amountOfLines = amountOfLines * 4
			} else if i >= totalLen-1 {
				amountOfLines = amountOfLines * 3
			} else if i >= totalLen-2 {
				amountOfLines = amountOfLines * 2
			}
			var ff, err = item.Read(amountOfLines)
			if err != nil {
				stackItem.Item(fmt.Sprintf("In function %s", item.FunctionName))
				goto closeItem
			}
			if ff != nil {
				if len(ff.Data()) > 0 {
					stackItem.Item(ff.AsString("<span style=\"color:red;\"><i>", "</i></span>"))
				}
			}
		} else {
			stackItem.Item(fmt.Sprintf("In function %s", item.FunctionName))
		}
	closeItem:
		stackItem.End()
		errb.End()
		if i != len(s.Trace())-1 {
			HorizontalRule(r)
		}
	}
	r.WriteString("</div>")
}

func RenderRequest(b *request.Request) {
	b.WriteString("<div class=\"container\">")
	Header(b, "Request")
	HorizontalRuleXS(b)
	SubHeader(b, `The following information is about the request that caused the error.`)
	var headers = b.Request.Header
	b.WriteString("<pre class=\"tracer-settings\">")
	if len(headers) > 0 {
		MedHeader(b, makeBold("Headers:"))
		var newHeaders = make([][]string, 0, len(headers))
		for k, v := range headers {
			// if cookie header, skip
			if k == "Cookie" {
				continue
			}
			newHeaders = append(newHeaders, []string{k, strings.Join(v, ", ")})
		}
		sort.Slice(newHeaders, func(i, j int) bool {
			return newHeaders[i][0] < newHeaders[j][0]
		})
		for _, header := range newHeaders {
			if len(header) != 2 {
				continue
			}
			fmt.Fprintf(b, "%s: %s\n", makeBold(header[0]), header[1])
		}
		fmt.Fprintln(b)
	}
	var cookies = b.Request.Cookies()
	if len(cookies) > 0 {
		MedHeader(b, makeBold("Cookies:"))
		for _, cookie := range b.Request.Cookies() {
			fmt.Fprintf(b, "%s: %s\n", makeBold(cookie.Name), cookie.Value)
		}
	}
	var buf = new(bytes.Buffer)
	if b.Request.Body != nil {
		defer b.Request.Body.Close()
		var _, err = io.Copy(buf, b.Request.Body)
		if err != nil {
			fmt.Fprintf(b, "Error reading request body: %s", err.Error())
		}
		if buf.Len() > 0 {
			MedHeader(b, makeBold("Body:"))
			b.WriteString(buf.String())
		}
	}
	var form = b.Form()
	if len(form) > 0 {
		MedHeader(b, makeBold("Form:"))
		for k, v := range form {
			fmt.Fprintf(b, "%s: %s\n", makeBold(k), strings.Join(v, ", "))
		}
		fmt.Fprintln(b)
	}
	if b.Data != nil && (len(b.Data.Data) > 0 ||
		(b.Data.CSRFToken != nil && b.Data.CSRFToken.String() != "") ||
		b.User != nil) {

		if len(b.Data.Data) > 0 {
			MedHeader(b, makeBold("Data:"))
			for k, v := range b.Data.Data {
				fmt.Fprintf(b, "%s: %v\n", makeBold(k), v)
			}
		}
		var csrfToken = csrf.Token(b)
		if csrfToken != "" {
			fmt.Fprintf(b, "CSRFToken: %s\n", csrfToken)
		}
		if b.User != nil {
			fmt.Fprintf(b, "User: %s\n", makeBold(b.User))
		}
		fmt.Fprintln(b)
	}
	b.WriteString("</pre>")
	b.WriteString("</div>")
}

func RenderSettings(b requestWriter, settings *AppSettings) {
	b.WriteString("<div class=\"container\">")
	Header(b, "Settings")
	HorizontalRuleXS(b)
	SubHeader(b, `The following information is about the settings of the application.`)
	b.WriteString("<pre class=\"tracer-settings\">")

	MedHeader(b, makeBold("Application:"))
	fmt.Fprintf(b, "Host: %s\n", makeBold(settings.HOST))
	fmt.Fprintf(b, "Port: %s\n", makeBold(settings.PORT))
	fmt.Fprintf(b, "DEBUG: %s\n", makeBold(settings.DEBUG))

	if len(settings.DATABASES) > 0 {
		fmt.Fprintln(b)
		MedHeader(b, makeBold("Databases:"))
		for i, v := range settings.DATABASES {
			if v.KEY != "" {
				fmt.Fprintf(b, "Key: %s\n", makeBold(v.KEY))
			}
			if v.ENGINE != "" {
				fmt.Fprintf(b, "Engine: %s\n", makeBold(v.ENGINE))
			}
			if v.NAME != "" {
				fmt.Fprintf(b, "Name: %s\n", makeBold(v.NAME))
			}
			if v.SSL_MODE != "" {
				fmt.Fprintf(b, "SSL Mode: %s\n", makeBold(v.SSL_MODE))
			}
			if v.DB_USER != "" && v.DB_PASS != "" {
				fmt.Fprintf(b, "User: %s\n", makeBold(v.DB_USER))
				fmt.Fprint(b, "Pass: **********\n")
			}
			if i < len(settings.DATABASES)-1 {
				fmt.Fprintln(b)
			}
		}
	}
	if settings.ROUTES != "" {
		fmt.Fprintln(b)
		MedHeader(b, makeBold("Routes:"))
		fmt.Fprintf(b, "%s\n", makeBold(settings.ROUTES))
	}
	b.WriteString("</pre>")
	b.WriteString("</div>")
}

func StyleBlock(b requestWriter) {
	fmt.Fprintf(b, `<style type="text/css">%s</style>`, CSS)
}

func HorizontalRule(b requestWriter) {
	b.WriteString("<hr")
	b.WriteString(" class=\"")
	b.WriteString(HTML_CLASS_SPACER)
	b.WriteString("\"")
	b.WriteString(">\n")
}

func HorizontalRuleXS(b requestWriter) {
	b.WriteString("<hr")
	b.WriteString(" class=\"")
	b.WriteString(HTML_CLASS_SPACER_XS)
	b.WriteString("\"")
	b.WriteString(">\n")
}

func Header(b requestWriter, s string) {
	b.WriteString("<h1 class=\"")
	b.WriteString(HTML_CLASS_H1)
	b.WriteString("\">")
	b.WriteString(s)
	b.WriteString("</h1>")
}

func SubHeader(b requestWriter, s string) {
	b.WriteString("<h2 class=\"")
	b.WriteString(HTML_CLASS_H2)
	b.WriteString("\">")
	b.WriteString(s)
	b.WriteString("</h2>")
}

func MedHeader(b requestWriter, s string) {
	b.WriteString("<h3 class=\"")
	b.WriteString(HTML_CLASS_H3)
	b.WriteString("\">")
	b.WriteString(s)
	b.WriteString("</h3>")
}

func Paragraph(b requestWriter, s string) {
	b.WriteString("<p class=\"")
	b.WriteString(HTML_CLASS_P)
	b.WriteString("\">")
	b.WriteString(s)
	b.WriteString("</p>")
}

func Pre(b requestWriter, s string) {
	b.WriteString("<pre class=\"")
	b.WriteString(HTML_CLASS_PRE)
	b.WriteString("\">")
	b.WriteString(s)
	b.WriteString("</pre>\n")
}

func Code(b requestWriter, s string) {
	b.WriteString("<code class=\"")
	b.WriteString(HTML_CLASS_CODE)
	b.WriteString("\">")
	b.WriteString(s)
	b.WriteString("</code>\n")
}

func Error(b requestWriter) *errorBuilder {
	b.WriteString("<div class=\"")
	b.WriteString(HTML_CLASS_ERROR)
	b.WriteString("\">\n")
	return &errorBuilder{b}
}

type errorBuilder struct {
	b requestWriter
}

func (e *errorBuilder) End() requestWriter {
	e.b.WriteString("</div>\n")
	return e.b
}

func (e *errorBuilder) Message(s string) *errorBuilder {
	e.b.WriteString("<div class=\"")
	e.b.WriteString(HTML_CLASS_ERROR_MESSAGE)
	e.b.WriteString("\">")
	e.b.WriteString(s)
	e.b.WriteString("</div>\n")
	return e
}

func makeBold(s any) string {
	return fmt.Sprintf(`<span class="%s">%v</span>`, HTML_CLASS_BOLD, s)
}

type stackBuilder errorBuilder

func (e *stackBuilder) End() *stackBuilder {
	e.b.WriteString("</div>\n")
	return e
}

func (e *errorBuilder) Stack() *stackBuilder {
	e.b.WriteString("<div class=\"")
	e.b.WriteString(HTML_CLASS_ERROR_STACK)
	e.b.WriteString("\">\n")
	return &stackBuilder{e.b}
}

func (sb *stackBuilder) Item(s string) *stackBuilder {
	sb.b.WriteString("<div class=\"")
	sb.b.WriteString(HTML_CLASS_ERROR_STACK_ITEM)
	sb.b.WriteString("\">")
	sb.b.WriteString(s)
	sb.b.WriteString("</div>\n")
	return sb
}
