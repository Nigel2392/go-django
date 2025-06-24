package models_test

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"testing"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/queries/src/fields"
	"github.com/Nigel2392/go-django/queries/src/models"
	"github.com/Nigel2392/go-django/queries/src/quest"
	"github.com/Nigel2392/go-django/src/core/attrs"
)

type ImageModel struct {
	models.Model
	ID         int64
	ImageTitle string
	ImageURL   string
}

func (m *ImageModel) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
		attrs.Unbound("ImageTitle"),
		attrs.Unbound("ImageURL"),
	)
}

type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	return json.Marshal(map[string]any(m))
}

func (m *JSONMap) Scan(value any) error {
	if value == nil {
		*m = nil
		return nil
	}

	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return fmt.Errorf("unsupported type for JSONMap: %T", value)
	}

	return json.Unmarshal(data, m)
}

type StatefulModel struct {
	models.Model
	ID        int64
	FirstName string
	LastName  string
	Age       int
	BinData   []byte
	MapData   JSONMap
	Image     *ImageModel
}

func (m *StatefulModel) FieldDefs() attrs.Definitions {
	return m.Model.Define(m,
		attrs.Unbound("ID", &attrs.FieldConfig{Primary: true}),
		attrs.Unbound("FirstName"),
		attrs.Unbound("LastName"),
		attrs.Unbound("Age"),
		attrs.Unbound("BinData"),
		attrs.Unbound("MapData"),
		fields.ForeignKey[*ImageModel]("Image", "image_id"),
	)
}

func TestState(t *testing.T) {
	var tables = quest.Table(t,
		&ImageModel{},
		&StatefulModel{},
	)

	tables.Create()
	defer tables.Drop()

	var model = models.Setup(&StatefulModel{
		FirstName: "John",
		LastName:  "Doe",
		Age:       30,
		BinData:   []byte{1, 2, 3},
		MapData: JSONMap{
			"key1": "value1",
			"key2": 42,
		},
		Image: &ImageModel{
			ImageTitle: "Sample Image",
			ImageURL:   "http://example.com/image.jpg",
		},
	})

	t.Run("StateNilBeforeFieldDefsCalled", func(t *testing.T) {
		var state = model.State()
		if state != nil {
			t.Errorf("Expected state to be nil, got: %v", state)
		}
	})

	t.Run("StateChangedAfterSetFirstName", func(t *testing.T) {
		var defs = model.FieldDefs()
		var state = model.State()
		if state == nil {
			t.Error("Expected state to be non-nil after change")
		}

		t.Run("InitialStateUnchanged", func(t *testing.T) {
			if state.Changed(false) {
				t.Error("Expected initial state to be unchanged")
			}
			if state.Changed(true) {
				t.Error("Expected initial state to be unchanged with checkState")
			}
		})

		defs.Set("FirstName", "Jane")

		t.Run("StateChanged", func(t *testing.T) {
			if !state.Changed(false) {
				t.Error("Expected state to be changed after modifying FirstName")
			}
		})

		t.Run("FirstNameChanged", func(t *testing.T) {
			if !state.HasChanged("FirstName") {
				t.Error("Expected FirstName to be marked as changed")
			}
		})

		t.Run("StateUnchangedAfterReset", func(t *testing.T) {
			var state = model.State()
			if state == nil {
				t.Error("Expected state to be non-nil after change")
			}

			if !state.Changed(false) {
				t.Error("Expected state to be changed after test \"StateChangedAfterSetFirstName\"")
			}

			state.Reset()

			if state.Changed(false) {
				t.Error("Expected state to be unchanged after reset")
			}

			if state.Changed(true) {
				t.Error("Expected state to be unchanged after reset with checkState")
			}
		})

		model.State().Reset()
	})

	t.Run("StateChangedAfterSetBinData", func(t *testing.T) {
		var defs = model.FieldDefs()
		var state = model.State()
		if state == nil {
			t.Error("Expected state to be non-nil after change")
		}

		t.Run("InitialStateUnchanged", func(t *testing.T) {
			if state.Changed(false) {
				t.Error("Expected initial state to be unchanged")
			}
			if state.Changed(true) {
				t.Error("Expected initial state to be unchanged with checkState")
			}
		})

		defs.Set("BinData", []byte{4, 5, 6})

		t.Run("StateChanged", func(t *testing.T) {
			if !state.Changed(false) {
				t.Error("Expected state to be changed after modifying BinData")
			}

			if !state.HasChanged("BinData") {
				t.Error("Expected BinData to be marked as changed")
			}
		})

		model.State().Reset()
	})

	t.Run("StateChangedAfterSetMapData", func(t *testing.T) {
		var defs = model.FieldDefs()
		var state = model.State()
		if state == nil {
			t.Error("Expected state to be non-nil after change")
		}
		t.Run("InitialStateUnchanged", func(t *testing.T) {
			if state.Changed(false) {
				t.Error("Expected initial state to be unchanged")
			}
			if state.Changed(true) {
				t.Error("Expected initial state to be unchanged with checkState")
			}
		})

		defs.Set("MapData", JSONMap{
			"key1": "value2",
			"key2": 84,
		})

		t.Run("StateChanged", func(t *testing.T) {
			if !state.Changed(false) {
				t.Error("Expected state to be changed after modifying MapData")
			}
			if !state.HasChanged("MapData") {
				t.Error("Expected MapData to be marked as changed")
			}
		})

		model.State().Reset()
	})

	t.Run("StateChangedAfterChangeImage", func(t *testing.T) {
		var defs = model.FieldDefs()
		var state = model.State()
		if state == nil {
			t.Error("Expected state to be non-nil after change")
		}

		t.Run("InitialStateUnchanged", func(t *testing.T) {
			if state.Changed(false) {
				t.Error("Expected initial state to be unchanged")
			}
			if state.Changed(true) {
				t.Error("Expected initial state to be unchanged with checkState")
			}
		})

		defs.Set("Image", &ImageModel{
			ImageTitle: "Updated Image",
			ImageURL:   "http://example.com/updated_image.jpg",
		})

		t.Run("StateChanged", func(t *testing.T) {
			if !state.Changed(false) {
				t.Error("Expected state to be changed without checkState after modifying Image")
			}
			if !state.HasChanged("Image") {
				t.Error("Expected Image to be marked as changed")
			}
		})

		model.State().Reset()
	})

	t.Run("StateAfterSave", func(t *testing.T) {
		var err = model.Save(context.Background())
		if err != nil {
			t.Fatalf("Failed to save model: %v", err)
		}

		t.Run("StatefulModelAttrsAfterSave", func(t *testing.T) {
			if model.ID == 0 {
				t.Error("Expected ID to be set after save, got 0")
			}
			if model.FirstName != "Jane" {
				t.Errorf("Expected FirstName to be 'Jane', got: %s", model.FirstName)
			}
			if model.LastName != "Doe" {
				t.Errorf("Expected LastName to be 'Doe', got: %s", model.LastName)
			}
			if model.Age != 30 {
				t.Errorf("Expected Age to be 30, got: %d", model.Age)
			}
			if len(model.BinData) != 3 || model.BinData[0] != 4 || model.BinData[1] != 5 || model.BinData[2] != 6 {
				t.Errorf("Expected BinData to be [1, 2, 3], got: %v", model.BinData)
			}
			if model.MapData == nil {
				t.Errorf("Expected MapData to be non-nil after save")
			}
			if len(model.MapData) != 2 {
				t.Errorf("Expected MapData to have 2 keys, got: %d", len(model.MapData))
			}
			if model.MapData["key1"] != "value2" {
				t.Errorf("Expected MapData['key1'] to be 'value2', got: %v", model.MapData["key1"])
			}
			// convert to float64 for comparison after JSON unmarshalling
			if model.MapData["key2"] != float64(84) {
				t.Errorf("Expected MapData['key2'] to be 84, got: %v (%T)", model.MapData["key2"], model.MapData["key2"])
			}
		})

		t.Run("ImageModelAttrsAfterSave", func(t *testing.T) {
			if model.Image == nil {
				t.Fatal("Expected Image to be non-nil after save")
			}

			if model.Image.ID == 0 {
				t.Error("Expected Image ID to be set after save, got 0")
			}

			if model.Image.ImageTitle != "Updated Image" {
				t.Errorf("Expected ImageTitle to be 'Updated Image', got: %s", model.Image.ImageTitle)
			}

			if model.Image.ImageURL != "http://example.com/updated_image.jpg" {
				t.Errorf("Expected ImageURL to be 'http://example.com/updated_image.jpg', got: %s", model.Image.ImageURL)
			}
		})

		t.Logf("Model saved successfully: %+v", model)

		var state = model.State()
		if state == nil {
			t.Error("Expected state to be non-nil after save")
		}

		t.Run("StateUnchangedAfterSave", func(t *testing.T) {
			if state.Changed(false) {
				t.Error("Expected state to be unchanged after save")
			}
			if state.Changed(true) {
				t.Error("Expected state to be unchanged after save with checkState")
			}
		})
	})

	t.Run("RelatedSave", func(t *testing.T) {
		t.Run("CheckImageFromDB", func(t *testing.T) {
			var imgRow, err = queries.
				GetQuerySet(&ImageModel{}).
				Filter("ID", model.Image.ID).
				Get()
			if err != nil {
				t.Fatalf("Failed to get ImageModel from DB: %v", err)
			}

			if imgRow.Object.ID != model.Image.ID {
				t.Errorf("Expected ImageModel ID to be %d, got: %d", model.Image.ID, imgRow.Object.ID)
			}

			if imgRow.Object.ImageTitle != model.Image.ImageTitle {
				t.Errorf("Expected ImageModel ImageTitle to be '%s', got: '%s'", model.Image.ImageTitle, imgRow.Object.ImageTitle)
			}

			if imgRow.Object.ImageURL != model.Image.ImageURL {
				t.Errorf("Expected ImageModel ImageURL to be '%s', got: '%s'", model.Image.ImageURL, imgRow.Object.ImageURL)
			}
		})

		model.Image.ImageTitle = "New Image Title"
		model.Image.ImageURL = "http://example.com/new_image.jpg"

		if err := model.Image.Save(context.Background()); err != nil {
			t.Fatalf("Failed to save model with updated Image: %v", err)
		}

		t.Run("CheckImageFromDB", func(t *testing.T) {
			var imgRow, err = queries.
				GetQuerySet(&ImageModel{}).
				Filter("ID", model.Image.ID).
				Get()
			if err != nil {
				t.Fatalf("Failed to get ImageModel from DB: %v", err)
			}

			if imgRow.Object.ID != model.Image.ID {
				t.Errorf("Expected ImageModel ID to be %d, got: %d", model.Image.ID, imgRow.Object.ID)
			}

			if imgRow.Object.ImageTitle != "New Image Title" {
				t.Errorf("Expected ImageModel ImageTitle to be '%s', got: '%s'", model.Image.ImageTitle, imgRow.Object.ImageTitle)
			}

			if imgRow.Object.ImageURL != "http://example.com/new_image.jpg" {
				t.Errorf("Expected ImageModel ImageURL to be '%s', got: '%s'", model.Image.ImageURL, imgRow.Object.ImageURL)
			}
		})

		t.Run("FetchStatefulModelAndImageFromDB", func(t *testing.T) {
			var row, err = queries.GetQuerySet(&StatefulModel{}).
				Select("*", "Image.*").
				Filter("ID", model.ID).
				Get()
			if err != nil {
				t.Fatalf("Failed to get StatefulModel from DB: %v", err)
			}

			if row.Object.ID != model.ID {
				t.Errorf("Expected StatefulModel ID to be %d, got: %d", model.ID, row.Object.ID)
			}

			if row.Object.FirstName != "Jane" {
				t.Errorf("Expected StatefulModel FirstName to be 'Jane', got: %s", row.Object.FirstName)
			}

			if row.Object.LastName != "Doe" {
				t.Errorf("Expected StatefulModel LastName to be 'Doe', got: %s", row.Object.LastName)
			}

			if row.Object.Age != 30 {
				t.Errorf("Expected StatefulModel Age to be 30, got: %d", row.Object.Age)
			}

			if len(row.Object.BinData) != 3 || row.Object.BinData[0] != 4 || row.Object.BinData[1] != 5 || row.Object.BinData[2] != 6 {
				t.Errorf("Expected StatefulModel BinData to be [4, 5, 6], got: %v", row.Object.BinData)
			}

			if row.Object.MapData == nil {
				t.Errorf("Expected StatefulModel MapData to be non-nil after save")
			}

			if len(row.Object.MapData) != 2 {
				t.Errorf("Expected StatefulModel MapData to have 2 keys, got: %d", len(row.Object.MapData))
			}

			if row.Object.MapData["key1"] != "value2" {
				t.Errorf("Expected StatefulModel MapData['key1'] to be 'value2', got: %v", row.Object.MapData["key1"])
			}

			// convert to float64 for comparison after JSON unmarshalling
			if row.Object.MapData["key2"] != float64(84) {
				t.Errorf("Expected StatefulModel MapData['key2'] to be 84, got: %v (%T)", row.Object.MapData["key2"], row.Object.MapData["key2"])
			}

			if row.Object.Image == nil {
				t.Fatal("Expected StatefulModel Image to be non-nil after save")
			}

			if row.Object.Image.ID != model.Image.ID {
				t.Errorf("Expected StatefulModel Image ID to be %d, got: %d", model.Image.ID, row.Object.Image.ID)
			}

			if row.Object.Image.ImageTitle != "New Image Title" {
				t.Errorf("Expected StatefulModel Image ImageTitle to be 'New Image Title', got: %s", row.Object.Image.ImageTitle)
			}

			if row.Object.Image.ImageURL != "http://example.com/new_image.jpg" {
				t.Errorf("Expected StatefulModel Image ImageURL to be 'http://example.com/new_image.jpg', got: %s", row.Object.Image.ImageURL)
			}
		})
	})

}
