/*
   package ngserver get the captured http data from ngnet,
   and send these data to frontend by websocket.

           chan                    +-----NGClient
   ngnet----------NGServer---------+-----NGClient
                                   +-----NGClient
*/
package ngserver

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"unicode"

	"github.com/ga0/netgraph/client"
	"golang.org/x/net/websocket"
)

type NGClient struct {
	eventChan chan interface{}
	server    *NGServer
	ws        *websocket.Conn
}

func (c *NGClient) RecvAndProcessCommand() {
	for {
		var msg string
		err := websocket.Message.Receive(c.ws, &msg)
		if err != nil {
			return
		}
		if len(msg) > 0 {
			if msg == "sync" {
				c.server.Sync(c)
			}
		} else {
			panic("empty command")
		}
	}
}

func (c *NGClient) TransmitEvents() {
	for ev := range c.eventChan {
		json, err := json.Marshal(ev)
		if err == nil {
			websocket.Message.Send(c.ws, string(json))
		}
	}
}

func (c *NGClient) Close() {
	close(c.eventChan)
}

func NewNGClient(ws *websocket.Conn, server *NGServer) *NGClient {
	c := new(NGClient)
	c.server = server
	c.ws = ws
	c.eventChan = make(chan interface{}, 16)
	return c
}

type NGServer struct {
	eventChan       chan interface{}
	addr            string
	staticFileDir   string
	connectedClient map[*websocket.Conn]*NGClient
	eventBuffer     []interface{}
	saveEvent       bool
}

func (s *NGServer) websocketHandler(ws *websocket.Conn) {
	c := NewNGClient(ws, s)
	s.connectedClient[ws] = c
	go c.TransmitEvents()
	c.RecvAndProcessCommand()
	c.Close()
	delete(s.connectedClient, ws)
}

func isBinary(body []byte) bool {
	for c := range body {
		if !unicode.IsGraphic(rune(c)) {
			return true
		}
	}
	return false
}

/*
   Dispatch the event received from ngnet to all clients connected with websocket.
*/
func (s *NGServer) DispatchEvent() {
	for ev := range s.eventChan {
		/*var body []byte
				switch v := ev.(type) {
				case ngnet.HTTPRequestEvent:
					if isBinary(v.Body) {
		                v.Body = body
		            }
				case ngnet.HTTPResponseEvent:
					body = v.Body
				default:
					log.Println("Unkown event")
				}*/

		if s.saveEvent {
			s.eventBuffer = append(s.eventBuffer, ev)
		}
		for _, c := range s.connectedClient {
			c.eventChan <- ev
		}
	}
}

/*
   If the flag '-s' is set and the browser sent a 'sync' command,
   the NGServer will push all the http message buffered in eventBuffer to
   the client.
*/
func (s *NGServer) Sync(c *NGClient) {
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
	c, err := client.GetContent(uri)
	if err != nil {
		log.Println(r.RequestURI)
		http.NotFound(w, r)
		return
	}
	w.Write([]byte(c))
}

func (s *NGServer) Serve() {
	go s.DispatchEvent()
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

	err = http.ListenAndServe(s.addr, nil)
	if err != nil {
		log.Fatalln(err)
	}
}

func NewNGServer(addr string, eventChan chan interface{}, saveEvent bool) *NGServer {
	s := new(NGServer)
	s.eventChan = eventChan
	s.addr = addr
	s.connectedClient = make(map[*websocket.Conn]*NGClient)
	s.saveEvent = saveEvent
	return s
}
