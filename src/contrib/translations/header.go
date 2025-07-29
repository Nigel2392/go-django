package translations

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/Nigel2392/go-django/src/core/trans"
	"github.com/casbin/govaluate"
	"github.com/elliotchance/orderedmap/v2"
	"gopkg.in/yaml.v3"
)

type translationHeader struct {
	hdr      *FileTranslationsHeader
	exprs    map[trans.Locale]*govaluate.EvaluableExpression
	wasSetup bool
}

func newTranslationHeader(header *FileTranslationsHeader) *translationHeader {
	var t = &translationHeader{
		hdr:   header,
		exprs: make(map[trans.Locale]*govaluate.EvaluableExpression),
	}

	return t
}

func (t *translationHeader) setup() {
	if t.wasSetup {
		return
	}

	t.wasSetup = true

	if t.hdr != nil && t.hdr.Locales != nil && t.hdr.Locales.Len() > 0 {
		for head := t.hdr.Locales.Front(); head != nil; head = head.Next() {
			var expr, err = govaluate.NewEvaluableExpression(head.Value.PluralRule)
			if err != nil {
				panic(fmt.Errorf("failed to parse plural rule for locale %s: %w", head.Key, err))
			}
			t.exprs[head.Key] = expr
		}
	}
}

func (t *translationHeader) pluralIndex(locale trans.Locale, count int) (int, error) {
	t.setup()

	if expr, ok := t.exprs[locale]; ok {
		result, err := expr.Evaluate(map[string]any{"n": count})
		if err != nil {
			return 0, fmt.Errorf("failed to evaluate plural rule for locale %s: %w", locale, err)
		}

		switch v := result.(type) {
		case bool:
			if v {
				return 1, nil // Plural form
			}
			return 0, nil // Singular form
		case int, int8, int16, int32, int64:
			var rv = reflect.ValueOf(v)
			return int(rv.Int()), nil // Return the integer value as plural form index
		case float32, float64:
			var rv = reflect.ValueOf(v)
			return int(rv.Float()), nil // Return the float value as plural form index
		case string:
			value, err := strconv.Atoi(v)
			if err != nil {
				return 0, err
			}
			return value, nil
		default:
			return 0, fmt.Errorf("unexpected result type %T for plural rule evaluation", v)
		}
	}

	logger.Debugf(
		"Plural rule for locale '%s' not found, using default rule (n != 1)", locale,
	)

	if count > 1 {
		return 1, nil
	}
	return 0, nil
}

type TranslationHeaderLocale struct {
	NumPluralForms int    `yaml:"nplural"` // Number of plural forms, e.g. 2 for English (singular, plural)
	PluralRule     string `yaml:"rule"`    // (n != 1), (n % 10 == 1 && n % 100 != 11), etc.
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
		Style:       yaml.FoldedStyle,
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
