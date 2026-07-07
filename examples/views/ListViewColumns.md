# ListView With Columns Example

The `go-django` `list.View` natively supports automated table generation using `ListColumns`. When configured, the view automatically passes a `view_list` object to the template context, which can instantly render an entire HTML table with proper headers and rows based on your columns. This mirrors the behavior seen in the `docker-mailserver-mailman` project.

## Context Parameters

The `ListView` provides the following specialized variables for auto-rendering:

- `view_list`: The list component responsible for safely rendering the entire HTML table via `.Render`.
- `view_paginator_object`: The standard pagination object containing current page context.
- `view_page_param`: The URL parameter representing the page query.

## Implementation

```go
package myviews

import (
    "fmt"
    "html/template"
    "net/http"
    "github.com/Nigel2392/go-django/src/views/list"
    queries "github.com/Nigel2392/go-django/queries/src"
    "github.com/Nigel2392/go-django/src/core/attrs"
    "github.com/Nigel2392/go-django/src/core/trans"
    "github.com/Nigel2392/go-django/examples/todoapp/todos"
)

var MyListViewWithColumns = &list.View[*todos.Todo]{
    Model:           &todos.Todo{},
    AllowedMethods:  []string{http.MethodGet},
    TemplateName:    "todos/list_columns.html",
    BaseTemplateKey: "base",
    AmountParam:     "limit",
    PageParam:       "page",
    DefaultAmount:   10,
    MaxAmount:       100,
    QuerySet: func(r *http.Request) *queries.QuerySet[*todos.Todo] {
        return queries.GetQuerySet(&todos.Todo{})
    },
    ListColumns: []list.ListColumn[*todos.Todo]{
        // Standard text column pulling from the "Title" struct field
        list.Column[*todos.Todo](trans.S("Title"), "Title"),
        
        // Specialized boolean indicator column for the "Done" struct field
        list.BooleanFieldColumn[*todos.Todo](trans.S("Done"), "Done"),
        
        // Function column mapping dynamic logic
        list.FuncColumn(trans.S("Description Prefix"), func(r *http.Request, defs attrs.Definitions, row *todos.Todo) interface{} {
            if len(row.Description) > 10 {
                return row.Description[:10] + "..."
            }
            return row.Description
        }),
        
        // HTML column useful for action buttons
        list.HTMLColumn(trans.S("Actions"), func(r *http.Request, defs attrs.Definitions, row *todos.Todo) template.HTML {
            return template.HTML(fmt.Sprintf(`<a href="/todos/%d/edit">Edit</a>`, row.ID))
        }),
    },
}
```

## Minimal Vanilla Template (`todos/list_columns.html`)

```html
<div>
    <h1>Advanced To-Do List</h1>
    
    <div class="list-table-container">
        {{ $list := .Get "view_list" }}
        
        <!-- The .Render method generates the HTML table based on the configured ListColumns -->
        {{ safe $list.Render }}
    </div>

    <!-- Render vanilla pagination controls underneath -->
    {{ $paginatorObj := .Get "view_paginator_object" }}
    {{ $pageParam := .Get "view_page_param" }}
    
    {{ $paginatorObj.HTML $pageParam 5 .Request.URL.Query }}
</div>
```
