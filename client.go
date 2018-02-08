package main

import (
	"fmt"
	"log"

	r "github.com/dancannon/gorethink"
	"github.com/gorilla/websocket"
)

type FindHandler func(string) (Handler, bool)

type Client struct {
	send         chan Message
	socket       *websocket.Conn
	findHandler  FindHandler
	session      *r.Session
	stopChannels map[int]chan bool
	id           string
	userName     string
}

func NewClient(socket *websocket.Conn, findHandler FindHandler, session *r.Session) *Client {
	var user User
	user.Name = "anônimo"
	rs, err := r.Table("user").Insert(user).RunWrite(session)
	if err != nil {
		log.Println(err.Error())
	}
	var id string
	if len(rs.GeneratedKeys) > 0 {
		id = rs.GeneratedKeys[0]
	}
	return &Client{
		send:         make(chan Message),
		socket:       socket,
		findHandler:  findHandler,
		session:      session,
		id:           id,
		userName:     user.Name,
		stopChannels: make(map[int]chan bool),
	}
}

func (client *Client) Write() {
	for msg := range client.send {
		if err := client.socket.WriteJSON(msg); err != nil {
			break
		}
	}
	client.socket.Close()
}

func (client *Client) Read() {
	var msg Message
	for {
		if err := client.socket.ReadJSON(&msg); err != nil {
			break
		}
		// Decide what function should we call based on msg.Name
		if handler, found := client.findHandler(msg.Name); found {
			handler(client, msg.Data)
		}
	}
	client.socket.Close()
}

func (client *Client) NewStopChannels(stopKey int) chan bool {
	client.StopChannelForKey(stopKey)
	stop := make(chan bool)
	client.stopChannels[stopKey] = stop
	fmt.Println(stop)
	return stop
}

// Close conenctions with channels
func (client *Client) Close() {
	for _, channel := range client.stopChannels {
		channel <- true
	}
	close(client.send)
	r.Table("user").Get(client.id).Delete().Exec(client.session)
}

func (client *Client) StopChannelForKey(key int) {
	if ch, found := client.stopChannels[key]; found {
		ch <- true
		delete(client.stopChannels, key)
	}
}
