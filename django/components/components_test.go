package components_test

import (
	"bytes"
	"context"
	"testing"

	"github.com/Nigel2392/django/components"
)

func renderToString(c components.Component) string {
	var buf = new(bytes.Buffer)
	var ctx = context.Background()
	c.Render(ctx, buf)
	return buf.String()
}

type renderTest struct {
	testName  string
	name      string
	namespace string
	renderNs  string
	component components.ComponentFunc
	setup     func(t *testing.T, r components.Registry)
	args      []interface{}
	expected  string
}

var tests = []renderTest{
	{
		testName: "TestRenderHeading",
		name:     "heading",
		args:     []interface{}{"Hello, World!"},
		expected: "<h1>Hello, World!</h1>",
		setup: func(t *testing.T, r components.Registry) {
			r.Register("heading", heading)
		},
	},
	{
		testName: "TestRenderList",
		name:     "list",
		args:     []interface{}{[]string{"one", "two", "three"}},
		expected: "<ul><li>one</li><li>two</li><li>three</li></ul>",
		setup: func(t *testing.T, r components.Registry) {
			r.Register("list", list)
		},
	},
	{
		testName: "TestRenderButton",
		name:     "button",
		args:     []interface{}{"Click me!"},
		expected: "<button>Click me!</button>",
		setup: func(t *testing.T, r components.Registry) {
			r.Register("button", button)
		},
	},
	{
		testName: "TestRenderTable",
		name:     "table",
		args:     []interface{}{[]string{"one", "two", "three"}, [][]string{{"one", "two", "three"}, {"one", "two", "three"}, {"one", "two", "three"}}},
		expected: "<table><thead><tr><th>one</th><th>two</th><th>three</th></tr></thead> <tbody><tr><td>one</td><td>two</td><td>three</td></tr><tr><td>one</td><td>two</td><td>three</td></tr><tr><td>one</td><td>two</td><td>three</td></tr></tbody></table>",
		setup: func(t *testing.T, r components.Registry) {
			r.Register("table", table)
		},
	},
	{
		testName: "TestNamespacedRender",
		name:     "components.heading",
		args:     []interface{}{"Hello, World!"},
		expected: "<h1>Hello, World!</h1>",
		setup: func(t *testing.T, r components.Registry) {
			var ns = r.(*components.ComponentRegistry).Namespace("components")
			ns.Register("heading", heading)
		},
	},
	{
		testName:  "TestRenderNamespace",
		name:      "heading",
		namespace: "components",
		renderNs:  "components",
		args:      []interface{}{"Hello, World!"},
		expected:  "<h1>Hello, World!</h1>",
		setup: func(t *testing.T, r components.Registry) {
			r.Register("heading", heading)
		},
	},
	{
		testName: "TestRegisterNamespaceMultipleDots",
		name:     "heading.func",
		renderNs: "components",
		args:     []interface{}{"Hello, World!"},
		expected: "<h1>Hello, World!</h1>",
		setup: func(t *testing.T, r components.Registry) {
			r.Register("components.heading.func", heading)
		},
	},
	{
		testName: "TestRenderNamespaceMultipleDots",
		name:     "components.heading.func",
		args:     []interface{}{"Hello, World!"},
		expected: "<h1>Hello, World!</h1>",
		setup: func(t *testing.T, r components.Registry) {
			var ns = r.(*components.ComponentRegistry).Namespace("components")
			ns.Register("heading.func", heading)
		},
	},
}

func TestRegister(t *testing.T) {
	for _, test := range tests {

		t.Run(test.testName, func(t *testing.T) {
			var r = components.NewComponentRegistry()
			var reg components.Registry = r
			if test.namespace != "" {
				reg = r.Namespace(test.namespace)
			}
			if test.setup != nil {
				test.setup(t, reg)
			}
			if test.component != nil {
				reg.Register(test.name, test.component)
			}
			var c components.Component

			var renderReg components.Registry = r
			if test.renderNs != "" {
				renderReg = r.Namespace(test.renderNs)
			}

			if test.args != nil {
				c = renderReg.Render(test.name, test.args...)
			} else {
				c = renderReg.Render(test.name)
			}
			if c == nil {
				t.Errorf("Component %q not found", test.name)
				return
			}
			var got = renderToString(c)
			if got != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, got)
			}
		})
	}
}
