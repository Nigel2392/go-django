package editor_test

import (
	"strings"
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/editor"
)

func TestEditorJSBlockDataRender(t *testing.T) {
	data := &editor.EditorJSBlockData{
		Blocks: []editor.FeatureBlock{
			MockFeatureBlock{IDVal: "1", TypeVal: "header-type"},
			MockFeatureBlock{IDVal: "2", TypeVal: "paragraph-type"},
		},
	}

	rendered := data.Render()

	if !strings.Contains(string(rendered), "header-type") {
		t.Errorf("Render output did not contain 'header'")
	}
	if !strings.Contains(string(rendered), "paragraph-type") {
		t.Errorf("Render output did not contain 'paragraph'")
	}
}
