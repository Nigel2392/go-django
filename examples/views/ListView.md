# ListView Example

A `ListView` is used to display a collection of objects. It typically includes pagination, filtering, and table rendering logic. `docker-mailserver-mailman` also showcases a custom `BoundListView` approach for robust vanilla lists.

## Context Parameters

The `go-django` list view automatically provides the following variables to your template:

- `Request`: The `*http.Request` object.
- `view`: The view instance itself.
- `view_paginator`: The pagination logic handler.
- `view_paginator_object`: The specific page object containing the items and current page data.
- `view_max_amount`, `view_amount_param`, `view_page_param`: Pagination configuration parameters.

## Implementation

```go
package myviews

import (
    "net/http"
    "github.com/Nigel2392/go-django/src/views/list"
    queries "github.com/Nigel2392/go-django/queries/src"
    "github.com/Nigel2392/go-django/examples/todoapp/todos"
)

var MyListView = &list.View[*todos.Todo]{
    Model:           &todos.Todo{},
    AllowedMethods:  []string{http.MethodGet},
    TemplateName:    "todos/list.html",
    BaseTemplateKey: "base",
    AmountParam:     "amount",
    PageParam:       "page",
    DefaultAmount:   10,
    MaxAmount:       100,
    QuerySet: func(r *http.Request) *queries.QuerySet[*todos.Todo] {
        return queries.GetQuerySet(&todos.Todo{})
    },
    ListColumns: []list.ListColumn[*todos.Todo]{
        list.Column[*todos.Todo]("Title", "Title"),
    },
}
```

## Minimal Vanilla Template (`todos/list.html`)

```html
<div>
    {{ $paginatorObj := .Get "view_paginator_object" }}
    {{ $pageParam := .Get "view_page_param" }}
    
    <h1>To-Do List</h1>
    <ul>
        <!-- Iterate over the items in the current page -->
        {{ range $item := $paginatorObj.Items }}
            <li>
                <a href="/todos/{{ $item.ID }}">{{ $item.Title }}</a>
                {{ if $item.Done }} (Done) {{ end }}
            </li>
        {{ else }}
            <li>No tasks available.</li>
        {{ end }}
    </ul>

    <!-- Vanilla Pagination Controls -->
    <div class="pagination">
        {{ if $paginatorObj.HasPrevious }}
            <a href="?{{ $pageParam }}={{ $paginatorObj.PreviousPageNumber }}">Previous</a>
        {{ end }}
        
        <span>Page {{ $paginatorObj.Number }} of {{ $paginatorObj.TotalPages }}</span>
        
        {{ if $paginatorObj.HasNext }}
            <a href="?{{ $pageParam }}={{ $paginatorObj.NextPageNumber }}">Next</a>
        {{ end }}
    </div>
</div>
```
