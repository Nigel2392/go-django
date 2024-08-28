# Staticfiles

Staticfiles are files that are served directly to the client.

This can be anything from images, CSS, JavaScript, or other static files.

Staticfiles will run through the middleware chain, so you can add custom middleware to serve static files.

All staticfiles are served from the `static/` URL prefix by default.

This can be changed by setting the [`APPVAR_STATIC_URL` setting](./configuring.md#pre-defined-settings).

## Configuration

To configure staticfiles and learn more about matches, see the [filesystem documentation](./filesystem.md#matchfs).

Staticfiles are configured using the global package- level functions:

### `AddFS(filesys fs.FS, matches func(path string) bool)`

Add a filesystem to the list of filesystems that will be used to serve static files.

The `matches` function is used to determine if a file should be served from this filesystem.

This allows for more fine-grained but performant matching of file paths in multiple filesystems.

### `Collect(fn func(path string, f fs.File) error) error`

Collect all files from all filesystems and call the `fn` function with the path and file.

This can be used to pre-load all files into memory, or to do other operations on the files, like hashing, minification, or other transformations.

None of this functionality is built-in, but can be added by the user.

### `Open(name string) (fs.File, error)`

Open a file by name.

This will return the first file that matches the name in any of the filesystems.

## Example

In this example, we add an embedded filesystem to the staticfiles, then we will pretend to collect all the files.

You need to implement your own logic for this.

```go
package main

import (
    "fmt"
    "embed"
    "io/fs"
    "github.com/Nigel2392/django"
    "github.com/Nigel2392/django/core/filesystem/staticfiles"
)

//go:embed assets/**
var staticFS embed.FS

func main() {
    var app = django.App(
        // ...
    )

    staticfiles.AddFS(staticFS, nil)


    var err = app.Initialize()
    if err != nil {
        panic(err)
    }

    err = staticfiles.Collect(func(path string, f fs.File) error {
        var stat, err = f.Stat()
        if err != nil {
            return err
        }
        fmt.Println("Collected", path, stat.Size())
        return nil
    })

    if err != nil {
        panic(err)
    }
}
```
