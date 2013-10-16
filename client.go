package ws

import (
	"code.google.com/p/go.net/websocket"
	"fmt"
	"github.com/straumur/straumur"
	"io"
)

var maxId int = 0

//Represents a connected websocket client
type Client struct {
	id     int
	ws     *websocket.Conn
	server *Server
	ch     chan *straumur.Event
	doneCh chan bool
	query  straumur.Query
}

func NewClient(ws *websocket.Conn, server *Server) *Client {

	if ws == nil {
		panic("ws cannot be nil")
	}

	maxId++
	ch := make(chan *straumur.Event)
	doneCh := make(chan bool)
	query := straumur.Query{}

	return &Client{maxId, ws, server, ch, doneCh, query}
}

func (c *Client) Conn() *websocket.Conn {
	return c.ws
}

func (c *Client) Write(e *straumur.Event) {
	select {
	case c.ch <- e:
	default:
		c.server.Del(c)
		err := fmt.Errorf("client %d is disconnected.", c.id)
		c.server.Err(err)
	}
}

func (c *Client) Done() {
	c.doneCh <- true
}

func (c *Client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

func (c *Client) listenWrite() {

	for {
		select {

		case event := <-c.ch:
			err := websocket.JSON.Send(c.ws, event)
			if err != nil {
				c.server.Err(err)
			}

		case <-c.doneCh:
			c.server.Del(c)
			c.doneCh <- true
			return
		}
	}
}

func (c *Client) listenRead() {

	for {
		select {

		case <-c.doneCh:
			c.server.Del(c)
			c.doneCh <- true
			return

		// read data from websocket connection
		default:
			var q straumur.Query
			err := websocket.JSON.Receive(c.ws, &q)
			if err == io.EOF {
				c.doneCh <- true
			} else if err != nil {
				c.server.Err(err)
			} else {
				c.query = q
			}
		}
	}
}
