package links

import (
	"encoding/json"
	"net/http"
	"strconv"

	autherrors "github.com/Nigel2392/go-django/src/contrib/auth/auth_errors"
	"github.com/Nigel2392/go-django/src/contrib/pages"
	"github.com/Nigel2392/go-django/src/contrib/pages/page_models"
	"github.com/Nigel2392/go-django/src/core/except"
	"github.com/Nigel2392/go-django/src/permissions"
	"github.com/Nigel2392/mux"
)

type listResponse struct {
	ParentItem *page_models.PageNode `json:"parent_item,omitempty"`
	Items      pages.ItemsList       `json:"items"`
}

func listPages(w http.ResponseWriter, r *http.Request) {
	var (
		ctx          = r.Context()
		mainItemID   = r.URL.Query().Get(pages.PageIDVariableName)
		getParentStr = r.URL.Query().Get("get_parent")
		qs           = pages.QuerySet()
		response     = &listResponse{}
		items        []page_models.PageNode
		mainItem     page_models.PageNode
		idInt        int
		getParent    bool
		err          error
	)

	if !permissions.HasPermission(r, "pages:list") {
		except.Fail(http.StatusForbidden, nil)
		return
	}

	// If the main item ID is empty, we fetch all root nodes.
	// Then continue to render the response JSON
	if mainItemID == "" {
		items, err = qs.GetNodesByDepth(ctx, 0, page_models.StatusFlagPublished, 0, 1000)
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return
		}

		goto renderJSON
	}

	idInt, err = strconv.Atoi(mainItemID)
	if err != nil {
		except.Fail(http.StatusBadRequest, err)
		return
	}

	if getParentStr != "" {
		getParent, err = strconv.ParseBool(getParentStr)
		if err != nil {
			except.Fail(http.StatusBadRequest, err)
			return
		}
	}

	mainItem, err = qs.GetNodeByID(ctx, int64(idInt))
	if err != nil {
		except.Fail(http.StatusNotFound, err)
		return
	}

	if getParent && !mainItem.IsRoot() {
		// Main item isn't a root node; we can safely fetch the parent node.
		mainItem, err = pages.ParentNode(qs, ctx, mainItem.Path, int(mainItem.Depth))
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return
		}

	} else if getParent && mainItem.IsRoot() {
		// Main item is a root node; we can't fetch the parent node.
		// Instead, override items and render the menu JSON.
		items, err = qs.GetNodesByDepth(ctx, 0, page_models.StatusFlagPublished, 0, 1000)
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return
		}
		goto renderJSON
	}

	// Fetch child nodes of the main item.
	items, err = qs.GetChildNodes(ctx, mainItem.Path, mainItem.Depth, page_models.StatusFlagPublished, 0, 1000)
	if err != nil {
		except.Fail(http.StatusInternalServerError, err)
		return
	}
	response.ParentItem = &mainItem

renderJSON:
	response.Items = items
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}

func retrievePage(w http.ResponseWriter, r *http.Request) {
	if !permissions.HasPermission(r, "pages:list") {
		autherrors.Fail(
			403, "You do not have permission to list or view pages",
		)
	}

	// Get the page ID from the request
	var vars = mux.Vars(r)
	var pageID = vars.GetInt(
		pages.PageIDVariableName,
	)

	if pageID == 0 {
		except.Fail(
			http.StatusBadRequest, "Page ID is required",
		)
	}

	// Get the page from the database
	var page, err = pages.QuerySet().GetNodeByID(
		r.Context(), int64(pageID),
	)
	if err != nil {
		except.Fail(http.StatusNotFound, err)
	}

	// Return the page as JSON
	var encoder = json.NewEncoder(w)
	if err = encoder.Encode(page); err != nil {
		except.Fail(http.StatusInternalServerError, err)
	}
}
