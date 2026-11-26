package translations

import (
	"errors"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// translationSourceDir keeps the scan root available when go.mod is above it.
type translationSourceDir struct {
	fs.FS
	root string
}

type goSourcePaths struct {
	fsys     fs.FS
	packages map[string]string
}

func (p *goSourcePaths) filename(name string) string {
	if pkg := p.importPath(path.Dir(name)); pkg != "" {
		return path.Join(pkg, path.Base(name))
	}
	return name
}

func (p *goSourcePaths) importPath(dir string) string {
	if pkg, ok := p.packages[dir]; ok {
		return pkg
	}

	var pkg string
	data, err := fs.ReadFile(p.fsys, path.Join(dir, "go.mod"))
	switch {
	case err == nil:
		pkg = goModulePath(data)
	case !errors.Is(err, fs.ErrNotExist):
		// Leave the path relative when module metadata is unreadable.
	case dir != ".":
		if parent := p.importPath(path.Dir(dir)); parent != "" {
			pkg = path.Join(parent, path.Base(dir))
		}
	default:
		if source, ok := p.fsys.(translationSourceDir); ok {
			pkg = enclosingGoPackage(source.root)
		}
	}
	p.packages[dir] = pkg
	return pkg
}

func enclosingGoPackage(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	for child := abs; ; {
		parent := filepath.Dir(child)
		if parent == child {
			return ""
		}
		data, err := os.ReadFile(filepath.Join(parent, "go.mod"))
		if errors.Is(err, fs.ErrNotExist) {
			child = parent
			continue
		}
		if err != nil {
			return ""
		}
		module := goModulePath(data)
		relative, err := filepath.Rel(parent, abs)
		if err != nil || module == "" {
			return ""
		}
		return path.Join(module, filepath.ToSlash(relative))
	}
}

func goModulePath(data []byte) string {
	for line := range strings.SplitSeq(string(data), "\n") {
		line, _, _ = strings.Cut(line, "//")
		fields := strings.Fields(line)
		if len(fields) == 2 && fields[0] == "module" {
			name := fields[1]
			if strings.HasPrefix(name, `"`) || strings.HasPrefix(name, "`") {
				name, _ = strconv.Unquote(name)
			}
			return name
		}
	}
	return ""
}
