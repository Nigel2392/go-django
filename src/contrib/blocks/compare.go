package blocks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"reflect"
	"slices"
	"strings"
	"unicode"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/admin/compare"
	"github.com/Nigel2392/go-django/src/core/attrs"
	corectx "github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/google/uuid"
	xhtml "golang.org/x/net/html"
)

func init() {
	compare.RegisterComparisonType(
		reflect.TypeOf((*StreamBlockValue)(nil)),
		StreamBlockComparison,
	)
	compare.RegisterComparisonType(
		reflect.TypeOf((*ListBlockValue)(nil)),
		ListBlockComparison,
	)
	compare.RegisterComparisonType(
		reflect.TypeOf((*StructBlockValue)(nil)),
		StructBlockComparison,
	)
}

var _ compare.Comparison = (*blockComparison)(nil)

type blockComparison struct {
	ctx       context.Context
	LabelText any // string or func() string
	MetaField attrs.FieldDefinition
	Old, New  interface{}
}

func StreamBlockComparison(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (compare.Comparison, error) {
	defs := modelMeta.Definitions()
	field, ok := defs.Field(fieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T",
			fieldName, modelMeta.Model(),
		)
	}

	return &blockComparison{
		ctx:       ctx,
		LabelText: label,
		MetaField: field,
		Old:       attrs.Get[*StreamBlockValue](old, fieldName),
		New:       attrs.Get[*StreamBlockValue](new, fieldName),
	}, nil
}

func ListBlockComparison(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (compare.Comparison, error) {
	defs := modelMeta.Definitions()
	field, ok := defs.Field(fieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T",
			fieldName, modelMeta.Model(),
		)
	}

	return &blockComparison{
		ctx:       ctx,
		LabelText: label,
		MetaField: field,
		Old:       attrs.Get[*ListBlockValue](old, fieldName),
		New:       attrs.Get[*ListBlockValue](new, fieldName),
	}, nil
}

func StructBlockComparison(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (compare.Comparison, error) {
	defs := modelMeta.Definitions()
	field, ok := defs.Field(fieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T",
			fieldName, modelMeta.Model(),
		)
	}

	return &blockComparison{
		ctx:       ctx,
		LabelText: label,
		MetaField: field,
		Old:       attrs.Get[*StructBlockValue](old, fieldName),
		New:       attrs.Get[*StructBlockValue](new, fieldName),
	}, nil
}

func (fc *blockComparison) Label() string {
	if t, ok := trans.GetText(fc.ctx, fc.LabelText); ok {
		return t
	}
	return fc.MetaField.Label(fc.ctx)
}

func (fc *blockComparison) HasChanged() (bool, error) {
	formField := fc.MetaField.FormField()
	return formField.HasChanged(fc.Old, fc.New), nil
}

func (fc *blockComparison) HTMLDiff() (template.HTML, error) {
	oldStream, oldIsStream := fc.Old.(*StreamBlockValue)
	newStream, newIsStream := fc.New.(*StreamBlockValue)
	if oldIsStream || newIsStream {
		return htmlDiffStream(fc.ctx, oldStream, newStream), nil
	}

	oldList, oldIsList := fc.Old.(*ListBlockValue)
	newList, newIsList := fc.New.(*ListBlockValue)
	if oldIsList || newIsList {
		return htmlDiffList(fc.ctx, oldList, newList), nil
	}

	oldStruct, oldIsStruct := fc.Old.(*StructBlockValue)
	newStruct, newIsStruct := fc.New.(*StructBlockValue)
	if oldIsStruct || newIsStruct {
		return htmlDiffStruct(fc.ctx, oldStruct, newStruct), nil
	}

	oldText := html.EscapeString(renderValueText(fc.ctx, fc.Old))
	newText := html.EscapeString(renderValueText(fc.ctx, fc.New))
	if oldText == newText {
		return template.HTML(`<div class="diff-unchanged">` + oldText + `</div>`), nil
	}

	td := compare.DiffText(oldText, newText)
	// Inputs are HTML-escaped text, so rendering diff output as unsafe is intentional.
	td.Unsafe = true
	return template.HTML(`<div class="diff-modified">` + string(td.HTML()) + `</div>`), nil
}

func htmlDiffStream(ctx context.Context, oldValue, newValue *StreamBlockValue) template.HTML {
	var (
		oldItems []*StreamBlockData
		newItems []*StreamBlockData
	)
	if oldValue != nil {
		oldItems = oldValue.V
	}
	if newValue != nil {
		newItems = newValue.V
	}

	oldKeys := streamKeys(oldItems)
	newKeys := streamKeys(newItems)
	ops := lcsOps(oldKeys, newKeys)

	oldRendered := make([]string, len(oldItems))
	for i := range oldItems {
		oldRendered[i] = renderStreamItem(ctx, oldValue, oldItems[i], i)
	}
	newRendered := make([]string, len(newItems))
	for i := range newItems {
		newRendered[i] = renderStreamItem(ctx, newValue, newItems[i], i)
	}

	var out []string
	oi, nj := 0, 0
	for _, op := range ops {
		switch op {
		case "equal":
			if oldRendered[oi] == newRendered[nj] {
				out = append(out, `<div class="diff-unchanged">`+newRendered[nj]+`</div>`)
			} else {
				td := compare.DiffText(oldRendered[oi], newRendered[nj])
				// Inputs are HTML-escaped text, so rendering diff output as unsafe is intentional.
				td.Unsafe = true
				out = append(out, `<div class="diff-modified">`+string(td.HTML())+`</div>`)
			}
			oi++
			nj++
		case "delete":
			out = append(out, `<div class="diff-removed">`+oldRendered[oi]+`</div>`)
			oi++
		case "insert":
			out = append(out, `<div class="diff-added">`+newRendered[nj]+`</div>`)
			nj++
		}
	}

	return template.HTML(strings.Join(out, ""))
}

func htmlDiffList(ctx context.Context, oldValue, newValue *ListBlockValue) template.HTML {
	var (
		oldItems []*ListBlockData
		newItems []*ListBlockData
	)
	if oldValue != nil {
		oldItems = oldValue.V
	}
	if newValue != nil {
		newItems = newValue.V
	}

	oldKeys := listKeys(oldItems)
	newKeys := listKeys(newItems)
	ops := lcsOps(oldKeys, newKeys)

	oldRendered := make([]string, len(oldItems))
	for i := range oldItems {
		oldRendered[i] = renderListItem(ctx, oldValue, oldItems[i], i)
	}
	newRendered := make([]string, len(newItems))
	for i := range newItems {
		newRendered[i] = renderListItem(ctx, newValue, newItems[i], i)
	}

	var out []string
	oi, nj := 0, 0
	for _, op := range ops {
		switch op {
		case "equal":
			if oldRendered[oi] == newRendered[nj] {
				out = append(out, `<div class="diff-unchanged">`+newRendered[nj]+`</div>`)
			} else {
				td := compare.DiffText(oldRendered[oi], newRendered[nj])
				// Inputs are HTML-escaped text, so rendering diff output as unsafe is intentional.
				td.Unsafe = true
				out = append(out, `<div class="diff-modified">`+string(td.HTML())+`</div>`)
			}
			oi++
			nj++
		case "delete":
			out = append(out, `<div class="diff-removed">`+oldRendered[oi]+`</div>`)
			oi++
		case "insert":
			out = append(out, `<div class="diff-added">`+newRendered[nj]+`</div>`)
			nj++
		}
	}

	return template.HTML(strings.Join(out, ""))
}

func htmlDiffStruct(ctx context.Context, oldValue, newValue *StructBlockValue) template.HTML {
	oldData := map[string]interface{}{}
	newData := map[string]interface{}{}
	if oldValue != nil {
		oldData = oldValue.V
	}
	if newValue != nil {
		newData = newValue.V
	}

	keys := structKeys(oldValue, newValue, oldData, newData)
	out := make([]string, 0, len(keys))
	for _, key := range keys {
		oldRaw, oldOK := oldData[key]
		newRaw, newOK := newData[key]

		oldText := html.EscapeString(renderValueText(ctx, oldRaw))
		newText := html.EscapeString(renderValueText(ctx, newRaw))

		var body string
		switch {
		case !oldOK && newOK:
			body = `<div class="diff-added">` + newText + `</div>`
		case oldOK && !newOK:
			body = `<div class="diff-removed">` + oldText + `</div>`
		case oldText == newText:
			body = `<div class="diff-unchanged">` + newText + `</div>`
		default:
			td := compare.DiffText(oldText, newText)
			// Inputs are HTML-escaped text, so rendering diff output as unsafe is intentional.
			td.Unsafe = true
			body = `<div class="diff-modified">` + string(td.HTML()) + `</div>`
		}

		out = append(out, `<dt>`+html.EscapeString(key)+`</dt><dd>`+body+`</dd>`)
	}

	return template.HTML("<dl>" + strings.Join(out, "") + "</dl>")
}

func streamKeys(items []*StreamBlockData) []string {
	keys := make([]string, len(items))
	for i, item := range items {
		if item == nil {
			keys[i] = fmt.Sprintf("nil#%d", i)
			continue
		}
		if item.ID != uuid.Nil {
			keys[i] = item.ID.String()
			continue
		}
		keys[i] = fmt.Sprintf("%s#%d", item.Type, i)
	}
	return keys
}

func listKeys(items []*ListBlockData) []string {
	keys := make([]string, len(items))
	for i, item := range items {
		if item == nil {
			keys[i] = fmt.Sprintf("nil#%d", i)
			continue
		}
		if item.ID != uuid.Nil {
			keys[i] = item.ID.String()
			continue
		}
		keys[i] = fmt.Sprintf("item#%d", i)
	}
	return keys
}

func structKeys(oldValue, newValue *StructBlockValue, oldData, newData map[string]interface{}) []string {
	keys := make([]string, 0, max(len(oldData), len(newData)))
	added := make(map[string]struct{}, len(oldData)+len(newData))
	appendKey := func(key string) {
		if _, ok := added[key]; ok {
			return
		}
		added[key] = struct{}{}
		keys = append(keys, key)
	}

	prefixLen := 0
	for _, value := range []*StructBlockValue{oldValue, newValue} {
		if value == nil || value.BlockObject == nil {
			continue
		}
		if block, ok := value.BlockObject.(*StructBlock); ok && block.Fields != nil {
			for head := block.Fields.Front(); head != nil; head = head.Next() {
				appendKey(head.Key)
			}
			prefixLen = len(keys)
		}
	}

	for key := range oldData {
		appendKey(key)
	}
	for key := range newData {
		appendKey(key)
	}

	if len(keys[prefixLen:]) > 1 {
		slices.Sort(keys[prefixLen:])
	}
	return keys
}

func renderStreamItem(ctx context.Context, parent *StreamBlockValue, item *StreamBlockData, index int) string {
	if item == nil {
		return "&lt;empty&gt;"
	}

	var text string
	if parent != nil {
		if streamBlock, ok := parent.BlockObject.(*StreamBlock); ok && streamBlock.Children != nil {
			if child, ok := streamBlock.Children.Get(item.Type); ok {
				if rendered, ok := renderWithBlock(ctx, child, item.Data); ok {
					text = rendered
				}
			}
		}
	}
	if text == "" {
		text = renderValueText(ctx, item.Data)
	}
	if text == "" {
		text = fmt.Sprintf("<%s:%d>", item.Type, index)
	}
	return html.EscapeString(text)
}

func renderListItem(ctx context.Context, parent *ListBlockValue, item *ListBlockData, index int) string {
	if item == nil {
		return "&lt;empty&gt;"
	}

	var text string
	if parent != nil {
		if listBlock, ok := parent.BlockObject.(*ListBlock); ok && listBlock.Child != nil {
			if rendered, ok := renderWithBlock(ctx, listBlock.Child, item.Data); ok {
				text = rendered
			}
		}
	}
	if text == "" {
		text = renderValueText(ctx, item.Data)
	}
	if text == "" {
		text = fmt.Sprintf("<item:%d>", index)
	}
	return html.EscapeString(text)
}

func renderWithBlock(ctx context.Context, block Block, value interface{}) (string, bool) {
	var buf bytes.Buffer
	if err := block.Render(ctx, &buf, value, corectx.NewContext(nil)); err != nil {
		return "", false
	}
	return normalizeText(extractText(buf.String())), true
}

func renderValueText(ctx context.Context, value interface{}) string {
	if value == nil {
		return ""
	}

	if bound, ok := value.(BoundBlockValue); ok {
		if block := bound.Block(); block != nil {
			if rendered, ok := renderWithBlock(ctx, block, value); ok {
				return rendered
			}
		}
		return renderValueText(ctx, bound.Data())
	}

	switch v := value.(type) {
	case string:
		return normalizeText(v)
	case []byte:
		return normalizeText(string(v))
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return normalizeText(fmt.Sprintf("%v", value))
	}
	return normalizeText(string(raw))
}

func lcsOps(a, b []string) []string {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}

	ops := make([]string, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			ops = append(ops, "equal")
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			ops = append(ops, "delete")
			i++
		} else {
			ops = append(ops, "insert")
			j++
		}
	}
	for i < n {
		ops = append(ops, "delete")
		i++
	}
	for j < m {
		ops = append(ops, "insert")
		j++
	}
	return ops
}

func extractText(h string) string {
	var b strings.Builder
	t := xhtml.NewTokenizer(strings.NewReader(h))
	for {
		switch t.Next() {
		case xhtml.ErrorToken:
			return normalizeText(b.String())
		case xhtml.TextToken:
			b.WriteString(string(t.Text()))
		}
	}
}

func normalizeText(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
			continue
		}
		b.WriteRune(r)
		space = false
	}
	return strings.TrimSpace(b.String())
}
