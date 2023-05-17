package engine_test

import (
	"reflect"
	"testing"

	"github.com/Nigel2392/go-django/core/serializer/engine"
	"github.com/Nigel2392/go-django/core/serializer/marshallers"
)

var testJSON = `{
	"id": "645d6ba37c0ceef387258080",
	"index": 19,
	"guid": "0e52f6d4-e35d-443c-af23-c8d2c690b8b3",
	"isActive": true
}`

var testYAML = `id: 645d6ba37c0ceef387258080
index: 19
guid: 0e52f6d4-e35d-443c-af23-c8d2c690b8b3
active: true`

type Struct struct {
	ID     string
	Index  int
	GUID   string
	Active bool
	// RequiredField bool
}

func TestMakeSerializeStruct(t *testing.T) {
	var sstruct = Struct{}
	var err = serializerEngine.Deserialize([]byte(testJSON), &sstruct)
	if err != nil {
		t.Error(err)
	}
	t.Log(sstruct)
	if sstruct.ID != "645d6ba37c0ceef387258080" {
		t.Error("ID not equal")
		return
	}
	if sstruct.Index != 19 {
		t.Error("Index not equal")
		return
	}
	if sstruct.GUID != "0e52f6d4-e35d-443c-af23-c8d2c690b8b3" {
		t.Error("GUID not equal")
		return
	}
	if sstruct.Active != true {
		t.Error("Active not equal")
		return
	}

	var data, err2 = serializerEngine.Serialize(sstruct)
	if err2 != nil {
		t.Error(err2)
	}
	t.Log(string(data))
}

var serializerEngine = engine.NewEngine("json",
	engine.Field{AbsName: "ID", EncName: "id", Type: reflect.TypeOf("")},
	engine.Field{AbsName: "Index", EncName: "index", Type: reflect.TypeOf(0)},
	engine.Field{AbsName: "GUID", EncName: "guid", Type: reflect.TypeOf("")},
	engine.Field{AbsName: "Active", EncName: "isActive", Type: reflect.TypeOf(false)},

	// serializer.Field{AbsName: "RequiredField", EncName: "requiredField", Required: true, Type: reflect.TypeOf(false)},
)

func TestMakeSerializeSlice(t *testing.T) {
	var sstruct = Struct{}
	var err = serializerEngine.Deserialize([]byte(testJSON),
		&sstruct.ID,
		&sstruct.Index,
		&sstruct.GUID,
		&sstruct.Active,
	)
	if err != nil {
		t.Error(err)
	}
	t.Log(sstruct)
	if sstruct.ID != "645d6ba37c0ceef387258080" {
		t.Error("ID not equal")
		return
	}
	if sstruct.Index != 19 {
		t.Error("Index not equal")
		return
	}
	if sstruct.GUID != "0e52f6d4-e35d-443c-af23-c8d2c690b8b3" {
		t.Error("GUID not equal")
		return
	}
	if sstruct.Active != true {
		t.Error("Active not equal")
		return
	}

	var data, err2 = serializerEngine.Serialize(
		sstruct.ID,
		sstruct.Index,
		sstruct.GUID,
		sstruct.Active,
	)
	if err2 != nil {
		t.Error(err2)
	}
	t.Log(string(data))
}

func TestMakeSerializeMap(t *testing.T) {
	var sMap = make(map[string]interface{})
	var err = serializerEngine.Deserialize([]byte(testJSON), &sMap)
	if err != nil {
		t.Error(err)
	}
	t.Log(sMap)
	if sMap["ID"] != "645d6ba37c0ceef387258080" {
		t.Error("ID not equal")
		return
	}
	if sMap["Index"] != 19 {
		t.Error("Index not equal")
		return
	}
	if sMap["GUID"] != "0e52f6d4-e35d-443c-af23-c8d2c690b8b3" {
		t.Error("GUID not equal")
		return
	}
	if sMap["Active"] != true {
		t.Error("Active not equal")
		return
	}

	var data, err2 = serializerEngine.Serialize(sMap)
	if err2 != nil {
		t.Error(err2)
	}
	t.Log(string(data))
}

func TestYAMLSerializer(t *testing.T) {
	serializerEngine.Serializer = &marshallers.YAML{}
	var sstruct = Struct{}
	var err = serializerEngine.Deserialize([]byte(testYAML), &sstruct)
	if err != nil {
		t.Error(err)
		return
	}
	t.Log(sstruct)

	if sstruct.ID != "645d6ba37c0ceef387258080" {
		t.Error("ID not equal")
		return
	}
	if sstruct.Index != 19 {
		t.Error("Index not equal")
		return
	}
	if sstruct.GUID != "0e52f6d4-e35d-443c-af23-c8d2c690b8b3" {
		t.Error("GUID not equal")
		return
	}
	if sstruct.Active != true {
		t.Error("Active not equal")
		return
	}

	var data, err2 = serializerEngine.Serialize(sstruct)
	if err2 != nil {
		t.Error(err2)
	}
	t.Log(string(data))
}

//	func TestSerializeXML(t *testing.T) {
//		engine := serializer.NewSerializer("xml",
//			serializer.Field{AbsName: "Person", EncName: "person", Type: reflect.TypeOf(xml.Name{})},
//			serializer.Field{AbsName: "ID", EncName: "id", Type: reflect.TypeOf("")},
//			serializer.Field{AbsName: "Index", EncName: "index", Type: reflect.TypeOf(0)},
//			serializer.Field{AbsName: "GUID", EncName: "guid", Type: reflect.TypeOf("")},
//			serializer.Field{AbsName: "Active", EncName: "active", Type: reflect.TypeOf(false)},
//			// serializer.Field{AbsName: "RequiredField", EncName: "requiredField", Required: true, Type: reflect.TypeOf(false)},
//		)
//		engine.Serializer = &serializer.XMLSerializer{}
//		var sstruct = Struct{}
//		var err = engine.Deserialize([]byte(testXML), &sstruct)
//		if err != nil {
//			t.Error(err)
//			return
//		}
//
//		t.Log(sstruct)
//
//		if sstruct.ID != "645d6ba37c0ceef387258080" {
//			t.Error("ID not equal")
//			return
//		}
//		if sstruct.Index != 19 {
//			t.Error("Index not equal")
//			return
//		}
//		if sstruct.GUID != "0e52f6d4-e35d-443c-af23-c8d2c690b8b3" {
//			t.Error("GUID not equal")
//			return
//		}
//		if sstruct.Active != true {
//			t.Error("Active not equal")
//			return
//		}
//
//		var data, err2 = engine.Serialize(sstruct)
//		if err2 != nil {
//			t.Error(err2)
//			return
//		}
//		t.Log(string(data))
//	}
//
