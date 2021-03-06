package main

import (
	"fmt"
	"net/http"

	r "github.com/dancannon/gorethink"
	"github.com/gorilla/websocket"
)

type Handler func(*Client, interface{})

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// CORS Politic will be ignore and allow every http connection followed by Socket communcation
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Router struct {
	rules   map[string]Handler
	session *r.Session
}

func NewRouter(session *r.Session) *Router {
	return &Router{
		rules:   make(map[string]Handler),
		session: session,
	}
}

func (r *Router) Handle(msgName string, handler Handler) {
	r.rules[msgName] = handler
}

func (r *Router) FindHandler(msgName string) (Handler, bool) {
	handler, found := r.rules[msgName]
	return handler, found
}

func (e *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	socket, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, err.Error())
		return
	}
	// client need websocket connections
	client := NewClient(socket, e.FindHandler, e.session)
	defer client.Close()
	go client.Write()
	client.Read()

}
