package httputils

import (
	"reflect"
	"regexp"
	"strings"

	"github.com/Nigel2392/router/v3/templates"
)

// NicePath formats a path nicely
func NicePath(forceSuffixSlash bool, p ...string) string {
	return templates.NicePath(forceSuffixSlash, p...)
}

// FilenameFromPath gets the filename from a path
func FilenameFromPath(p string) string {
	return templates.FilenameFromPath(p)
}

// DirFromPath gets the directory from a path
func DirFromPath(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	var parts = strings.Split(p, "/")
	parts = parts[:len(parts)-1]
	var pathParts []string
	for i, part := range parts {
		if part == "" {
			if i == 0 {
				pathParts = append(pathParts, "/")
			}
			continue
		}
		pathParts = append(pathParts, part)
	}
	var path = strings.Join(pathParts, "/")
	if path == "" {
		path = "/"
	}
	return path
}

// NameFromPath gets the filename from a path and removes the extension
func NameFromPath(p string) string {
	var name = FilenameFromPath(p)
	if strings.Contains(name, ".") {
		name = strings.Split(name, ".")[0]
	}
	return name
}

// WrapSlash wraps a path with a slash if it doesn't have one
func WrapSlash(p string) string {
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	if !strings.HasSuffix(p, "/") {
		p = p + "/"
	}
	return p
}

// Format the package path nicely
// For use in DB tables
func GetPkgPath(s any) string {
	var typeOf = reflect.TypeOf(s)
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	var pkgPath = typeOf.PkgPath()
	if pkgPath == "" {
		return "main"
	}
	var parts = strings.Split(pkgPath, "/")
	return strings.ReplaceAll(parts[len(parts)-1], ".", "_")
}

// SimpleSlugify is a simple slugify function
func SimpleSlugify(s string) string {
	var re = regexp.MustCompile("[^a-zA-Z0-9]+")
	return strings.ToLower(re.ReplaceAllString(s, "-"))
}

// Cut the front of a path, and add "..." if it was cut.
func CutFrontPath(s string, length int) string {
	return CutStart(s, length, "/", true)
}

// Cut the string if it is longer than the specified length, and add "..." if it was cut.
func CutStart(s string, length int, delim string, prefixIfCut bool) string {
	if len(s) > length {
		var cut = len(s) - length
		s = s[cut:]
		var parts = strings.Split(s, delim)
		if len(parts) > 1 {
			if prefixIfCut {
				return "..." + delim + strings.Join(parts[1:], delim)
			}
			return "..." + strings.Join(parts[1:], delim)
		}
		return "..." + s
	}
	return s
}
