package editor_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	django "github.com/Nigel2392/go-django/src"
	"github.com/Nigel2392/go-django/src/contrib/editor"
	_ "github.com/Nigel2392/go-django/src/contrib/editor/features"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/forms/media"
	"github.com/Nigel2392/mux"
)

var (
	_ editor.FeatureBlock = (*MockFeatureBlock)(nil)
	_ editor.BaseFeature  = MockBaseFeature{}
)

func init() {
	editor.Register(MockBaseFeature{
		NameVal: "MockBaseFeature",
	})
}

type MockFeatureBlock struct {
	IDVal   string
	TypeVal string
}

func (m MockFeatureBlock) ID() string {
	return m.IDVal
}

func (m MockFeatureBlock) Type() string {
	return m.TypeVal
}

func (m MockFeatureBlock) Feature() editor.BaseFeature {
	return MockBaseFeature{}
}

func (m MockFeatureBlock) Render(ctx context.Context, w io.Writer) error {
	_, err := w.Write([]byte(m.TypeVal))
	return err
}

func (m MockFeatureBlock) Attribute(key string, value any) {}

func (m MockFeatureBlock) Attributes() map[string]interface{} {
	return map[string]interface{}{}
}

func (m MockFeatureBlock) Data() editor.BlockData {
	return editor.BlockData{
		ID:    m.IDVal,
		Type:  m.TypeVal,
		Data:  map[string]interface{}{},
		Tunes: map[string]interface{}{},
	}
}

type MockBaseFeature struct {
	NameVal string
}

func (m MockBaseFeature) Name() string {
	return m.NameVal
}

func (m MockBaseFeature) Config(widgetContext ctx.Context) map[string]interface{} {
	return map[string]interface{}{}
}

func (m MockBaseFeature) OnValidate(editor.BlockData) error {
	return nil
}

func (m MockBaseFeature) OnRegister(mx django.Mux) error {
	mx.Any("/mock", mux.NewHandler(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("mock"))
	}), "mock")
	return nil
}

func (m MockBaseFeature) Constructor() string {
	return "MockConstructor"
}

func (m MockBaseFeature) Media() media.Media {
	return media.NewMedia()
}

func (m MockBaseFeature) Render(data editor.BlockData) editor.FeatureBlock {
	return MockFeatureBlock{
		IDVal:   data.ID,
		TypeVal: data.Type,
	}
}

func getMockData() *editor.EditorJSBlockData {
	return &editor.EditorJSBlockData{
		Time:    42,
		Version: "2.22.2",
		Blocks: []editor.FeatureBlock{
			MockFeatureBlock{IDVal: "1", TypeVal: "MockBaseFeature"},
			MockFeatureBlock{IDVal: "2", TypeVal: "MockBaseFeature"},
			MockFeatureBlock{IDVal: "3", TypeVal: "MockBaseFeature"},
			MockFeatureBlock{IDVal: "4", TypeVal: "MockBaseFeature"},
			MockFeatureBlock{IDVal: "5", TypeVal: "MockBaseFeature"},
		},
		Features: editor.Features(
			"MockType1", "MockType2", "MockType3", "MockType4", "MockType5",
		),
	}
}

func TestUnmarshalEditorJSBlockData(t *testing.T) {
	var data = getMockData()
	var serialized, err = editor.JSONMarshalEditorData(data)
	if err != nil {
		t.Fatalf("Error marshalling: %v", err)
	}

	deserialized, err := editor.JSONUnmarshalEditorData(nil, serialized)
	if err != nil {
		t.Fatalf("Error unmarshalling: %v", err)
	}

	if len(data.Blocks) != len(deserialized.Blocks) {
		t.Fatalf("Expected %d blocks, got %d", len(data.Blocks), len(deserialized.Blocks))
	}

	if data.Time != deserialized.Time {
		t.Errorf("Expected Time %d, got %d", data.Time, deserialized.Time)
	}

	if data.Version != deserialized.Version {
		t.Errorf("Expected Version %s, got %s", data.Version, deserialized.Version)
	}

	for i, block := range data.Blocks {
		if block.ID() != deserialized.Blocks[i].ID() {
			t.Errorf("Expected ID %s, got %s", block.ID(), deserialized.Blocks[i].ID())
		}

		if block.Type() != deserialized.Blocks[i].Type() {
			t.Errorf("Expected Type %s, got %s", block.Type(), deserialized.Blocks[i].Type())
		}

		if block.Feature().Name() != deserialized.Blocks[i].Feature().Name() {
			t.Errorf("Expected Feature Name %s, got %s", block.Feature().Name(), deserialized.Blocks[i].Feature().Name())
		}
	}
}

func TestOnRegisterRun(t *testing.T) {
	const (
		app_port = "8123"
		app_host = "127.0.0.1"
	)

	var a = django.App(
		django.Configure(map[string]interface{}{
			django.APPVAR_DEBUG: true,
			django.APPVAR_PORT:  app_port,
			django.APPVAR_HOST:  app_host,
		}),
		django.Flag(
			django.FlagSkipDepsCheck,
			django.FlagSkipCmds,
		),
		django.Apps(
			editor.NewAppConfig,
		),
	)

	err := a.Initialize()
	if err != nil {
		t.Fatalf("Error initializing app: %v", err)
	}

	go func(t *testing.T) {
		err := a.Serve()
		if err != nil {
			t.Errorf("Error serving app: %v", err)
		}
	}(t)

	resp, err := http.Get(fmt.Sprintf(
		"http://%s:%s%s",
		app_host, app_port,
		django.Reverse("editor:mock"),
	))
	if err != nil {
		t.Fatalf("Error making request: %v", err)
	}

	var b = new(strings.Builder)
	_, err = io.Copy(b, resp.Body)
	if err != nil {
		t.Fatalf("Error reading response: %v", err)
	}

	if b.String() != "mock" {
		t.Errorf("Expected response 'mock', got %s", b.String())
	}
}
