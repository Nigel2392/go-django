package serializer

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/go-structs"
	"github.com/Nigel2392/router/v3"
	"github.com/Nigel2392/router/v3/middleware/csrf"
	"github.com/Nigel2392/router/v3/request"

	_ "embed"
)

var (
	pageParam  string = "page"
	limitParam string = "limit"

	//go:embed assets/index.html
	indexTemplate string

	//go:embed assets/*
	assetFS embed.FS

	ui_index = template.Must(template.New("index").Parse(indexTemplate))

	STATIC_URL string

	api_url                              = "/api"
	serializer_endpoint router.Registrar = router.NewRoute(router.GET, api_url, "serializer", endPointIndex)
	allRoutes           []routeObject    = make([]routeObject, 0)
)

func endPointIndex(r *request.Request) {

	var accept = r.GetHeader("Accept")
	if IsApiResponse(accept) {
		__json(r, allRoutes)
		return
	} else {
		__html(r, UIContext{
			Static: STATIC_URL,
			Object: allRoutes,
			Index:  "/",
			// EscapeHTML: true,
		})
		return
	}
}

func Endpoint() router.Registrar {
	serializer_endpoint.AddGroup(staticRoute(api_url+"/static", "/static"))
	return serializer_endpoint
}

type Serializer[T Model[T]] struct {
	baseURL    string
	getObjects ObjectGetterFunc[T]
	structie   *structs.Struct
	Validators structs.ValidatorMap
	config     ModelConfig
}

type ModelConfig struct {
	MaxPerPage               int
	URL                      string
	RouteName                string
	CSRF_Protection          bool
	DisableRegisterInURLList bool
	Fields                   []string
}

// Register creates a new serializer for a listview, createview and updateview with the given maxPerPage.
// The maxPerPage is the maximum number of items per page.
// The getObjects function is used to get the objects for the current page.
// If the object does not implement interfaces.Lister, an error is returned.
// This is the only error which can be returned.
func Register[T Model[T]](conf ModelConfig) (*Serializer[T], error) {
	var s = structs.New("json")
	var typeOf = reflect.TypeOf(*new(T))
	if typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}

	for _, field := range conf.Fields {
		fieldType, ok := typeOf.FieldByName(field)
		if !ok {
			panic("field " + field + " does not exist")
		}
		s.AddField(field, strings.ToLower(field), fieldType.Type)
	}

	s.Make()

	var url = fmt.Sprintf("%s%s", serializer_endpoint.Format(), conf.URL)
	var serializer = &Serializer[T]{
		getObjects: (*new(T)).List,
		structie:   s,
		Validators: make(structs.ValidatorMap),
		baseURL:    url,
		config:     conf,
	}
	serializer_endpoint.AddGroup(
		serializer.route(),
	)
	return serializer, nil
}

// Serves the serializer at the registered URL.
//
// The serializer needs the following header to be set for interacting with raw JSON:
//   - Accept: application/json
func (s *Serializer[T]) ServeHTTP(r *request.Request) {
	var page, itemsPerPage = queryInt(r, pageParam, 1), queryInt(r, limitParam, 10)

	if itemsPerPage > s.config.MaxPerPage {
		itemsPerPage = s.config.MaxPerPage
	}

	var items, totalCount, err = s.getObjects(page, itemsPerPage)
	if err != nil {
		NewErrorResponse(500, err.Error()).WriteTo(r)
		return
	}

	var structList = make([]interface{}, len(items))
	for i, item := range items {
		var iFace = s.structie.NewPointer()
		err = structs.ScanInto(item, iFace, []string{"json"}, nil, s.config.Fields...)
		if err != nil {
			NewErrorResponse(500, err.Error()).WriteTo(r)
			return
		}
		structList[i] = iFace
	}

	var hasNext = page*itemsPerPage < int(totalCount)
	var hasPrevious = page > 1

	var nextURL, previousURL string
	var queryParams = r.QueryParams
	queryParams.Set(limitParam, strconv.Itoa(itemsPerPage))

	if hasNext {
		queryParams.Set(pageParam, strconv.Itoa(page+1))
		var encoded = queryParams.Encode()
		var b = make([]byte, len(encoded)+len(s.baseURL)+1)
		copy(b, s.baseURL)
		b[len(s.baseURL)] = '?'
		copy(b[len(s.baseURL)+1:], encoded)
		nextURL = string(b)
	}

	if hasPrevious {
		queryParams.Set(pageParam, strconv.Itoa(page-1))
		var encoded = queryParams.Encode()
		var b = make([]byte, len(encoded)+len(s.baseURL)+1)
		copy(b, s.baseURL)
		b[len(s.baseURL)] = '?'
		copy(b[len(s.baseURL)+1:], encoded)
		previousURL = string(b)
	}

	var numPages int
	if totalCount == 0 || itemsPerPage == 0 {
		numPages = 0
	} else {
		numPages = int(totalCount)/itemsPerPage + 1
	}

	var paginator = Paginator{
		Count:        len(items),
		Next:         nextURL,
		Previous:     previousURL,
		Results:      structList,
		ItemsPerPage: itemsPerPage,
		Page:         page,
		NumPages:     numPages,
	}

	var accept = r.GetHeader("Accept")
	if IsApiResponse(accept) {
		__json(r, paginator)
	} else {
		__html(r, UIContext{
			Paginator:    &paginator,
			Static:       STATIC_URL,
			LimitOptions: []int{10, 20, 50, 100},
			Index:        "/",
		})
	}
}

func (s *Serializer[T]) DetailView() router.Handler {
	return router.HandleFunc(func(r *request.Request) {
		var getter T
		var id = r.URLParams.Get("id")
		if id == "" {
			NewErrorResponse(400, "No ID provided.").WriteTo(r)
			return
		}
		var item, err = getter.GetFromStringID(id)
		if err != nil {
			NewErrorResponse(400, "Invalid ID.").WriteTo(r)
			return
		}

		var iFace = s.structie.NewPointer()
		err = structs.ScanInto(item, iFace, []string{"json"}, nil, s.config.Fields...)
		if err != nil {
			NewErrorResponse(500, err.Error()).WriteTo(r)
			return
		}

		var accept = r.GetHeader("Accept")
		if IsApiResponse(accept) {
			__json(r, iFace)
		} else {
			__html(r, UIContext{
				Static: STATIC_URL,
				Index:  "/",
				Object: iFace,
			})

		}
	})
}

func (s *Serializer[T]) CreateView() router.Handler {
	return router.HandleFunc(func(r *request.Request) {
		if r.Method() == http.MethodGet {
			var accept = r.Request.Header.Get("Accept")
			if IsApiResponse(accept) {
				var fields = getTypedJSON(s.structie)
				if s.config.CSRF_Protection {
					__json(r, CSRFResponse{
						CSRFToken: csrf.Token(r),
						Detail:    fields,
					})
				} else {
					__json(r, fields)
				}
			} else {
				var fields = getTypedJSON(s.structie)
				r.SetHeader("Content-Type", "application/json")
				var object any
				if s.config.CSRF_Protection {
					object = CSRFResponse{
						CSRFToken: csrf.Token(r),
						Detail:    fields,
					}
				} else {
					object = fields
				}
				__html(r, UIContext{
					Static: STATIC_URL,
					Index:  "/",
					Object: object,
				})
			}
			return
		} else if r.Method() != http.MethodPost {
			NewErrorResponse(405, "Method not allowed.").WriteTo(r)
			return
		}
		var ModelItem T
		var item = s.structie.NewPointer()
		var err = json.NewDecoder(r.Request.Body).Decode(item)
		if err != nil {
			NewErrorResponse(400, "Error decoding JSON.").WriteTo(r)
			return
		}
		if valueOf := reflect.ValueOf(ModelItem); valueOf.Kind() == reflect.Ptr {
			if valueOf.IsNil() {
				ModelItem = reflect.New(valueOf.Type().Elem()).Interface().(T)
			}
			err = structs.ScanInto(item, ModelItem, []string{"json"}, s.Validators, s.config.Fields...)
		} else {
			err = structs.ScanInto(item, &ModelItem, []string{"json"}, s.Validators, s.config.Fields...)
		}
		if err != nil {
			NewErrorResponse(400, err.Error()).WriteTo(r)
			return
		}

		err = ModelItem.Save(true)
		if err != nil {
			NewErrorResponse(500, "Error saving item.").WriteTo(r)
			return
		}
		r.SetHeader("Content-Type", "application/json")
		json.NewEncoder(r).Encode(item)
	})

}

func (s *Serializer[T]) UpdateView() router.Handler {
	return router.HandleFunc(func(r *request.Request) {
		var err error
		if r.Method() == http.MethodGet {
			var accept = r.Request.Header.Get("Accept")
			if IsApiResponse(accept) {
				var id = r.URLParams.Get("id")
				var ModelItem T
				ModelItem, err = ModelItem.GetFromStringID(id)
				if err != nil {
					NewErrorResponse(500, "Error getting item.").WriteTo(r)
					return
				}
				var item = s.structie.NewPointer()
				err = structs.ScanInto(ModelItem, item, []string{"json"}, nil, s.config.Fields...)
				if err != nil {
					NewErrorResponse(500, err.Error()).WriteTo(r)
					return
				}
				if s.config.CSRF_Protection {
					__json(r, CSRFResponse{
						CSRFToken: csrf.Token(r),
						Detail:    item,
					})
				} else {
					__json(r, item)
				}
			} else {
				var fields = getTypedJSON(s.structie)
				var object any
				if s.config.CSRF_Protection {
					object = CSRFResponse{
						CSRFToken: csrf.Token(r),
						Detail:    fields,
					}
				} else {
					object = fields
				}
				r.SetHeader("Content-Type", "application/json")
				__html(r, UIContext{
					Static: STATIC_URL,
					Index:  "/",
					Object: object,
				})
			}
			return
		} else if r.Method() != http.MethodPost {
			NewErrorResponse(405, "Method not allowed.").WriteTo(r)
			return
		}
		var modelItem = *new(T)
		var item = s.structie.NewPointer()
		err = json.NewDecoder(r.Request.Body).Decode(&item)
		if err != nil {
			NewErrorResponse(400, "Error decoding JSON.").WriteTo(r)
			return
		}
		if valueOf := reflect.ValueOf(modelItem); valueOf.Kind() == reflect.Ptr {
			if valueOf.IsNil() {
				modelItem = reflect.New(valueOf.Type().Elem()).Interface().(T)
			}
			err = structs.ScanInto(item, modelItem, []string{"json"}, s.Validators, s.config.Fields...)
		} else {
			err = structs.ScanInto(item, &modelItem, []string{"json"}, s.Validators, s.config.Fields...)
		}
		if err != nil {
			NewErrorResponse(400, err.Error()).WriteTo(r)
			return
		}
		err = modelItem.Save(false)
		if err != nil {
			var errResp = NewErrorResponse(500, "Error saving item.")
			errResp.AddError(err)
			errResp.WriteTo(r)
			return
		}
		r.SetHeader("Content-Type", "application/json")
		json.NewEncoder(r).Encode(item)
	})
}

func __json(r *request.Request, obj any) {
	r.SetHeader("Content-Type", "application/json")
	var encoder = json.NewEncoder(r)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	var err = encoder.Encode(obj)
	if err != nil {
		r.Response.Clear()
		NewErrorResponse(500, err.Error()).WriteTo(r)
		return
	}
}

func __html(r *request.Request, uiContext UIContext) {
	var buf = new(bytes.Buffer)
	var enc = json.NewEncoder(buf)
	enc.SetEscapeHTML(uiContext.EscapeHTML)
	enc.SetIndent("", "  ")
	if uiContext.Object == nil {
		enc.Encode(uiContext.Paginator)
	} else {
		enc.Encode(uiContext.Object)
	}
	uiContext.Content = template.HTML(html.EscapeString(buf.String()))
	buf.Reset()
	err := ui_index.Execute(buf, uiContext)
	if err != nil {
		r.Response.Clear()
		NewErrorResponse(500, err.Error()).WriteTo(r)
		return
	}
	r.SetHeader("Content-Type", "text/html")
	r.Response.Write(buf.Bytes())
}

type routeObject struct {
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Methods []string `json:"methods"`
}

func (s *Serializer[T]) route() router.Registrar {
	var baseRoute = router.NewRoute(router.GET, s.config.URL, s.config.RouteName, s.ServeHTTP)
	var createView = baseRoute.Any("/create", s.CreateView(), "create")
	var updateView = baseRoute.Any("/update/<<id:alphanum>>", s.UpdateView(), "update")
	var detailView = baseRoute.Any("/<<id:alphanum>>", s.DetailView(), "detail")
	if !s.config.DisableRegisterInURLList {
		allRoutes = append(allRoutes, routeObject{
			Name:    baseRoute.(*router.Route).Name(),
			URL:     baseRoute.Format(),
			Methods: []string{"GET"},
		})
		allRoutes = append(allRoutes, routeObject{
			Name:    createView.(*router.Route).Name(),
			URL:     createView.Format(),
			Methods: []string{"GET", "POST"},
		})
		allRoutes = append(allRoutes, routeObject{
			Name:    updateView.(*router.Route).Name(),
			URL:     updateView.Format(),
			Methods: []string{"GET", "POST"},
		})
		allRoutes = append(allRoutes, routeObject{
			Name:    detailView.(*router.Route).Name(),
			URL:     detailView.Format(),
			Methods: []string{"GET"},
		})
	}
	return baseRoute
}

func IsApiResponse(accept string) bool {
	return strings.Contains(accept, "application/json") || accept == ""
}

func staticRoute(abs_url, static_url string) router.Registrar {
	assetFS, err := fs.Sub(assetFS, "assets")
	if err != nil {
		panic(err)
	}
	STATIC_URL = abs_url
	return router.NewRoute(
		router.GET,
		static_url+"/<<any>>",
		"static",
		router.NewFSHandler(abs_url, assetFS).ServeHTTP,
	)
}

func queryInt(r *request.Request, key string, defaultValue int) int {
	var value = r.QueryParams.Get(key)
	if value == "" {
		return defaultValue
	}
	var intValue, err = strconv.Atoi(value)
	if err != nil {
		return defaultValue
	}
	return intValue
}
