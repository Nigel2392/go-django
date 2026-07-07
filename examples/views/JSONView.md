# JSONView Example

To return JSON instead of rendering an HTML template, create a custom view struct overriding HTTP methods natively (e.g., `ServeGET`).

## Context Parameters & Templates

*Note: Because this view directly encodes and writes JSON payloads to the `http.ResponseWriter`, it does not utilize the standard HTML rendering pipeline nor template context parameters.*

## Implementation

```go
package myviews

import (
    "encoding/json"
    "net/http"
)

type MyJSONView struct {}

func (v *MyJSONView) ServeXXX(w http.ResponseWriter, req *http.Request) {}

func (v *MyJSONView) ServeGET(w http.ResponseWriter, req *http.Request) {
    w.Header().Set("Content-Type", "application/json")
    
    payload := map[string]interface{}{
        "status": "success",
        "data": map[string]string{
            "message": "Hello, this is a JSON response!",
        },
    }
    
    if err := json.NewEncoder(w).Encode(payload); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
    }
}

var ApiStatusView = &MyJSONView{}
```
