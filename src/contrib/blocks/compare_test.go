package blocks

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBlockFormFieldHasChanged_StreamBlockValue(t *testing.T) {
	field := BlockField(NewStreamBlock())
	id := uuid.New()

	oldValue := &StreamBlockValue{
		BlockObject: NewStreamBlock(),
		V: []*StreamBlockData{
			{ID: id, Type: "text", Data: "hello", Order: 0},
		},
	}
	newSameLogicalValue := &StreamBlockValue{
		BlockObject: NewStreamBlock(),
		V: []*StreamBlockData{
			{ID: id, Type: "text", Data: "hello", Order: 0},
		},
	}
	newChangedValue := &StreamBlockValue{
		BlockObject: NewStreamBlock(),
		V: []*StreamBlockData{
			{ID: id, Type: "text", Data: "changed", Order: 0},
		},
	}

	if field.HasChanged(oldValue, newSameLogicalValue) {
		t.Fatalf("expected logically equal stream values to be unchanged")
	}
	if !field.HasChanged(oldValue, newChangedValue) {
		t.Fatalf("expected changed stream values to be detected")
	}
	if field.HasChanged(nil, nil) {
		t.Fatalf("expected nil/nil to be unchanged")
	}
	if !field.HasChanged(nil, newChangedValue) {
		t.Fatalf("expected nil/non-nil to be changed")
	}
}

func TestHTMLDiffStream_ModifiedBlockByID(t *testing.T) {
	id := uuid.New()
	oldValue := &StreamBlockValue{
		V: []*StreamBlockData{
			{ID: id, Type: "paragraph", Data: "old text", Order: 0},
		},
	}
	newValue := &StreamBlockValue{
		V: []*StreamBlockData{
			{ID: id, Type: "paragraph", Data: "new text", Order: 0},
		},
	}

	diff := string(htmlDiffStream(context.Background(), oldValue, newValue))
	if !strings.Contains(diff, `class="diff-modified"`) {
		t.Fatalf("expected modified stream diff, got: %s", diff)
	}
}

func TestHTMLDiffList_AddedItem(t *testing.T) {
	oldValue := &ListBlockValue{
		V: []*ListBlockData{
			{ID: uuid.New(), Order: 0, Data: "a"},
		},
	}
	newValue := &ListBlockValue{
		V: []*ListBlockData{
			{ID: oldValue.V[0].ID, Order: 0, Data: "a"},
			{ID: uuid.New(), Order: 1, Data: "b"},
		},
	}

	diff := string(htmlDiffList(context.Background(), oldValue, newValue))
	if !strings.Contains(diff, `class="diff-added"`) {
		t.Fatalf("expected added list item diff, got: %s", diff)
	}
}

func TestHTMLDiffStruct_FieldByField(t *testing.T) {
	oldValue := &StructBlockValue{
		V: map[string]interface{}{
			"title": "Old Title",
			"body":  "Same",
		},
	}
	newValue := &StructBlockValue{
		V: map[string]interface{}{
			"title": "New Title",
			"body":  "Same",
		},
	}

	diff := string(htmlDiffStruct(context.Background(), oldValue, newValue))
	if !strings.Contains(diff, "<dt>title</dt>") {
		t.Fatalf("expected title key in struct diff, got: %s", diff)
	}
	if !strings.Contains(diff, `class="diff-modified"`) {
		t.Fatalf("expected modified struct field diff, got: %s", diff)
	}
}
