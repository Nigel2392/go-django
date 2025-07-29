package todos

import (
	"encoding/json"
	"errors"
	"net/http"

	queries "github.com/Nigel2392/go-django/queries/src"
	"github.com/Nigel2392/go-django/src/core/ctx"
	"github.com/Nigel2392/go-django/src/core/filesystem/tpl"
	"github.com/Nigel2392/go-django/src/core/pagination"
	"github.com/Nigel2392/mux"
)

func ListTodos(w http.ResponseWriter, r *http.Request) {
	// Create a new paginator for the Todo model
	var paginator = pagination.Paginator[[]*Todo, *Todo]{
		Context: r.Context(),
		Amount:  25,
		// Define a function to retrieve a list of objects based on the amount and offset
		GetObjects: func(amount, offset int) ([]*Todo, error) {
			var rows, err = queries.ListObjects(&Todo{}, uint64(offset), uint64(amount))
			return rows, err
		},
		GetCount: func() (int, error) {
			var ct, err = queries.CountObjects(&Todo{})
			return int(ct), err
		},
	}

	// Get the page number from the request's query string
	// We provide a utility function to get the page number from a string, int(8/16/32/64) and uint(8/16/32/64/ptr).
	var pageNum = pagination.GetPageNum(
		r.URL.Query().Get("page"),
	)

	// Get the page from the paginator
	//
	// This will return a PageObject[Todo] which contains the list of todos for the current page.
	var page, err = paginator.Page(pageNum)
	if err != nil && !errors.Is(err, pagination.ErrNoResults) {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Create a new RequestContext
	// Add the page object to the context
	var context = ctx.RequestContext(r)
	context.Set("Page", page)

	// Render the template
	err = tpl.FRender(
		w, context, "todos",
		"todos/list.tmpl",
	)
	if err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func MarkTodoDone(w http.ResponseWriter, r *http.Request) {
	// Get the todo ID from the URL
	var vars = mux.Vars(r)
	var id = vars.GetInt("id")
	if id == 0 {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  "Invalid todo ID",
		})
		return
	}

	// Get the todo from the database
	var todo, err = queries.GetObject(&Todo{}, id)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Mark the todo as done
	todo.Done = !todo.Done

	// Save the todo
	err = todo.Save(r.Context())
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	// Send a JSON response
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "success",
		"done":   todo.Done,
	})
}
