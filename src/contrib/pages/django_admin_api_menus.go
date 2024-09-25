package pages

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Nigel2392/go-django/src/contrib/pages/models"
	"github.com/Nigel2392/go-django/src/core/except"
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

type MenuHeader struct {
	Text string `json:"text"`
	URL  string `json:"url"`
}

type pageMenuResponse struct {
	Header     MenuHeader       `json:"header,omitempty"`
	ParentItem *models.PageNode `json:"parent_item,omitempty"`
	Items      ItemsList        `json:"items"`
}

func pageMenuHandler(w http.ResponseWriter, r *http.Request) {
	var (
		ctx        = r.Context()
		mainItemID = r.URL.Query().Get("page_id")
		getParent  = r.URL.Query().Get("get_parent")
		qs         = QuerySet()
		response   = &pageMenuResponse{}
		items      []models.PageNode
		mainItem   models.PageNode
		idInt      int
		prntBool   bool
		err        error
	)

	if mainItemID == "" {
		items, err = qs.GetNodesByDepth(ctx, 0, 1000, 0)
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

	if getParent != "" {
		prntBool, err = strconv.ParseBool(getParent)
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

	if prntBool && !mainItem.IsRoot() {
		// Main item isn't a root node; we can safely fetch the parent node.
		mainItem, err = ParentNode(qs, ctx, mainItem.Path, int(mainItem.Depth))
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return
		}

	} else if prntBool && mainItem.IsRoot() {
		// Main item is a root node; we can't fetch the parent node.
		// Instead, override items and render the menu JSON.
		items, err = qs.GetNodesByDepth(ctx, 0, 1000, 0)
		if err != nil {
			except.Fail(http.StatusInternalServerError, err)
			return
		}
		goto renderJSON
	}

	// Fetch child nodes of the main item.
	items, err = qs.GetChildNodes(ctx, mainItem.Path, mainItem.Depth, 1000, 0)
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
