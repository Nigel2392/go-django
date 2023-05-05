package paginator

import (
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/secret"
	"gorm.io/gorm"
)

type paginator struct {
	// The current page.
	Page int
	// The number of items per page.
	Limit int
	// The total number of items.
	Total int
	// The total number of pages.
	TotalPages int
	// Base URL for the paginator.
	BaseURL *url.URL
	// Extra URL parameters.
	Extra map[string]string

	StartCut bool
	EndCut   bool
}

func (p *paginator) NextPageURL() string {
	var query = p.BaseURL.Query()
	if p.Page >= p.TotalPages {
		return ""
	}
	query.Set("page", strconv.Itoa(p.Page+1))
	query.Set("limit", strconv.Itoa(p.Limit))
	for k, v := range p.Extra {
		if k == "" || v == "" {
			continue
		}
		query.Set(k, v)
	}
	p.BaseURL.RawQuery = query.Encode()
	return p.BaseURL.String()
}

func (p *paginator) PrevPageURL() string {
	var query = p.BaseURL.Query()
	if p.Page <= 1 {
		return ""
	}
	query.Set("page", strconv.Itoa(p.Page-1))
	query.Set("limit", strconv.Itoa(p.Limit))
	for k, v := range p.Extra {
		if k == "" || v == "" {
			continue
		}
		query.Set(k, v)
	}
	p.BaseURL.RawQuery = query.Encode()
	return p.BaseURL.String()
}

func (p *paginator) LastPageURL() string {
	var query = p.BaseURL.Query()
	query.Set("page", strconv.Itoa(p.TotalPages))
	query.Set("limit", strconv.Itoa(p.Limit))
	for k, v := range p.Extra {
		if k == "" || v == "" {
			continue
		}
		query.Set(k, v)
	}
	p.BaseURL.RawQuery = query.Encode()
	return p.BaseURL.String()
}

func (p *paginator) FirstPageURL() string {
	var query = p.BaseURL.Query()
	query.Set("page", "1")
	query.Set("limit", strconv.Itoa(p.Limit))
	for k, v := range p.Extra {
		if k == "" || v == "" {
			continue
		}
		query.Set(k, v)
	}
	p.BaseURL.RawQuery = query.Encode()
	return p.BaseURL.String()
}

func (p *paginator) HasNext() bool {
	return p.Page < p.TotalPages
}

func (p *paginator) HasPrev() bool {
	return p.Page > 1
}

func (p *paginator) NextPage() int {
	if p.Page >= p.TotalPages {
		return p.TotalPages
	}
	return p.Page + 1
}

func (p *paginator) PrevPage() int {
	if p.Page <= 1 {
		return 1
	}
	return p.Page - 1
}

type Page struct {
	IsCurrent bool
	Number    int
	URL       string
}

func (p *paginator) Pages() []*Page {
	var pages = make([]*Page, 0)
	for i := 1; i <= p.TotalPages; i++ {
		pages = append(pages, &Page{
			IsCurrent: i == p.Page,
			Number:    i,
			URL: func() string {
				var query = p.BaseURL.Query()
				query.Set("page", strconv.Itoa(i))
				query.Set("limit", strconv.Itoa(p.Limit))
				for k, v := range p.Extra {
					query.Set(k, v)
				}
				p.BaseURL.RawQuery = query.Encode()
				return p.BaseURL.String()
			}(),
		})
	}
	return pages
}

func (p *paginator) ElidedPages() (pages []*Page) {
	var maxPages = 5

	var total = p.TotalPages
	if total <= maxPages {
		return p.Pages()
	}

	var page = p.Page
	var half = maxPages / 2
	var left = page - half
	var right = page + half

	if left < 1 {
		left = 1
		right = maxPages
	}
	if right > total {
		left = total - maxPages + 1
		right = total
	}

	if left > 1 {
		p.StartCut = true
	}

	if right < total {
		p.EndCut = true
	}

	for i := left; i <= right; i++ {
		pages = append(pages, &Page{
			IsCurrent: i == p.Page,
			Number:    i,
			URL: func() string {
				var query = p.BaseURL.Query()
				query.Set("page", strconv.Itoa(i))
				query.Set("limit", strconv.Itoa(p.Limit))
				for k, v := range p.Extra {
					query.Set(k, v)
				}
				p.BaseURL.RawQuery = query.Encode()
				return p.BaseURL.String()
			}(),
		})
	}

	return pages
}

func PaginateDB(page int, perPage int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Offset((page - 1) * perPage).Limit(perPage)
	}
}

func PaginateRequest(rq *request.Request, model any, baseURL string, db *gorm.DB, extra map[string]string) (page int, limit int, count int64, redirected bool) {
	var currentPageHash = secret.FnvHash(rq.Request.URL.Path)
	var pageStr = rq.Request.URL.Query().Get("page")
	var limitStr = rq.Request.URL.Query().Get("limit")

	var err error
	page, err = strconv.Atoi(pageStr)
	if err != nil {
		page = 1
	}
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit > 100 {
		var cookies = rq.Request.Cookies()
		limit = getLimitCookie(cookies, currentPageHash, 25)
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 1
	}

	var maxPage int

	// Validate if the page is out of range
	db.Model(model).Count(&count)
	maxPage = int(count) / limit
	if int(count)%limit > 0 {
		maxPage++
	}

	if limit > 0 || limit <= 100 {
		setLimitCookie(rq, currentPageHash, limit)
	}

	if page > maxPage && maxPage > 0 && count > 0 {
		page = maxPage
		var url = rq.Request.URL
		var query = url.Query()
		query.Set("page", strconv.Itoa(page))
		query.Set("limit", strconv.Itoa(limit))
		for k, v := range extra {
			query.Set(k, v)
		}
		url.RawQuery = query.Encode()
		rq.Redirect(url.String(), 302)
		return page, limit, count, true
	}

	if maxPage > 1 {
		baseURL, _ := url.Parse(baseURL)
		// rq.Data.Set("paginator", &Paginator{})
		rq.Data.Set("paginator", &paginator{
			Page:       page,
			Limit:      limit,
			Total:      int(count),
			TotalPages: maxPage,
			BaseURL:    baseURL,
			Extra:      extra,
		})
	}

	return page, limit, count, false
}

func setLimitCookie(rq *request.Request, currentPageHash secret.HashUint, limit int) {
	rq.SetCookies(&http.Cookie{
		Name:     "page-" + currentPageHash.String(),
		Value:    strconv.Itoa(limit),
		Expires:  time.Now().AddDate(0, 0, 1),
		Path:     "/",
		HttpOnly: true,
	})
}

func getLimitCookie(cookies []*http.Cookie, currentPageHash secret.HashUint, defaultLimit int) int {
	var limit = defaultLimit
	var err error
	for _, cookie := range cookies {
		if cookie.Name == "page-"+currentPageHash.String() {
			limit, err = strconv.Atoi(cookie.Value)
			if err == nil && limit > 1 && limit <= 100 {
				return limit
			}
			break
		}
	}
	return defaultLimit
}
