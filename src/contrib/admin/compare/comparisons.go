package compare

import (
	"context"
	"fmt"
	"html/template"
	"reflect"
	"unicode"

	"github.com/Nigel2392/go-django/queries/src/drivers/errors"
	"github.com/Nigel2392/go-django/src/core/attrs"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/Nigel2392/go-django/src/utils/text"
)

func init() {
	RegisterDefaultComparison(FieldComparison)
	RegisterComparisonKind(reflect.String, FieldComparison)
	RegisterComparisonKind(reflect.Int, FieldComparison)
	RegisterComparisonKind(reflect.Float64, FieldComparison)
	RegisterComparisonKind(reflect.Bool, FieldComparison)
	RegisterComparisonKind(reflect.Slice, FieldComparison)
	RegisterComparisonKind(reflect.Map, FieldComparison)
	RegisterComparisonKind(reflect.Struct, FieldComparison)
	RegisterComparisonKind(reflect.Ptr, FieldComparison)
	RegisterComparisonKind(reflect.Interface, FieldComparison)
	RegisterComparisonKind(reflect.Array, FieldComparison)
}

var (
	_ Comparison = (*fieldComparison)(nil)
)

type fieldComparison struct {
	ctx       context.Context
	LabelText any // string or func() string
	MetaField attrs.FieldDefinition
	Old, New  interface{}
}

func FieldComparison(ctx context.Context, label func(context.Context) string, fieldName string, modelMeta attrs.ModelMeta, old, new attrs.Definer) (Comparison, error) {
	var defs = modelMeta.Definitions()
	var field, ok = defs.Field(fieldName)
	if !ok {
		return nil, errors.FieldNotFound.Wrapf(
			"field %q not found in model %T",
			fieldName, modelMeta.Model(),
		)
	}

	var oldValue = attrs.Get[any](old, fieldName)
	var newValue = attrs.Get[any](new, fieldName)
	var fc = &fieldComparison{
		ctx:       ctx,
		LabelText: label,
		MetaField: field,
		Old:       oldValue,
		New:       newValue,
	}

	return fc, nil
}

func (fc *fieldComparison) Label() string {
	var t, ok = trans.GetText(fc.ctx, fc.LabelText)
	if !ok {
		return fc.MetaField.Label(fc.ctx)
	}
	return t
}

func (fc *fieldComparison) HasChanged() (bool, error) {
	var formField = fc.MetaField.FormField()
	return formField.HasChanged(fc.Old, fc.New), nil
}

func (fc *fieldComparison) HTMLDiff() (template.HTML, error) {
	var changed, _ = fc.HasChanged()
	if !changed {
		var diff = TextDiff{
			Changes: []Differential{
				{Type: DIFF_EQUALS, Value: fc.Old},
			},
		}
		return diff.HTML(), nil
	}

	var diff = TextDiff{
		Changes: []Differential{
			{Type: DIFF_REMOVED, Value: fc.Old},
			{Type: DIFF_ADDED, Value: fc.New},
		},
	}
	return diff.HTML(), nil
}

type multipleComparison struct {
	ctx         context.Context
	Comparisons []Comparison
}

func MultipleComparison(ctx context.Context, comparisons ...Comparison) Comparison {
	return &multipleComparison{
		ctx:         ctx,
		Comparisons: comparisons,
	}
}

func (mc *multipleComparison) Unwrap() []Comparison {
	return mc.Comparisons
}

func (mc *multipleComparison) Label() string {
	return ""
}

func (mc *multipleComparison) HasChanged() (bool, error) {
	for _, comp := range mc.Comparisons {
		changed, err := comp.HasChanged()
		if err != nil {
			return false, err
		}
		if changed {
			return true, nil
		}
	}
	return false, nil
}

func (mc *multipleComparison) HTMLDiff() (template.HTML, error) {
	var argList = make([][]any, 0, len(mc.Comparisons))
	for _, comp := range mc.Comparisons {

		html, err := comp.HTMLDiff()
		if err != nil {
			return "", err
		}

		argList = append(argList, []any{comp.Label(), html})
	}

	var inner = text.JoinFormat(
		"\n", "<dt>%s</dt><dd>%s</dd>", argList...,
	)

	return template.HTML(fmt.Sprintf("<dl>%s</dl>", inner)), nil
}

// DiffText performs a token-based diff between a and b, returning a TextDiff
// whose Changes contain merged "equals", "added", and "removed" runs.
func DiffText(a, b string) *TextDiff {
	aTok := tokenize(a)
	bTok := tokenize(b)

	changes := diffTokens(aTok, bTok)
	merged := mergeAdjacent(changes)

	return &TextDiff{
		Changes:   merged,
		Separator: "", // tokens are already concatenated per run
		Tagname:   "span",
	}
}

// tokenize splits text into tokens by grouping consecutive alphanumeric runes
// and emitting every non-alphanumeric rune (punctuation, whitespace, etc.) as
// a separate token. This mirrors the Python tokeniser using str.isalnum().
func tokenize(s string) []string {
	tokens := make([]string, 0, len(s))
	var cur []rune

	flush := func() {
		if len(cur) > 0 {
			tokens = append(tokens, string(cur))
			cur = cur[:0]
		}
	}

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			cur = append(cur, r)
		} else {
			flush()
			tokens = append(tokens, string(r))
		}
	}
	flush()
	return tokens
}

// diffTokens computes a longest-common-subsequence-based diff over token slices,
// yielding a flat list of per-token changes. This produces only equal/add/remove;
// "replace" regions appear as a run of removals followed by a run of additions,
// just like the Python opcodes expansion.
func diffTokens(a, b []string) []Differential {
	n, m := len(a), len(b)

	// dp[i][j] = LCS length of a[i:] and b[j:]
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}

	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else {
				if dp[i+1][j] >= dp[i][j+1] {
					dp[i][j] = dp[i+1][j]
				} else {
					dp[i][j] = dp[i][j+1]
				}
			}
		}
	}

	// Backtrack to build the change list
	changes := make([]Differential, 0, n+m)
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			changes = append(changes, Differential{Type: DIFF_EQUALS, Value: a[i]})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			changes = append(changes, Differential{Type: DIFF_REMOVED, Value: a[i]})
			i++
		} else {
			changes = append(changes, Differential{Type: DIFF_ADDED, Value: b[j]})
			j++
		}
	}
	for i < n {
		changes = append(changes, Differential{Type: DIFF_REMOVED, Value: a[i]})
		i++
	}
	for j < m {
		changes = append(changes, Differential{Type: DIFF_ADDED, Value: b[j]})
		j++
	}

	return changes
}

// mergeAdjacent collapses consecutive changes of the same type into a single
// Differential whose Value is the concatenated string.
func mergeAdjacent(changes []Differential) []Differential {
	if len(changes) == 0 {
		return nil
	}
	out := make([]Differential, 0, len(changes))
	curType := changes[0].Type
	curVal := ""

	toString := func(v any) string {
		if s, ok := v.(string); ok {
			return s
		}
		// fall back
		return fmtSprint(v)
	}

	for _, ch := range changes {
		if ch.Type != curType {
			out = append(out, Differential{Type: curType, Value: curVal})
			curType = ch.Type
			curVal = ""
		}
		curVal += toString(ch.Value)
	}
	out = append(out, Differential{Type: curType, Value: curVal})
	return out
}

// small printf-free helper to avoid importing fmt just for Sprintf here
func fmtSprint(v any) string {
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	default:
		// minimal fallback; feel free to replace with fmt.Sprint if you already import fmt elsewhere
		return "<non-string>"
	}
}
