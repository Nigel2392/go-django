package views

import (
	"strconv"

	"github.com/Nigel2392/go-django/core/views/interfaces"
	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/response"
)

// An implementation of a generic listview.
//
// This view is used for displaying a list of items.
//
// The following template variables are set by this view:
//
//	current_page (int)
//	total_pages (int)
//	total_items (int)
//	items_per_page (int)
//	has_next_page (bool)
//	has_previous_page (bool)
//	next_page (int)
//	previous_page (int)
//	items (slice of T)
type ListView[T interfaces.Lister[T]] struct {
	BaseView[[]T]

	// Fallback number of items per page (When an error occurs for example)
	//
	// This is also the default.
	//
	// This is used when the user doesn't specify the number of items per page.
	FallbackPerPage int

	// Maximum number of items per page.
	//
	// This is used when the user specifies a number of items per page that is
	// greater than this value.
	MaxPerPage int
}

func (v *ListView[T]) ServeHTTP(r *request.Request) {

	var (
		currentPageStr string
		currentPage    int
		perPageStr     string
		perPage        int
		err            error
	)

	// Get the current page.
	currentPageStr = r.Request.URL.Query().Get("page")
	if currentPageStr == "" {
		currentPage = 1
	} else {
		currentPage, err = strconv.Atoi(currentPageStr)
		if err != nil {
			currentPage = 1
		}
	}

	// Get the number of items per page.
	perPageStr = r.Request.URL.Query().Get("items")
	if perPageStr == "" {
		perPage = v.FallbackPerPage
	} else {
		perPage, err = strconv.Atoi(perPageStr)
		if err != nil {
			perPage = v.FallbackPerPage
		}
	}

	if perPage > v.MaxPerPage {
		perPage = v.MaxPerPage
	}

	// Get the list.
	var newItem T
	items, totalcount, err := newItem.List(currentPage, perPage)

	// Get the total number of pages.
	var totalPages int
	if totalcount%int64(perPage) == 0 {
		totalPages = int(totalcount / int64(perPage))
	} else {
		totalPages = int(totalcount/int64(perPage)) + 1
	}

	v.BaseView.Action = "list"
	v.BaseView.GetQuerySet = func(r *request.Request) ([]T, error) {
		return items, nil
	}

	r.Data.Set("current_page", currentPage)
	r.Data.Set("total_pages", totalPages)
	r.Data.Set("total_items", totalcount)
	r.Data.Set("items_per_page", perPage)
	r.Data.Set("has_next_page", currentPage < totalPages)
	r.Data.Set("has_previous_page", currentPage > 1)
	r.Data.Set("next_page", currentPage+1)
	r.Data.Set("previous_page", currentPage-1)

	if v.BaseView.Get == nil {
		v.BaseView.Get = v.get
	}

	v.BaseView.Serve(r)
}

func (v *ListView[T]) get(r *request.Request, data []T) {
	r.Data.Set("items", data)
	var err = response.Render(r, v.Template)
	if err != nil {
		r.Error(500, err.Error())
		return
	}
}
