package admin

import (
	"context"
	"iter"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms"
)

const PANEL_ID_PREFIX = "panel"

func addToPanelCount(key string, panelCount map[string]int) (nextIdx int) {
	if panelCount == nil {
		return 0
	}

	if _, ok := panelCount[key]; !ok {
		panelCount[key] = 0
	}
	var idx, ok = panelCount[key]
	if !ok {
		panelCount[key] = 0
		return 0
	}

	panelCount[key] = idx + 1
	return idx
}

type PanelContext struct {
	*ctx.HTTPRequestContext
	Panel      Panel
	BoundPanel BoundPanel
}

func NewPanelContext(r *http.Request, panel Panel, boundPanel BoundPanel) *PanelContext {
	return &PanelContext{
		HTTPRequestContext: ctx.RequestContext(r),
		Panel:              panel,
		BoundPanel:         boundPanel,
	}
}

func BuildPanelID(key string, panelCount map[string]int, extra ...string) string {
	var b = new(strings.Builder)
	var totalLen = len(PANEL_ID_PREFIX)
	var panelIdx = addToPanelCount(key, panelCount)

	if panelIdx > 0 {
		totalLen += int(math.Log10(float64(panelIdx))) + 1 // +1 for the hyphen
	}

	for i := 0; i < len(extra); i++ {
		totalLen += len(extra[i]) + 1 // +1 for the hyphen
	}

	b.Grow(totalLen)

	b.WriteString(PANEL_ID_PREFIX)

	// Append any extra strings
	for i := 0; i < len(extra); i++ {
		b.WriteString("-")
		b.WriteString(extra[i])
	}

	if panelIdx > 0 {
		b.WriteString("-")
		b.WriteString(strconv.Itoa(panelIdx))
	}

	return b.String()
}

func PanelClass(className string, panel Panel) Panel {
	return panel.Class(className)
}

func BindPanels(panels []Panel, r *http.Request, panelCount map[string]int, form forms.Form, ctx context.Context, boundFields map[string]forms.BoundField) iter.Seq2[int, BoundPanel] {
	return func(yield func(int, BoundPanel) bool) {
		var idx = 0
		for _, panel := range panels {
			var boundPanel = panel.Bind(r, panelCount, form, ctx, boundFields)
			if boundPanel == nil {
				continue
			}

			if !yield(idx, boundPanel) {
				break
			}

			idx++
		}
	}
}
