package pages

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Nigel2392/django/contrib/pages/models"
)

type pageMenuResponse struct {
	ParentItem *models.PageNode  `json:"parent_item,omitempty"`
	Items      []models.PageNode `json:"items"`
}

func pageMenuHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx        = r.Context()
		mainItemID = r.URL.Query().Get("page_id")
		response   = &pageMenuResponse{}
		items      []models.PageNode
		mainItem   models.PageNode
		idInt      int
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

	mainItem, err = pageApp.QuerySet().GetNodeByID(ctx, int64(idInt))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

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
