/*Package ngserver get the captured http data from ngnet,
  and send these data to frontend by websocket.

          chan                    +-----NGClient
  ngnet----------NGServer---------+-----NGClient
                                  +-----NGClient
*/
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/ga0/netgraph/web"
	"golang.org/x/net/websocket"
)

// NGClient is the websocket client
type NGClient struct {
	eventChan chan interface{}
	server    *NGServer
	ws        *websocket.Conn
}

func (c *NGClient) recvAndProcessCommand() {
	for {
		var msg string
		err := websocket.Message.Receive(c.ws, &msg)
		if err != nil {
			return
		}
		if len(msg) > 0 {
			if msg == "sync" {
				c.server.sync(c)
			}
		} else {
			panic("empty command")
		}
	}
}

func (c *NGClient) transmitEvents() {
	for ev := range c.eventChan {
		json, err := json.Marshal(ev)
		if err == nil {
			websocket.Message.Send(c.ws, string(json))
		}
	}
}

func (c *NGClient) close() {
	close(c.eventChan)
}

// NewNGClient creates NGClient
func NewNGClient(ws *websocket.Conn, server *NGServer) *NGClient {
	c := new(NGClient)
	c.server = server
	c.ws = ws
	c.eventChan = make(chan interface{}, 16)
	return c
}

// NGServer is a http server which push captured HTTPEvent to the front end
type NGServer struct {
	addr            string
	staticFileDir   string
	connectedClient map[*websocket.Conn]*NGClient
	eventBuffer     []interface{}
	saveEvent       bool
	wg              sync.WaitGroup
}

func (s *NGServer) websocketHandler(ws *websocket.Conn) {
	c := NewNGClient(ws, s)
	s.connectedClient[ws] = c
	go c.transmitEvents()
	c.recvAndProcessCommand()
	c.close()
	delete(s.connectedClient, ws)
}

// PushEvent dispatches the event received from ngnet to all clients connected with websocket.
func (s *NGServer) PushEvent(e interface{}) {
	if s.saveEvent {
		s.eventBuffer = append(s.eventBuffer, e)
	}
	for _, c := range s.connectedClient {
		c.eventChan <- e
	}
}

// Wait waits for serving
func (s *NGServer) Wait() {
	s.wg.Wait()
}

/*
   If the flag '-s' is set and the browser sent a 'sync' command,
   the NGServer will push all the http message buffered in eventBuffer to
   the client.
*/
func (s *NGServer) sync(c *NGClient) {
	for _, ev := range s.eventBuffer {
		c.eventChan <- ev
	}
}

/*
   Handle static files (.html, .js, .css).
*/
func (s *NGServer) handleStaticFile(w http.ResponseWriter, r *http.Request) {
	uri := r.RequestURI
	if uri == "/" {
		uri = "/index.html"
	}
	c, err := web.GetContent(uri)
	if err != nil {
		log.Println(r.RequestURI)
		http.NotFound(w, r)
		return
	}
	w.Write([]byte(c))
}

// Serve the web page
func (s *NGServer) Serve() {
	http.Handle("/data", websocket.Handler(s.websocketHandler))

	/*
	   If './client' directory exists, create a FileServer with it,
	   otherwise we use package client.
	*/
	_, err := os.Stat("client")
	if err == nil {
		fs := http.FileServer(http.Dir("client"))
		http.Handle("/", fs)
	} else {
		http.HandleFunc("/", s.handleStaticFile)
	}
	s.wg.Add(1)
	defer s.wg.Done()
	err = http.ListenAndServe(s.addr, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

// NewNGServer creates NGServer
func NewNGServer(addr string, saveEvent bool) *NGServer {
	s := new(NGServer)
	s.addr = addr
	s.connectedClient = make(map[*websocket.Conn]*NGClient)
	s.saveEvent = saveEvent
	return s
}
