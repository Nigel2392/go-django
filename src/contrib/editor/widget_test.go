package editor_test

import (
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/editor"
)

func TestNewEditorJSWidget(t *testing.T) {
	widget := editor.NewEditorJSWidget("feature1", "feature2")

	if widget == nil {
		t.Error("Expected widget to be initialized, got nil")
		return
	}

	if widget.Type() != "editorjs" {
		t.Errorf("Expected widget type 'editorjs', but got %s", widget.Type())
	}

	if len(widget.Features) != 2 {
		t.Errorf("Expected 2 features, got %d", len(widget.Features))
	}
}

func TestEditorJSWidgetValueToForm(t *testing.T) {
	widget := editor.NewEditorJSWidget("feature1")

	blockData := &editor.EditorJSBlockData{
		Time:    1234567890,
		Version: "2.0",
		Blocks:  []editor.FeatureBlock{},
	}

	formValue := widget.ValueToForm(blockData)

	if formValue.(editor.EditorJSData).Version != "2.0" {
		t.Errorf("Expected form version '2.0', got %s", formValue.(editor.EditorJSData).Version)
	}
}

func TestEditorJSWidgetValueToGo(t *testing.T) {
	widget := editor.NewEditorJSWidget("feature1")

	jsData := editor.EditorJSData{
		Time:    1234567890,
		Version: "2.0",
	}

	goValue, err := widget.ValueToGo(jsData)
	if err != nil {
		t.Fatalf("Error converting ValueToGo: %v", err)
	}

	if goValue.(*editor.EditorJSBlockData).Version != "2.0" {
		t.Errorf("Expected Go value version '2.0', got %s", goValue.(*editor.EditorJSBlockData).Version)
	}
}
