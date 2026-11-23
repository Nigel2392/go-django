# Filesystem and Media Files

The `filesystem` package (`github.com/Nigel2392/go-django/src/core/filesystem`) provides composable `fs.FS` wrappers (`MultiFS`, `MatchFS`, `CacheFS`, `Sub`) used across templates and static files. Its `mediafiles` subpackages handle user-uploaded media storage and HTTP serving.

## Core Helpers

### `FileHeader` and `Sub`

```go
type FileHeader interface {
    Name() string
    Size() int64
    Open() (multipart.File, error)
}

func Sub(fileSys fs.FS, path string) fs.FS
```

`filesystem.Sub` wraps `fs.Sub` from the standard library and panics via `assert.Fail` if `path` is invalid:

```go
myFS := os.DirFS("/path/to/files")
mySubFS := filesystem.Sub(myFS, "subdir")
```

---

## Filesystem Wrappers

### `MultiFS`

`MultiFS` combines multiple `fs.FS` instances into a single `fs.FS`:

```go
func NewMultiFS(fileSystems ...fs.FS) *MultiFS
func (m *MultiFS) Add(fs fs.FS, matches func(filepath string) bool)
func (m *MultiFS) Open(name string) (fs.File, error)
func (m *MultiFS) ForceOpen(name string) (fs.File, error)
func (m *MultiFS) ReadDir(name string) ([]fs.DirEntry, error)
func (m *MultiFS) FS() []fs.FS
```

- `Add`: Appends `fs` to the pool. If `matches != nil`, `fs` is wrapped in a `*MatchFS`.
- **Resolution order:** `Open`, `ForceOpen`, and `ReadDir` iterate filesystems in **reverse order** (from the last-added filesystem back to the first). This means **later filesystems override earlier ones** for matching file paths. `Open` also caches the winning filesystem index per path.
- `ForceOpen`: Walks filesystems in reverse order and bypasses `MatchFS` matcher restrictions when opening `name`.
- `FS`: Returns the slice of underlying filesystems.

```go
myFS := filesystem.NewMultiFS()
myFS.Add(os.DirFS("/path/to/defaults"), nil)
myFS.Add(
    os.DirFS("/path/to/overrides"),
    filesystem.MatchPrefix("templates/"),
)
```

### `MatchFS`

`MatchFS` restricts `Open` calls on an `fs.FS` to paths approved by a matcher predicate:

```go
func NewMatchFS(fs fs.FS, matches func(filepath string) bool) *MatchFS
func (m *MatchFS) Open(name string) (fs.File, error)
func (m *MatchFS) ForceOpen(name string) (fs.File, error)
func (m *MatchFS) ReadDir(name string) ([]fs.DirEntry, error)
func (m *MatchFS) FS() fs.FS
```

Built-in matchers:

- `MatchNever(string) bool`: Never matches any path.
- `MatchAnd(matches ...func(filepath string) bool)`: Matches only if **all** matchers return `true`.
- `MatchOr(matches ...func(filepath string) bool)`: Matches if **any** matcher returns `true`.
- `MatchPrefix(prefix string)`: Normalizes `prefix` with `/` separators and appends `/` if non-empty and not ending with `.` or `/`. Matches paths equal to or starting with `prefix`.
- `MatchSuffix(suffix string)`: Normalizes `suffix` with `/` separators and checks `path == suffix || strings.HasSuffix(path, suffix)`.
- `MatchExt(extension string)`: Normalizes `extension` to start with `"."` and compares against `filepath.Ext(path)`.

> [!WARNING]
> **Caveat on `MatchSuffix`:** If `suffix` is non-empty and does not begin with `"."` or `"/"`, `MatchSuffix` appends a trailing `"/"` (`suffix += "/"`). Use `MatchExt` for file extensions, or pass a leading `"/"` or `"."` (for example `MatchSuffix("/index.html")` or `MatchSuffix(".tmpl")`).

> [!NOTE]
> When matching multiple extensions under a directory prefix, combine extensions with `MatchOr` inside `MatchAnd`—passing two `MatchExt` calls directly to `MatchAnd` requires a single file to have both extensions simultaneously and will match nothing.

```go
baseFS := os.DirFS("/path/to/files")

matchedFS := filesystem.NewMatchFS(
    baseFS,
    filesystem.MatchOr(
        // Match .html or .txt files inside "templates/"
        filesystem.MatchAnd(
            filesystem.MatchPrefix("templates/"),
            filesystem.MatchOr(
                filesystem.MatchExt(".html"),
                filesystem.MatchExt(".txt"),
            ),
        ),
        // Match .css or .js files inside "static/"
        filesystem.MatchAnd(
            filesystem.MatchPrefix("static/"),
            filesystem.MatchOr(
                filesystem.MatchExt(".css"),
                filesystem.MatchExt(".js"),
            ),
        ),
    ),
)
```

### `CacheFS[T]`

`CacheFS[T fs.FS]` caches file bytes and `fs.FileInfo` in memory on first `Open`:

```go
func NewCacheFS[T fs.FS](fsys T) *CacheFS[T]
func (c *CacheFS[T]) Open(path string) (fs.File, error)
func (c *CacheFS[T]) ReadDir(name string) ([]fs.DirEntry, error)
func (c *CacheFS[T]) Changed(paths ...string)
```

Call `c.Changed(paths...)` when underlying files change to evict specific cached paths, or call `c.Changed()` with no arguments to clear the entire cache.

---

## Media Files

The `mediafiles` package (`github.com/Nigel2392/go-django/src/core/filesystem/mediafiles`) defines a backend registry and storage interfaces for uploaded or dynamically generated files.

### Sentinel Errors

- `mediafiles.ErrNotImplemented`: Operation unsupported by the storage backend.
- `mediafiles.ErrNotFound`: Alias for `os.ErrNotExist`.
- `mediafiles.ErrExists`: Alias for `os.ErrExist`.
- `mediafiles.ErrSuspiciousOperation`: Alias for `os.ErrInvalid`.

### Core Media Interfaces

```go
type FileHeader interface {
    Name() string
    Path() string
    Size() int64
    TimeAccessed() (t time.Time, err error)
    TimeCreated() (t time.Time, err error)
    TimeModified() (t time.Time, err error)
}

type File interface {
    io.Reader
    io.Closer
    Stat() (FileHeader, error)
}

type StoredObject interface {
    Name() string
    Path() string
    Open() (File, error)
}

type Backend interface {
    Delete(path string) error
    Exists(path string) (bool, error)
    GetAlternateName(fileRoot, fileExt string) string
    GetAvailableName(path string, retries, maxLength int) (string, error)
    Stat(path string) (FileHeader, error)
    ListDir(path string) ([]string, error)
    Open(path string) (StoredObject, error)
    Save(path string, file io.Reader, maxLength ...int) (string, error)
}
```

### `SimpleStoredObject`

`mediafiles.SimpleStoredObject` implements `StoredObject`, `sql.Scanner`, `driver.Valuer`, `json.Marshaler`, `json.Unmarshaler`, and `DBType() dbtype.Type` (`dbtype.String`). When scanned from a database or unmarshaled from JSON, it looks up `mediafiles.GetDefault()` to populate `OpenFn`, so a default backend must already be set before scanning or unmarshaling.

### Backend Registry and Package Helpers

- `mediafiles.RegisterBackend(name string, backend Backend)`: Registers a named backend.
- `mediafiles.RetrieveBackend(name string) (Backend, bool)`: Retrieves a registered backend (passing `""` returns the current default backend).
- `mediafiles.MapRegistry() map[string]Backend`: Returns the registered backend map (including `"default"` if a default backend is set).
- `mediafiles.GetDefault() Backend` / `mediafiles.SetDefault(backend string) Backend`: Gets or sets the default backend (`SetDefault` panics if `backend` is not registered).
- Top-level helpers forwarding to `GetDefault()`: `Delete`, `Exists`, `GetAlternateName`, `GetAvailableName`, `Stat`, `ListDir`, `Open`, and `Save`.

### Built-in Media Backends

- **Local filesystem (`github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs`):**
  Construct with `fs.NewBackend(baseDir string, retries int)` to store files on disk under `baseDir`. Importing the package registers `"filesystem"` with an empty `BaseDir` and sets it as the default backend.
- **In-memory (`github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/memory`):**
  Construct with `memory.NewBackend(retries int)`.

> [!WARNING]
> **Caveat on `mediafiles/memory`:** Its `init()` registers `&memory.Backend{}` with a `nil` `files` map and `nil` `mu` mutex and sets it as the default backend. Always replace it with `memory.NewBackend(...)` before saving or reading files, otherwise operations will panic:
>
> ```go
> mediafiles.RegisterBackend("memory", memory.NewBackend(5))
> mediafiles.SetDefault("memory")
> ```

### Serving Media Files over HTTP

The `mhttp` subpackage (`github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/mhttp`) provides `mhttp.Serve(engine string) http.Handler`.

During `app.Initialize()`, if at least one backend is registered (`len(mediafiles.MapRegistry()) > 0`) and `django.APPVAR_MEDIA_URL` is non-empty (default `"/media/"`), Go-Django automatically mounts `mhttp.Serve` at `APPVAR_MEDIA_URL` using the backend named in `django.APPVAR_MEDIA_BACKEND` (default `""`, meaning the default backend) and marks the route with `django.MarkStaticRouteMiddleware`.

```go
package main

import (
    "github.com/Nigel2392/go-django/src"
    "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles"
    mediafs "github.com/Nigel2392/go-django/src/core/filesystem/mediafiles/fs"
)

func init() {
    mediafiles.RegisterBackend("uploads", mediafs.NewBackend("./.media", 5))
    mediafiles.SetDefault("uploads")
}

// Configure in django.Configure(...)
// django.APPVAR_MEDIA_URL:     "/media/",
// django.APPVAR_MEDIA_BACKEND: "uploads",
```

Generate media URLs in Go with `django.Media(path)` and in templates with `{{ media "uploads/avatar.png" }}`.
