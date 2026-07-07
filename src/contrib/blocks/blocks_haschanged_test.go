package blocks_test

import (
	"testing"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/google/uuid"
)

func TestListBlock_HasChanged(t *testing.T) {
	b := blocks.NewListBlock(blocks.CharBlock())
	id1 := uuid.New()
	id2 := uuid.New()

	initial := &blocks.ListBlockValue{
		BlockObject: b,
		V: []*blocks.ListBlockData{
			{ID: id1, Order: 0, Data: "hello"},
			{ID: id2, Order: 1, Data: "world"},
		},
	}

	dataUnchanged := &blocks.ListBlockValue{
		BlockObject: b,
		V: []*blocks.ListBlockData{
			{ID: id1, Order: 0, Data: "hello"},
			{ID: id2, Order: 1, Data: "world"},
		},
	}
	if b.HasChanged(initial, dataUnchanged) {
		t.Errorf("expected HasChanged to be false for identical data")
	}

	dataChangedValue := &blocks.ListBlockValue{
		BlockObject: b,
		V: []*blocks.ListBlockData{
			{ID: id1, Order: 0, Data: "hello"},
			{ID: id2, Order: 1, Data: "WORLD"},
		},
	}
	if !b.HasChanged(initial, dataChangedValue) {
		t.Errorf("expected HasChanged to be true when a child's value changes")
	}

	dataChangedOrder := &blocks.ListBlockValue{
		BlockObject: b,
		V: []*blocks.ListBlockData{
			{ID: id2, Order: 0, Data: "world"},
			{ID: id1, Order: 1, Data: "hello"},
		},
	}
	if !b.HasChanged(initial, dataChangedOrder) {
		t.Errorf("expected HasChanged to be true when children are reordered")
	}
}

func TestStreamBlock_HasChanged(t *testing.T) {
	b := blocks.NewStreamBlock(
		blocks.WithBlockField[*blocks.StreamBlock]("title", blocks.CharBlock()),
	)
	id1 := uuid.New()
	id2 := uuid.New()

	initial := &blocks.StreamBlockValue{
		BlockObject: b,
		V: []*blocks.StreamBlockData{
			{ID: id1, Type: "title", Order: 0, Data: "hello"},
			{ID: id2, Type: "title", Order: 1, Data: "world"},
		},
	}

	dataUnchanged := &blocks.StreamBlockValue{
		BlockObject: b,
		V: []*blocks.StreamBlockData{
			{ID: id1, Type: "title", Order: 0, Data: "hello"},
			{ID: id2, Type: "title", Order: 1, Data: "world"},
		},
	}
	if b.HasChanged(initial, dataUnchanged) {
		t.Errorf("expected HasChanged to be false for identical data")
	}

	dataChangedValue := &blocks.StreamBlockValue{
		BlockObject: b,
		V: []*blocks.StreamBlockData{
			{ID: id1, Type: "title", Order: 0, Data: "hello"},
			{ID: id2, Type: "title", Order: 1, Data: "WORLD"},
		},
	}
	if !b.HasChanged(initial, dataChangedValue) {
		t.Errorf("expected HasChanged to be true when a child's value changes")
	}

	dataChangedOrder := &blocks.StreamBlockValue{
		BlockObject: b,
		V: []*blocks.StreamBlockData{
			{ID: id2, Type: "title", Order: 0, Data: "world"},
			{ID: id1, Type: "title", Order: 1, Data: "hello"},
		},
	}
	if !b.HasChanged(initial, dataChangedOrder) {
		t.Errorf("expected HasChanged to be true when children are reordered")
	}
}
