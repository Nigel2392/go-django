package blocks_test

import (
	"context"
	"net/mail"
	"reflect"
	"testing"
	"time"

	"github.com/Nigel2392/go-django/src/contrib/blocks"
	"github.com/google/uuid"
)

func NewListBlock() *blocks.ListBlock {
	var b = blocks.NewListBlock(NewSimpleStructBlock())
	b.SetName("test_list_block")
	return b
}

var (
	ListBlockDataRaw = map[string][]string{
		"test_list_block-added":      {"2"},
		"test_list_block-id-0":       {uuid.Nil.String()},
		"test_list_block-order-0":    {"0"},
		"test_list_block-0-name":     {"John Doe"},
		"test_list_block-0-age":      {"30"},
		"test_list_block-0-email":    {"test@localhost"},
		"test_list_block-0-password": {"password"},
		"test_list_block-0-date":     {"2021-01-01"},
		"test_list_block-0-datetime": {"2021-01-01T00:00:00"},
		"test_list_block-id-1":       {uuid.Nil.String()},
		"test_list_block-order-1":    {"1"},
		"test_list_block-1-name":     {"Jane Doe"},
		"test_list_block-1-age":      {"25"},
		"test_list_block-1-email":    {"test2@localhost"},
		"test_list_block-1-password": {"password2"},
		"test_list_block-1-date":     {"2021-01-02"},
		"test_list_block-1-datetime": {"2021-01-02T00:00:00"},
	}

	ListBlockDataRawCmp = blocks.ListBlockData{
		{
			ID:    uuid.Nil,
			Order: 0,
			Data: map[string]interface{}{
				"name":     "John Doe",
				"age":      "30",
				"email":    "test@localhost",
				"password": "password",
				"date":     "2021-01-01",
				"datetime": "2021-01-01T00:00:00",
			},
		},
		{
			ID:    uuid.Nil,
			Order: 1,
			Data: map[string]interface{}{
				"name":     "Jane Doe",
				"age":      "25",
				"email":    "test2@localhost",
				"password": "password2",
				"date":     "2021-01-02",
				"datetime": "2021-01-02T00:00:00",
			},
		},
	}

	ListBlockDataGo = blocks.ListBlockData{
		{
			ID:    uuid.Nil,
			Order: 0,
			Data: map[string]interface{}{
				"name":     "John Doe",
				"age":      30,
				"email":    must(mail.ParseAddress("test@localhost")),
				"password": "password",
				"date":     must(time.Parse("2006-01-02", "2021-01-01")),
				"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-01T00:00:00")),
			},
		},
		{
			ID:    uuid.Nil,
			Order: 1,
			Data: map[string]interface{}{
				"name":     "Jane Doe",
				"age":      25,
				"email":    must(mail.ParseAddress("test2@localhost")),
				"password": "password2",
				"date":     must(time.Parse("2006-01-02", "2021-01-02")),
				"datetime": must(time.Parse("2006-01-02T15:04:05", "2021-01-02T00:00:00")),
			},
		},
	}
)

func TestListBlock(t *testing.T) {
	var b = NewListBlock()

	t.Run("ValueFromDataDict", func(t *testing.T) {
		var data, err = b.ValueFromDataDict(context.Background(), ListBlockDataRaw, nil, "test_list_block")

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if data == nil {
			t.Errorf("Expected data, got nil")
		}

		var d = data.(blocks.ListBlockData)

		for i, v := range d {
			if !reflect.DeepEqual(*v, *ListBlockDataRawCmp[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataRawCmp[i], *v)
			}
		}

		if len(d) != len(ListBlockDataRawCmp) {
			t.Errorf("Expected length %d, got %d", len(ListBlockDataRawCmp), len(d))
		}
	})

	t.Run("ValueToGo", func(t *testing.T) {
		var data, err = b.ValueToGo(ListBlockDataRawCmp)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var d = data.(blocks.ListBlockData)

		for i, v := range d {
			if !reflect.DeepEqual(*v, *ListBlockDataGo[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataGo[i], v)
			}
		}
	})

	t.Run("ValueToForm", func(t *testing.T) {
		var (
			data = b.ValueToForm(ListBlockDataGo)
			d    = data.(blocks.ListBlockData)
		)

		for i, v := range d {
			if !reflect.DeepEqual(*v, *ListBlockDataRawCmp[i]) {
				t.Errorf("Expected %v, got %v", *ListBlockDataRawCmp[i], *v)
			}
		}
	})

	t.Run("ConversionsEqual", func(t *testing.T) {
		var (
			data, err = b.ValueToGo(ListBlockDataRawCmp)
			data2     = b.ValueToForm(data)
		)

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if !reflect.DeepEqual(data2, ListBlockDataRawCmp) {
			t.Errorf("Expected %v, got %v", ListBlockDataRawCmp, data2)
		}
	})
}
