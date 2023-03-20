package fs

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Nigel2392/go-django/core/httputils"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/templates"
)

// Return the default static URL for the path.
func (fm *Manager) AsStaticURL(path string) string {
	return templates.NicePath(false, fm.FS_STATIC_URL, path)
}

// Return the default media URL for the path.
func (fm *Manager) AsMediaURL(path string) string {
	return templates.NicePath(false, fm.FS_MEDIA_URL, path)
}

func (fm *Manager) PathToURL(path string) (string, error) {
	if strings.HasPrefix(path, fm.FS_STATIC_ROOT) {
		return fm.StaticPathToURL(path)
	} else if strings.HasPrefix(path, fm.FS_MEDIA_ROOT) {
		return fm.MediaPathToURL(path)
	}
	return "", errors.New("path not in static or media root")
}

func (fm *Manager) StaticPathToURL(path string) (string, error) {
	if !strings.HasPrefix(path, fm.FS_STATIC_ROOT) {
		return "", errors.New("path not in static root")
	}
	return templates.NicePath(false,
		fm.FS_STATIC_URL, strings.TrimPrefix(path, fm.FS_STATIC_ROOT)), nil
}

func (fm *Manager) MediaPathToURL(path string) (string, error) {
	if !strings.HasPrefix(path, fm.FS_MEDIA_ROOT) {
		return "", errors.New("path not in media root")
	}
	return templates.NicePath(false,
		fm.FS_MEDIA_URL, strings.TrimPrefix(path, fm.FS_MEDIA_ROOT)), nil
}

// Return the default registrars for the routes.
// If the registrars are already set, they will be returned.
func (fm *Manager) Registrars() (router.Registrar, router.Registrar) {
	var StaticRegistrar router.Registrar
	var MediaRegistrar router.Registrar

	if fm.staticRegistrar != nil || fm.mediaRegistrar != nil {
		return fm.staticRegistrar, fm.mediaRegistrar
	}

	// Register staticfiles
	if fm.fs_STATICFILES != nil {
		var static_url = templates.NicePath(false, "/", fm.FS_STATIC_URL, "/<<any>>")
		StaticRegistrar = router.Group(
			static_url, "static",
		)
		var r = StaticRegistrar.(*router.Route)
		r.Method = router.GET
		r.HandlerFunc = router.FromHTTPHandler(http.StripPrefix(
			httputils.WrapSlash(fm.FS_STATIC_URL),
			http.FileServer(
				http.FS(fm.fs_STATICFILES),
			),
		)).ServeHTTP
	}
	// Register mediafiles
	if fm.fs_MEDIAFILES != nil {
		var media_url = templates.NicePath(false, "/", fm.FS_MEDIA_URL, "/<<any>>")
		MediaRegistrar = router.Group(
			media_url, "media",
		)
		var r = MediaRegistrar.(*router.Route)
		r.Method = router.GET
		r.HandlerFunc = router.FromHTTPHandler(http.StripPrefix(
			httputils.WrapSlash(fm.FS_MEDIA_URL),
			http.FileServer(
				http.FS(fm.fs_MEDIAFILES),
			),
		)).ServeHTTP
	}

	fm.staticRegistrar = StaticRegistrar
	fm.mediaRegistrar = MediaRegistrar

	return StaticRegistrar, MediaRegistrar
}
