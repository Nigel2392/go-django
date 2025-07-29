package translations

import (
	"fmt"
	"time"

	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

type TranslationHeaderLocale struct {
	NumPluralForms int    `yaml:"num_plural_forms"` // Number of plural forms, e.g. 2 for English (singular, plural)
	PluralRule     string `yaml:"plural_rule"`      // (n != 1), (n % 10 == 1 && n % 100 != 11), etc.
}

type FileTranslationsHeader struct {
	Comment     string                                                  //  `yaml:"comment,omitempty"`
	Generator   string                                                  //  `yaml:"generator"`
	Created     time.Time                                               //  `yaml:"created"`
	Translators []string                                                //  `yaml:"translators"`
	Locales     *orderedmap.OrderedMap[string, TranslationHeaderLocale] //  `yaml:"locales"`
	Data        map[string]any                                          //  `yaml:"data,omitempty"` // Additional data, e.g. version, license, etc.
}

func (h *FileTranslationsHeader) UnmarshalYAML(node *yaml.Node) error {

	h.Locales = orderedmap.NewOrderedMap[string, TranslationHeaderLocale]()

	if h.Data == nil {
		h.Data = make(map[string]any)
	}

	for i, item := range node.Content {
		if i%2 != 0 {
			continue // Skip value nodes
		}

		if len(node.Content) <= i+1 {
			return fmt.Errorf("expected value node after key %s, but found none", item.Value)
		}

		switch item.Value {
		case "comment":
			h.Comment = node.Content[i+1].Value

		case "generator":
			h.Generator = node.Content[i+1].Value

		case "created":
			created, err := time.Parse(time.RFC3339, node.Content[i+1].Value)
			if err != nil {
				return fmt.Errorf("failed to parse created time: %w", err)
			}
			h.Created = created

		case "translators":
			var translators []string
			if err := node.Content[i+1].Decode(&translators); err != nil {
				return fmt.Errorf("failed to decode translators: %w", err)
			}
			h.Translators = translators

		case "locales":
			if node.Content[i+1].Kind != yaml.MappingNode {
				return fmt.Errorf(
					"expected mapping node (%d) for locales, got %d",
					yaml.MappingNode, node.Content[i+1].Kind,
				)
			}

			for j := 0; j < len(node.Content[i+1].Content); j += 2 {
				key := node.Content[i+1].Content[j].Value
				var value TranslationHeaderLocale
				if err := node.Content[i+1].Content[j+1].Decode(&value); err != nil {
					return fmt.Errorf("failed to decode locale %s: %w", key, err)
				}
				h.Locales.Set(key, value)
			}

		default:
			var value any
			if err := node.Content[i+1].Decode(&value); err != nil {
				return fmt.Errorf("failed to decode data for key %s: %w", item.Value, err)
			}
			h.Data[item.Value] = value
		}
	}

	return nil
}

func (h *FileTranslationsHeader) MarshalYAML() (any, error) {
	var root = &yaml.Node{
		Kind:        yaml.MappingNode,
		Tag:         "!!map",
		HeadComment: h.Comment,
	}

	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "generator"},
		&yaml.Node{
			Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Tag: "!!str", Value: h.Generator,
		},
	)

	root.Content = append(root.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "created"},
		&yaml.Node{
			Kind: yaml.ScalarNode, Style: yaml.DoubleQuotedStyle, Tag: "!!str", Value: h.Created.Format(time.RFC3339),
		},
	)

	if len(h.Translators) > 0 {
		var translatorsNode = &yaml.Node{
			Kind:  yaml.SequenceNode,
			Tag:   "!!seq",
			Style: yaml.FoldedStyle,
		}

		for _, translator := range h.Translators {
			translatorsNode.Content = append(translatorsNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: translator},
			)
		}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "translators"},
			translatorsNode,
		)
	}

	var localesNode = &yaml.Node{
		Kind:  yaml.ScalarNode,
		Tag:   "!!str",
		Value: "locales",
	}

	var localesNodeMap = &yaml.Node{
		Kind:  yaml.MappingNode,
		Tag:   "!!map",
		Style: yaml.FoldedStyle,
	}

	root.Content = append(root.Content,
		localesNode,
		localesNodeMap,
	)

	if h.Locales != nil && h.Locales.Len() > 0 {
		for head := h.Locales.Front(); head != nil; head = head.Next() {
			localesNodeMap.Content = append(localesNodeMap.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: head.Key},
				&yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "num_plural_forms"},
					{Kind: yaml.ScalarNode, Tag: "!!int", Value: fmt.Sprintf("%d", head.Value.NumPluralForms)},

					{Kind: yaml.ScalarNode, Tag: "!!str", Value: "plural_rule"},
					{Kind: yaml.ScalarNode, Tag: "!!str", Value: head.Value.PluralRule},
				}},
			)
		}
	}

	for key, value := range h.Data {
		var node = &yaml.Node{}
		if err := node.Encode(value); err != nil {
			return nil, fmt.Errorf("failed to encode data for key %s: %w", key, err)
		}

		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key},
			node,
		)
	}

	return root, nil
}
