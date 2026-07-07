# RedirectView Example

A `RedirectView` is an explicit, programmable way to route an incoming request to another URL immediately.

## Context Parameters & Templates

*Note: Because this view halts the request lifecycle and dispatches an HTTP redirect status code, it does not utilize HTML templates or context parameters.*

## Implementation

```go
package myviews

import (
    "net/http"
)

type RedirectView struct {
    URL        string
    Permanent  bool
}

func (v *RedirectView) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *RedirectView) ServeGET(w http.ResponseWriter, req *http.Request) {
    statusCode := http.StatusTemporaryRedirect
    if v.Permanent {
        statusCode = http.StatusMovedPermanently
    }
    http.Redirect(w, req, v.URL, statusCode)
}

var OldPageRedirect = &RedirectView{
    URL:       "/new-page-url",
    Permanent: true,
}
```
