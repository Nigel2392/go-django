package websocket

import (
	"net/http"
	"net/url"

	"github.com/Nigel2392/router/v3/request"
	"github.com/Nigel2392/router/v3/request/params"
	"github.com/Nigel2392/router/v3/request/response"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type Conn[T any] struct {
	*websocket.Conn
	id   uuid.UUID
	data T
}

func (c *Conn[T]) ID() uuid.UUID {
	return c.id
}

func (c *Conn[T]) Data() T {
	return c.data
}

type WebSocket[T any] struct {
	Upgrader       *websocket.Upgrader
	Handler        func(*Conn[T])
	NewDataFunc    func(*request.Request, *websocket.Conn) T
	CheckOrigin    func(*http.Request) bool
	ResponseHeader http.Header
}

func (s *WebSocket[T]) Endpoint() func(r *request.Request) {
	if s.Handler == nil {
		panic("websocket handler is nil")
	}
	if s.Upgrader == nil {
		panic("websocket upgrader is nil")
	}
	return func(r *request.Request) {
		var conn, err = s.Upgrader.Upgrade(r.Response, r.Request, s.ResponseHeader)
		if err != nil {
			return
		}

		uuid, err := uuid.NewRandom()
		if err != nil {
			conn.WriteJSON(response.JSONError{
				StatusCode: http.StatusInternalServerError,
				Message:    "Internal Server Error",
			})
			return
		}

		var data T
		if s.NewDataFunc != nil {
			data = s.NewDataFunc(r, conn)
		}

		var c = &Conn[T]{
			id:   uuid,
			Conn: conn,
			data: data,
		}
		s.Handler(c)
	}
}

type SocketData struct {
	User  request.User
	Vars  params.URLParams
	Query url.Values
}

func NewWebSocket(handler func(*Conn[*SocketData])) *WebSocket[*SocketData] {
	return &WebSocket[*SocketData]{
		Upgrader: &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		Handler: handler,
		NewDataFunc: func(r *request.Request, conn *websocket.Conn) *SocketData {
			return &SocketData{
				User:  r.User,
				Vars:  r.URLParams,
				Query: r.QueryParams,
			}
		},
	}
}
