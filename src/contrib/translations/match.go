package translations

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

type Translation struct {
	Path       string
	Paths      []string
	Comment    string
	Line       int
	Col        int
	Text       trans.Untranslated
	Plural     trans.Untranslated
	Preference int // the higher the number, the more preferred this Match is
	Locales    *orderedmap.OrderedMap[trans.Locale, []trans.Translation]
}

type ymlTranslation struct {
	Path       string             `yaml:"path"`
	Text       trans.Untranslated `yaml:"text"`
	Plural     trans.Untranslated `yaml:"plural,omitempty"`
	Preference int                `yaml:"preference,omitempty"`
}

var (
	_ yaml.Marshaler   = Translation{}
	_ yaml.Unmarshaler = &Translation{}
)

func (m *Translation) UnmarshalYAML(node *yaml.Node) error {

	var yml ymlTranslation
	if err := node.Decode(&yml); err != nil {
		return fmt.Errorf("failed to decode Match: %w", err)
	}

	m.Text = yml.Text
	m.Plural = yml.Plural
	m.Preference = yml.Preference

	switch {
	case node.HeadComment != "":
		m.Comment = node.HeadComment
	case node.LineComment != "":
		m.Comment = node.LineComment
	}

	var localesNode *yaml.Node
	for idx, item := range node.Content {
		if idx%2 != 0 {
			continue
		}

		if item.Value == "locales" {
			if idx+1 < len(node.Content) {
				localesNode = node.Content[idx+1]
			} else {
				return fmt.Errorf("expected value node after 'locales', but found none")
			}
		}
	}

	if localesNode == nil {
		return nil // No locales defined, nothing to do
	}

	if localesNode.Kind != yaml.MappingNode {
		return fmt.Errorf(
			"expected mapping node for locales, got %d", localesNode.Kind,
		)
	}

	m.Locales = orderedmap.NewOrderedMap[trans.Locale, []trans.Translation]()
	for i := 0; i < len(localesNode.Content); {
		if i+1 >= len(localesNode.Content) {
			return fmt.Errorf("expected value node after key %s, but found none", localesNode.Content[i].Value)
		}
		var key = localesNode.Content[i].Value
		var valueNode = localesNode.Content[i+1]

		switch valueNode.Kind {
		case yaml.ScalarNode:
			// Single translation
			m.Locales.Set(key, []string{valueNode.Value})
		case yaml.MappingNode:
			// Multiple translations
			var translations = make([]string, 0)
			for j := 0; j < len(valueNode.Content); j += 2 {
				if j+1 >= len(valueNode.Content) {
					return fmt.Errorf("expected value node after key %s, but found none", valueNode.Content[j].Value)
				}
				translations = append(translations, valueNode.Content[j+1].Value)
			}
			m.Locales.Set(key, translations)
		default:
			return fmt.Errorf(
				"unexpected node kind %d for locale %s, expected scalar or mapping",
				valueNode.Kind, key,
			)
		}

		i += 2 // Move to the next key-value pair
	}

	return nil
}

func (m Translation) MarshalYAML() (interface{}, error) {
	var n = yaml.Node{
		Kind:    yaml.MappingNode,
		Tag:     "!!map",
		Content: []*yaml.Node{
			//	{
			//		Kind:  yaml.ScalarNode,
			//		Tag:   "!!str",
			//		Value: "path",
			//	},
			//	{
			//		Kind:  yaml.ScalarNode,
			//		Tag:   "!!str",
			//		Value: fmt.Sprintf("%s:%d:%d", m.Path, m.Line, m.Col),
			//	},
		},
	}

	//	var pathStyle = yaml.FlowStyle
	//	if len(m.Paths) > 0 {
	//		pathStyle = yaml.FoldedStyle
	//	}
	//
	//	n.Content = append(n.Content,
	//		&yaml.Node{
	//			Kind:  yaml.ScalarNode,
	//			Tag:   "!!str",
	//			Value: "paths",
	//		},
	//		&yaml.Node{
	//			Kind:    yaml.SequenceNode,
	//			Style:   pathStyle,
	//			Tag:     "!!seq",
	//			Content: make([]*yaml.Node, len(m.Paths)+1),
	//		},
	//	)
	//	for i, path := range append([]string{fmt.Sprintf("%s:%d:%d", m.Path, m.Line, m.Col)}, m.Paths...) {
	//		n.Content[len(n.Content)-1].Content[i] = &yaml.Node{
	//			Kind:  yaml.ScalarNode,
	//			Tag:   "!!str",
	//			Value: path,
	//		}
	//	}

	n.Content = append(n.Content,
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: "text",
		},
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: m.Text,
		},
	)

	if m.Plural != "" {
		n.Content = append(n.Content,
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "plural",
			},
			&yaml.Node{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: m.Plural,
			},
		)
	}

	if m.Locales != nil {
		var nodeContent = make([]*yaml.Node, 0, m.Locales.Len()*2)
		for head := m.Locales.Front(); head != nil; head = head.Next() {
			nodeContent = append(nodeContent,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: head.Key,
				},
			)

			if len(head.Value) == 1 {
				nodeContent = append(nodeContent, &yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: head.Value[0],
				})
			} else {

				var mapNode = &yaml.Node{
					Kind:    yaml.MappingNode,
					Tag:     "!!map",
					Content: make([]*yaml.Node, 0, len(head.Value)*2),
				}

				for idx, text := range head.Value {
					mapNode.Content = append(mapNode.Content,
						&yaml.Node{
							Kind:  yaml.ScalarNode,
							Tag:   "!!int",
							Value: strconv.Itoa(idx),
						},
						&yaml.Node{
							Kind:  yaml.ScalarNode,
							Tag:   "!!str",
							Value: text,
						},
					)
				}

				nodeContent = append(nodeContent, mapNode)
			}

		}

		if len(nodeContent) > 0 {
			n.Content = append(n.Content,
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: "locales",
				},
				// Add the locales node as the value for "locales"
				// This is a nested mapping node
				&yaml.Node{
					Kind:    yaml.MappingNode,
					Tag:     "!!map",
					Content: nodeContent,
				},
			)
		}
	}

	n.HeadComment = strings.Join(append(
		[]string{fmt.Sprintf("%s:%d:%d", m.Path, m.Line, m.Col)}, m.Paths...),
		"\n",
	)

	return &n, nil
}
