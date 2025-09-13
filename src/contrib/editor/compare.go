package editor

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"unicode"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/contrib/admin/compare"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"golang.org/x/net/html"
)

func init() {
	compare.RegisterComparisonType(reflect.TypeOf((*EditorJSBlockData)(nil)), EditorComparison)
}

var (
	_ compare.Comparison = (*editorComparison)(nil)
)

type editorComparison struct {
	ctx       context.Context
	LabelText any // string or func() string
	MetaField attrs.FieldDefinition
	Old, New  *EditorJSBlockData
}

func EditorComparison(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (compare.Comparison, error) {
	var defs = modelMeta.Definitions()
	var field, ok = defs.Field(fieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T",
			fieldName, modelMeta.Model(),
		)
	}

	var oldValue = attrs.Get[*EditorJSBlockData](old, fieldName)
	var newValue = attrs.Get[*EditorJSBlockData](new, fieldName)
	var fc = &editorComparison{
		ctx:       ctx,
		LabelText: label,
		MetaField: field,
		Old:       oldValue,
		New:       newValue,
	}

	return fc, nil
}

func (fc *editorComparison) Label() string {
	var t, ok = trans.GetText(fc.ctx, fc.LabelText)
	if !ok {
		return fc.MetaField.Label(fc.ctx)
	}
	return t
}

func (fc *editorComparison) HasChanged() (bool, error) {
	var formField = fc.MetaField.FormField()
	return formField.HasChanged(fc.Old, fc.New), nil
}

// HTMLDiff builds an HTML representation of differences between the old and new
// EditorJSBlockData. It diffs at the block level (using IDs if available) and
// for blocks that share the same ID/Type but changed content, it renders an
// inner text diff.
func (fc *editorComparison) HTMLDiff() (template.HTML, error) {
	var oldBlocks, newBlocks []FeatureBlock
	if fc.Old != nil {
		oldBlocks = fc.Old.Blocks
	}
	if fc.New != nil {
		newBlocks = fc.New.Blocks
	}

	// Pre-render all blocks to HTML once (used both for equality checks and output).
	oldHTML := renderAll(fc.ctx, oldBlocks)
	newHTML := renderAll(fc.ctx, newBlocks)

	// Compute LCS on keys (prefer ID; fallback to Type+index).
	oldKeys := blockKeys(oldBlocks)
	newKeys := blockKeys(newBlocks)
	ops := lcsOps(oldKeys, newKeys)

	var out []string
	oi, nj := 0, 0
	for _, op := range ops {
		switch op {
		case "equal":
			// Same key; check if content changed
			oh := oldHTML[oi]
			nh := newHTML[nj]
			if oh == nh {
				out = append(out, nh) // unchanged
			} else {
				// Modified: show an inner textual diff of the rendered text
				ot := extractText(oh)
				nt := extractText(nh)
				td := compare.DiffText(ot, nt)
				inner := td.HTML()
				// Wrap with a container so it's visually grouped as a modified block
				out = append(out, `<div class="diff-modified">`+string(inner)+`</div>`)
			}
			oi++
			nj++
		case "delete":
			out = append(out, `<div class="diff-removed">`+oldHTML[oi]+`</div>`)
			oi++
		case "insert":
			out = append(out, `<div class="diff-added">`+newHTML[nj]+`</div>`)
			nj++
		}
	}

	return template.HTML(strings.Join(out, "")), nil
}

// --- helpers ---

// renderAll renders each FeatureBlock to HTML (best-effort; empty string on error).
func renderAll(ctx context.Context, blocks []FeatureBlock) []string {
	out := make([]string, 0, len(blocks))
	for i, b := range blocks {
		var buf bytes.Buffer
		_ = b.Render(ctx, &buf) // best-effort; ignore error to keep diff robust
		var s = buf.String()
		s = extractText(s) // strip tags for cleaner diffs
		s = squeezeSpaces(s)
		if s == "" {
			s = fmt.Sprintf("<empty block %d: %v>", i+1, b.Type())
		}
		out = append(out, s)
	}
	return out
}

// blockKeys builds the identity key used for LCS: prefer stable ID(); fallback to Type()+index.
func blockKeys(blocks []FeatureBlock) []string {
	keys := make([]string, len(blocks))
	for i, b := range blocks {
		id := b.ID()
		if id == "" {
			id = b.Type() + "#" + itoa(i)
		}
		keys[i] = id
	}
	return keys
}

// lcsOps returns a sequence of ops ("equal" | "delete" | "insert")
// describing how to transform old -> new, using a standard LCS DP.
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

	i, j := 0, 0
	ops := make([]string, 0, n+m)
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

// extractText strips HTML tags to get a plain-text representation suitable for inner diffs.
func extractText(h string) string {
	var b strings.Builder
	t := html.NewTokenizer(strings.NewReader(h))
	for {
		tt := t.Next()
		switch tt {
		case html.ErrorToken:
			return squeezeSpaces(b.String())
		case html.TextToken:
			b.WriteString(string(t.Text()))
		}
	}
}

// squeezeSpaces normalizes whitespace for cleaner diffs.
func squeezeSpaces(s string) string {
	var b strings.Builder
	space := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !space {
				b.WriteByte(' ')
				space = true
			}
		} else {
			b.WriteRune(r)
			space = false
		}
	}
	return strings.TrimSpace(b.String())
}

// tiny int->string without fmt (keeps imports tight)
func itoa(i int) string {
	// enough for typical sizes; fallback to fmt if you prefer
	return strconvItoa(i)
}

// minimal base-10 int to string
func strconvItoa(x int) string {
	// handle zero
	if x == 0 {
		return "0"
	}
	neg := x < 0
	if neg {
		x = -x
	}
	var buf [20]byte
	i := len(buf)
	for x > 0 {
		i--
		buf[i] = byte('0' + x%10)
		x /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
