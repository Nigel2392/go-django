package pages

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type ItemsList []models.PageNode

func (il ItemsList) MarshalJSON() ([]byte, error) {
	var items = make([]models.PageNode, 0, len(il))
	for _, item := range il {
		if item.StatusFlags.Is(models.StatusFlagHidden) ||
			item.StatusFlags.Is(models.StatusFlagDeleted) {
			continue
		}
		items = append(items, item)
	}
	return json.Marshal(items)

}

type pageMenuResponse struct {
	ParentItem *models.PageNode `json:"parent_item,omitempty"`
	Items      ItemsList        `json:"items"`
}

func pageMenuHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx        = r.Context()
		mainItemID = r.URL.Query().Get("page_id")
		getParent  = r.URL.Query().Get("get_parent")
		response   = &pageMenuResponse{}
		items      []models.PageNode
		mainItem   models.PageNode
		idInt      int
		prntBool   bool
		err        error
	)

	if mainItemID == "" {
		items, err = pageApp.QuerySet().GetNodesByDepth(ctx, 0)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

		goto renderJSON
	}

	idInt, err = strconv.Atoi(mainItemID)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if getParent != "" {
		prntBool, err = strconv.ParseBool(getParent)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
	}

	mainItem, err = pageApp.QuerySet().GetNodeByID(ctx, int64(idInt))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	if prntBool && !mainItem.IsRoot() {
		// Main item isn't a root node; we can safely fetch the parent node.
		mainItem, err = ParentNode(pageApp.QuerySet(), ctx, mainItem.Path, int(mainItem.Depth))
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}

	} else if prntBool && mainItem.IsRoot() {
		// Main item is a root node; we can't fetch the parent node.
		// Instead, override items and render the menu JSON.
		items, err = pageApp.QuerySet().GetNodesByDepth(ctx, 0)
		if err != nil {
			http.Error(w, err.Error(), 500)
			return
		}
		goto renderJSON
	}

	// Fetch child nodes of the main item.
	items, err = pageApp.QuerySet().GetChildNodes(ctx, mainItem.Path, mainItem.Depth)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	response.ParentItem = &mainItem

renderJSON:
	response.Items = items
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(response)
}
