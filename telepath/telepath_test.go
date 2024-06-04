package telepath_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Nigel2392/telepath"
)

func init() {

}

var AlbumAdapter = &telepath.TelepathAdapter{
	JSConstructor: "js.funcs.Album",
	GetJSArgs: func(obj interface{}) []interface{} {
		album := obj.(*Album)
		return []interface{}{album.Name, album.Artists}
	},
}

var ArtistAdapter = &telepath.TelepathAdapter{
	JSConstructor: "js.funcs.Artist",
	GetJSArgs: func(obj interface{}) []interface{} {
		artist := obj.(*Artist)
		return []interface{}{artist.Name}
	},
}

type Album struct {
	Name    string
	Artists []*Artist
}

type Artist struct {
	Name string
}

type TelepathEncoder struct {
	json.Encoder
}

func TestPacking(t *testing.T) {
	telepath.Register(AlbumAdapter, &Album{})
	telepath.Register(ArtistAdapter, &Artist{})

	t.Run("TestPackObject", func(t *testing.T) {
		var object = &Album{Name: "Hello"}
		var ctx = telepath.NewContext()
		var result, err = ctx.Pack(object)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		var chk = result.(telepath.TelepathValue)
		if chk.Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk.Type)
		}

		if chk.Args[0] != "Hello" {
			t.Errorf("Expected Hello, got %v", chk.Args[0])
		}

	})

	t.Run("TestPackList", func(t *testing.T) {
		var object = []*Album{
			{Name: "Hello 1"},
			{Name: "Hello 2"},
		}

		var ctx = telepath.NewContext()
		var result, err = ctx.Pack(object)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		var chk = result.(telepath.TelepathValue)
		if chk.List == nil {
			t.Errorf("Expected list, got nil")
			return
		}

		if len(chk.List) != 2 {
			t.Errorf("Expected 2, got %v", len(chk.List))
			return
		}

		if chk.List[0].(telepath.TelepathValue).Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk.List[0].(telepath.TelepathValue).Type)
		}

		if chk.List[0].(telepath.TelepathValue).Args[0] != "Hello 1" {
			t.Errorf("Expected Hello 1, got %v", chk.List[0].(telepath.TelepathValue).Args[0])
		}

		if chk.List[1].(telepath.TelepathValue).Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk.List[1].(telepath.TelepathValue).Type)
		}

		if chk.List[1].(telepath.TelepathValue).Args[0] != "Hello 2" {
			t.Errorf("Expected Hello 2, got %v", chk.List[1].(telepath.TelepathValue).Args[0])
		}
	})

	t.Run("TestPackMap", func(t *testing.T) {

		var object = map[string]*Album{
			"1": {Name: "Hello 1"},
			"2": {Name: "Hello 2"},
		}

		var ctx = telepath.NewContext()
		var result, err = ctx.Pack(object)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		var chk = result.(map[string]interface{})
		if len(chk) != 2 {
			t.Errorf("Expected 2, got %v", len(chk))
			return
		}

		if chk["1"].(telepath.TelepathValue).Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk["1"].(telepath.TelepathValue).Type)
		}

		if chk["1"].(telepath.TelepathValue).Args[0] != "Hello 1" {
			t.Errorf("Expected Hello 1, got %v", chk["1"].(telepath.TelepathValue).Args[0])
		}

		if chk["2"].(telepath.TelepathValue).Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk["2"].(telepath.TelepathValue).Type)
		}

		if chk["2"].(telepath.TelepathValue).Args[0] != "Hello 2" {
			t.Errorf("Expected Hello 2, got %v", chk["2"].(telepath.TelepathValue).Args[0])
		}
	})

	t.Run("TestDictReservedWords", func(t *testing.T) {
		var object = map[string]interface{}{
			"_artist": &Album{Name: "Hello"},
			"_type":   "Album",
		}

		var ctx = telepath.NewContext()
		var result, err = ctx.Pack(object)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		var chk = result.(telepath.TelepathValue)
		if chk.Dict == nil {
			t.Errorf("Expected dict, got nil")
			return
		}

		if chk.Dict["_artist"].(telepath.TelepathValue).Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk.Dict["_artist"].(telepath.TelepathValue).Type)
		}

		if chk.Dict["_artist"].(telepath.TelepathValue).Args[0] != "Hello" {
			t.Errorf("Expected Hello, got %v", chk.Dict["_artist"].(telepath.TelepathValue).Args[0])
		}

		if len(chk.Dict["_artist"].(telepath.TelepathValue).Args) != 2 {
			t.Errorf("Expected 2, got %v", len(chk.Dict["_artist"].(telepath.TelepathValue).Args))
		}

		if chk.Dict["_type"] != "Album" {
			t.Errorf("Expected Album, got %v", chk.Dict["_type"])
		}

	})

	t.Run("TestRecursiveArgPacking", func(t *testing.T) {
		var object = &Album{
			Name: "Hello",
			Artists: []*Artist{
				{Name: "Artist 1"},
				{Name: "Artist 2"},
			},
		}

		telepath.Register(ArtistAdapter, &Artist{})

		var ctx = telepath.NewContext()
		var result, err = ctx.Pack(object)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		var chk = result.(telepath.TelepathValue)
		if chk.Type != "js.funcs.Album" {
			t.Errorf("Expected js.funcs.Album, got %v", chk.Type)
		}

		if chk.Args[0] != "Hello" {
			t.Errorf("Expected Hello, got %v", chk.Args[0])
		}

		if chk.Args[1].(telepath.TelepathValue).List[0].(telepath.TelepathValue).Type != "js.funcs.Artist" {
			t.Errorf("Expected js.funcs.Artist, got %v", chk.Args[1].(telepath.TelepathValue).List[0].(telepath.TelepathValue).Type)
		}

		if chk.Args[1].(telepath.TelepathValue).List[0].(telepath.TelepathValue).Args[0] != "Artist 1" {
			t.Errorf("Expected Artist 1, got %v", chk.Args[1].(telepath.TelepathValue).List[1].(telepath.TelepathValue).Args[0])
		}

		if chk.Args[1].(telepath.TelepathValue).List[1].(telepath.TelepathValue).Type != "js.funcs.Artist" {
			t.Errorf("Expected js.funcs.Artist, got %v", chk.Args[1].(telepath.TelepathValue).List[0].(telepath.TelepathValue).Type)
		}

		if chk.Args[1].(telepath.TelepathValue).List[1].(telepath.TelepathValue).Args[0] != "Artist 2" {
			t.Errorf("Expected Artist 2, got %v", chk.Args[1].(telepath.TelepathValue).List[1].(telepath.TelepathValue).Args[0])
		}
	})
}

type StringLike struct {
	Value string
}

var StringLikeAdapter = &telepath.TelepathAdapter{
	JSConstructor: "js.funcs.StringLike",
	GetJSArgs: func(obj interface{}) []interface{} {
		str := obj.(*StringLike)
		return []interface{}{strings.ToUpper(str.Value)}
	},
}

func TestPackingToString(t *testing.T) {
	var value = []any{
		&StringLike{Value: "hello"},
		"world",
	}

	telepath.Register(StringLikeAdapter, &StringLike{})

	var ctx = telepath.NewContext()
	var result, err = ctx.Pack(value)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		return
	}

	var chk = result.(telepath.TelepathValue)
	if chk.List[0].(telepath.TelepathValue).Type != "js.funcs.StringLike" {
		t.Errorf("Expected js.funcs.StringLike, got %v", chk.List[0].(telepath.TelepathValue).Type)
	}

	if chk.List[0].(telepath.TelepathValue).Args[0] != "HELLO" {
		t.Errorf("Expected HELLO, got %v", chk.List[0].(telepath.TelepathValue).Args[0])
	}

	if chk.List[1] != "world" {
		t.Errorf("Expected world, got %v", chk.List[1])
	}
}
