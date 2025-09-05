package icons

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-django/src/components"
	"github.com/Nigel2392/go-django/src/core/assert"
	"github.com/Nigel2392/go-django/src/core/errs"
	"github.com/Nigel2392/go-django/src/core/filesystem"
	"github.com/Nigel2392/go-django/src/core/logger"
	"github.com/elliotchance/orderedmap/v2"
)

var (
	registry = &iconRegistry{
		icons: orderedmap.NewOrderedMap[string, IconInterface](),
		fs:    filesystem.NewMultiFS(),
	}
	iconIDPattern      = regexp.MustCompile(`id=["']icon-([a-z0-9-]+)["']`)
	iconCommentPattern = regexp.MustCompile(`<!--!(.*?)-->`)
)

type IconInterface interface {
	Name() string
	Folder() string
	Fullpath() string
	Source() string
	HTML() template.HTML
}

type (
	icon struct {
		name     string
		folder   string
		fullpath string
		source   string
		loaded   template.HTML
	}
	iconRegistry struct {
		icons *orderedmap.OrderedMap[string, IconInterface]
		fs    *filesystem.MultiFS
	}
)

func (i *icon) Name() string {
	return i.name
}

func (i *icon) Folder() string {
	return i.folder
}

func (i *icon) Fullpath() string {
	return i.fullpath
}

func (i *icon) Source() string {
	return i.source
}

func (i *icon) HTML() template.HTML {
	return i.loaded
}

func Component(name string) func(c context.Context, w io.Writer) error {
	var icon, ok = registry.icons.Get(name)
	assert.True(ok, "icons: %s is not a registered icon", name)

	return func(c context.Context, w io.Writer) error {
		_, err := w.Write([]byte(icon.HTML()))
		if err != nil {
			return err
		}
		return nil
	}
}

func Register(fs fs.FS, path ...string) error {
	registry.fs.Add(fs, nil)
	return registry.add(path...)
}

func Icons() []IconInterface {
	var icons = make([]IconInterface, 0, registry.icons.Len())
	for front := registry.icons.Front(); front != nil; front = front.Next() {
		icons = append(icons, front.Value)
	}
	return icons
}

func Load(name string) template.HTML {
	var filename = filepath.Base(name)
	if strings.Contains(filename, ".") {
		logger.Fatalf(500, "icons: %s is not a valid icon name", name)
		return ""
	}

	var icon, ok = registry.icons.Get(filename)
	if !ok {
		logger.Fatalf(500, "icons: %s is not a registered icon", name)
		return ""
	}

	return icon.HTML()
}

func Icon(name string, attrs map[string]string) components.Component {
	var attrString = new(strings.Builder)
	var className string
	for k, v := range attrs {
		if k == "class" {
			className = v
			continue
		}

		fmt.Fprintf(attrString,
			` %s="%s"`,
			strconv.Quote(k),
			template.HTMLEscapeString(v),
		)
	}

	if className != "" {
		className = fmt.Sprintf("icon %s %s", className, name)
	} else {
		className = fmt.Sprintf("icon %s", name)
	}

	return components.FuncComponent(func(ctx context.Context, w io.Writer) error {
		_, err := fmt.Fprintf(w,
			`<svg class="%s"%s><use href="#%s"></use></svg>`,
			className, attrString.String(), name,
		)
		return err
	})
}

func DefsHTML() template.HTML {
	var replacer = strings.NewReplacer(
		"xmlns=\"http://www.w3.org/2000/svg\"", "",
		"<svg", "<symbol",
		"</svg>", "</symbol>",
	)

	var icons = Icons()
	var htmlString = make([]string, 0, len(icons))
	for _, icon := range icons {
		htmlString = append(htmlString, replacer.Replace(
			string(icon.HTML()),
		))
	}
	return template.HTML(
		fmt.Sprintf(
			`<svg xmlns="http://www.w3.org/2000/svg" style="display: none;"><defs>%s</defs></svg>`,
			strings.Join(htmlString, ""),
		),
	)
}

func (r *iconRegistry) add(path ...string) error {
	if len(path) < 1 {
		return errs.Error("icons: path is required")
	}

	var icons = make([]*icon, 0, len(path))
	for _, p := range path {
		var (
			name   = filepath.Base(p)
			folder = filepath.Dir(p)
		)

		var b = new(bytes.Buffer)
		var f, err = r.fs.Open(p)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err = io.Copy(b, f); err != nil && err != io.EOF {
			return err
		}

		var s = b.String()
		var matches = iconIDPattern.FindAllStringSubmatch(s, -1)
		if len(matches) < 1 {
			return fmt.Errorf("icons: no icon ID found, please provide an ID (%s) %s", p, s)
		}

		var id = matches[0][1]
		if id != name[:strings.LastIndex(name, ".")] {
			return fmt.Errorf("icons: icon ID (%s) does not match filename (%s), id 'icon-my-example' should match file 'my-example.svg'", id, name)
		}

		var sourceMatch = iconCommentPattern.FindAllStringSubmatch(s, -1)
		var source string
		if len(sourceMatch) > 0 {
			source = sourceMatch[0][1]
		}

		icons = append(icons, &icon{
			name:     name,
			folder:   folder,
			fullpath: p,
			source:   source,
			loaded:   template.HTML(s),
		})
	}

	slices.SortStableFunc(icons, func(a, b *icon) int {
		return strings.Compare(a.name, b.name)
	})

	for _, i := range icons {
		r.icons.Set(i.name, i)
	}

	return nil
}
