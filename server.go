package ws

import (
	"code.google.com/p/go.net/websocket"
	"github.com/straumur/straumur"
	"log"
	"net/http"
)

type Server struct {
	pattern string
	events  chan *straumur.Event
	clients map[int]*Client
	addCh   chan *Client
	delCh   chan *Client
	doneCh  chan bool
	errCh   chan error
}

// Create a new Websocket broadcaster
func NewServer(pattern string) *Server {

	clients := make(map[int]*Client)
	addCh := make(chan *Client)
	delCh := make(chan *Client)
	doneCh := make(chan bool)
	errCh := make(chan error)
	events := make(chan *straumur.Event)

	return &Server{
		pattern,
		events,
		clients,
		addCh,
		delCh,
		doneCh,
		errCh,
	}
}

func (s *Server) Add(c *Client) {
	s.addCh <- c
}

func (s *Server) Del(c *Client) {
	s.delCh <- c
}

func (s *Server) Done() {
	s.doneCh <- true
}

func (s *Server) Err(err error) {
	s.errCh <- err
}

func (s *Server) sendAll(event *straumur.Event) {
	for _, c := range s.clients {
		log.Printf("%+v", c.query)
		if c.query.Match(*event) {
			c.Write(event)
		}
	}
}

func (s *Server) Broadcast(e *straumur.Event) {
	s.events <- e
}

func (s *Server) Run(ec chan error) {

	// websocket handler
	onConnected := func(ws *websocket.Conn) {

		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		client := NewClient(ws, s)
		s.Add(client)
		client.Listen()
	}

	http.Handle(s.pattern, websocket.Handler(onConnected))

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			log.Println("Added new client")
			s.clients[c.id] = c
			log.Println("Now", len(s.clients), "clients connected.")

		// del a client
		case c := <-s.delCh:
			log.Println("Delete client")
			delete(s.clients, c.id)

		// consume event feed
		case event := <-s.events:
			log.Println("Send all:", event)
			s.sendAll(event)

		case err := <-s.errCh:
			log.Println("Error:", err.Error())
			ec <- err

		case <-s.doneCh:
			return
		}
	}
}
