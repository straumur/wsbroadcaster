package ws

import (
	"code.google.com/p/go.net/websocket"
	"github.com/howbazaar/loggo"
	"github.com/straumur/straumur"
	"net/http"
)

var (
	logger = loggo.GetLogger("straumur.websocket")
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
		logger.Debugf("%+v", c.query)
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
			logger.Debugf("Added new client")
			s.clients[c.id] = c
			logger.Debugf("Now", len(s.clients), "clients connected.")

		// del a client
		case c := <-s.delCh:
			logger.Debugf("Delete client")
			delete(s.clients, c.id)

		// consume event feed
		case event := <-s.events:
			logger.Debugf("Send all:", event)
			s.sendAll(event)

		case err := <-s.errCh:
			logger.Errorf("Error:", err.Error())
			ec <- err

		case <-s.doneCh:
			return
		}
	}
}
