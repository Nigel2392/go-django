package translations

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/pkg/yml"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

type Match struct {
	Path       string
	Comment    string
	Line       int
	Col        int
	Text       trans.Untranslated
	Preference int // the higher the number, the more preferred this Match is
	Locales    *orderedmap.OrderedMap[trans.Locale, trans.Translation]
}

type ymlMatch struct {
	Path       string                                          `yaml:"path"`
	Text       trans.Untranslated                              `yaml:"text"`
	Preference int                                             `yaml:"preference,omitempty"`
	Locales    yml.OrderedMap[trans.Locale, trans.Translation] `yaml:"locales,omitempty"`
}

var (
	_ yaml.Marshaler   = Match{}
	_ yaml.Unmarshaler = &Match{}
)

func (m *Match) UnmarshalYAML(node *yaml.Node) error {

	var yml ymlMatch
	if err := node.Decode(&yml); err != nil {
		return fmt.Errorf("failed to decode Match: %w", err)
	}

	var split = strings.SplitN(yml.Path, ":", 3)
	if len(split) < 3 {
		return fmt.Errorf("invalid path format: %s", yml.Path)
	}

	line, err := strconv.Atoi(split[1])
	if err != nil {
		return fmt.Errorf("invalid line number in path %s: %w", yml.Path, err)
	}

	col, err := strconv.Atoi(split[2])
	if err != nil {
		return fmt.Errorf("invalid column number in path %s: %w", yml.Path, err)
	}

	m.Path = split[0]
	m.Line = line
	m.Col = col
	m.Text = yml.Text
	m.Preference = yml.Preference

	switch {
	case node.HeadComment != "":
		m.Comment = node.HeadComment
	case node.LineComment != "":
		m.Comment = node.LineComment
	}

	m.Locales = orderedmap.NewOrderedMap[string, string]()

	if yml.Locales.OrderedMap != nil {
		for head := yml.Locales.OrderedMap.Front(); head != nil; head = head.Next() {
			m.Locales.Set(head.Key, head.Value)
		}
	}

	return nil
}

func (m Match) MarshalYAML() (interface{}, error) {
	var n = yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
		Content: []*yaml.Node{
			{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "path",
			},
			{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: fmt.Sprintf("%s:%d:%d", m.Path, m.Line, m.Col),
			},
			{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: "text",
			},
			{
				Kind:  yaml.ScalarNode,
				Tag:   "!!str",
				Value: m.Text,
			},
		},
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
				&yaml.Node{
					Kind:  yaml.ScalarNode,
					Tag:   "!!str",
					Value: head.Value,
				},
			)
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

	n.HeadComment = strings.TrimSpace(m.Comment)

	return &n, nil
}
