# Filesystem

The filesystem is a tree of directories and files.

To easily work with files within the framework and provide a uniform approach to handling files and directories, the framework provides a filesystem abstraction.

All filesystems ultimately do adhere to the `fs.FS` interface from the `io/fs` package.

## Structures

### `MultiFS`

MultiFS is a filesystem that combines multiple filesystems.

It can be used to combine multiple filesystems into a single filesystem.

When opening a file, it will try to open the file in each filesystem in the order they were added.

It is best (and automatically) used with the `MatchFS` filesystem to restrict access to files in the filesystems.

This allows for faster skipping of filesystems that do not contain the file.

It generally works like any other filesystem and adheres to the `fs.FS` interface, but with the added benefit of being able to combine multiple filesystems.

A method for adding filesystems is provided:

```go
var myFs = filesystem.NewMultiFS()
myFs.AddFS(
    // Add the filesystem to the MultiFS
    os.DirFS("/path/to/files"),

    // Add optional matcher-funcs
    // 
    // If any are provided (non-nil), the filesystem is automatically wrapped in a MatchFS
    nil,
)
```

### `MatchFS`

MatchFS is a filesystem that only allows opening files that match a given matcher-func.  

It can be used to restrict access to files in a filesystem.  

The matcher-func is called with the path of the file that is being opened.

We provide a few default matchers:

* `MatchNever`  
  MatchNever returns a matcher-func that never matches any file.
* `MatchAnd(matches ...func(filepath string) bool)`  
  MatchAnd returns a matcher-func that matches a file if all the given matchers match.  
  It can be passed any number of matchers.  
  This allows for more complex logic when matching paths.
* `MatchOr(matches ...func(filepath string) bool)`  
  MatchOr returns a matcher-func that matches a file if any of the given matchers match.  
  It can be passed any number of matchers.  
  This allows for more complex logic when matching paths.
* `MatchPrefix(prefix string)`  
  MatchPrefix returns a matcher-func that matches a file if the given prefix matches the file path.  
  The prefix is normalized to use "/" as the path separator.  
  If the prefix is not empty and does not end with a "." or "/", it is appended with a "/".  
  When matching, the file path is compared to the prefix, the provided path either has to be the prefix or start with the prefix.
* `MatchSuffix(suffix string)`  
  MatchSuffix returns a matcher-func that matches a file if the given suffix matches the file path.  
  The suffix is normalized to use "/" as the path separator.  
  If the suffix is not empty and does not start with a "." or "/", it is prepended with a "/".  
  When matching, the file path is compared to the suffix, the provided path either has to be the suffix or end with the suffix.
* `MatchExt(extension string)`  
  MatchExt returns a matcher-func that matches a file if the given extension matches the file path.  
  The extension passed to this function is normalized to start with a ".".  
  When matching, the extension is retrieved from the file path with filepath.Ext and compared to the provided extension.

#### Example

Using a `MatchFS` is relatively simple.

First we will obtain a new `fs.FS` object, then we will wrap it in a `MatchFS` object.

For this example, we will imagine to only need ".html" and ".txt" files if the directory starts with "templates".

Otherwise if the directory starts with "static", we will only need ".css" and ".js" files.

```go
var myFs = os.DirFS("/path/to/files")

var myMatchFs = filesystem.NewMatchFS(
    myFs, filesystem.MatchOr(
        // Match files in the "templates" directory with the extensions ".html" and ".txt"
        filesystem.MatchAnd(
            filesystem.MatchPrefix("templates/"),
            filesystem.MatchExt(".html"),
            filesystem.MatchExt(".txt"),
        ),

        // Match files in the "static" directory with the extensions ".css" and ".js"
        filesystem.MatchAnd(
            filesystem.MatchPrefix("static/"),
            filesystem.MatchExt(".css"),
            filesystem.MatchExt(".js"),
        ),
    ),
)
```