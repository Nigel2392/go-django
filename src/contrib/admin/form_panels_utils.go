package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/src/core/ctx"
)

const PANEL_ID_PREFIX = "panel"

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

func BuildPanelID(panelIdx []int, extra ...string) string {
	var b = new(strings.Builder)
	var totalLen = len(PANEL_ID_PREFIX) + 1

	totalLen += len(panelIdx) * 2

	for i := 0; i < len(extra); i++ {
		totalLen += len(extra[i])
		totalLen++ // for the dash
	}

	b.Grow(totalLen)

	b.WriteString(PANEL_ID_PREFIX)

	for _, idx := range panelIdx {
		b.WriteString("-")
		b.WriteString(fmt.Sprintf("%d", idx))
	}

	// Append any extra strings
	for i := 0; i < len(extra); i++ {
		b.WriteString("-")
		b.WriteString(extra[i])
	}

	return b.String()
}

func PanelClass(className string, panel Panel) Panel {
	return panel.Class(className)
}
