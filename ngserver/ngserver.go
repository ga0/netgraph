package ngserver

import (
    "encoding/json"
    "flag"
    "golang.org/x/net/websocket"
    "log"
    "net/http"
)

var saveEvent = flag.Bool("s", false, "save network event in server")

func init() {
}

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
}

func (s *NGServer) webHandler(ws *websocket.Conn) {
    c := NewNGClient(ws, s)
    s.connectedClient[ws] = c
    go c.TransmitEvents()
    c.RecvAndProcessCommand()
    c.Close()
    delete(s.connectedClient, ws)
}

func (s *NGServer) DispatchEvent() {
    for ev := range s.eventChan {
        if *saveEvent {
            s.eventBuffer = append(s.eventBuffer, ev)
        }
        for _, c := range s.connectedClient {
            c.eventChan <- ev
        }
    }
}

func (s *NGServer) Sync(c *NGClient) {
    for _, ev := range s.eventBuffer {
        c.eventChan <- ev
    }
}

func (s *NGServer) Serve() {
    go s.DispatchEvent()
    http.Handle("/data", websocket.Handler(s.webHandler))
    fs := http.FileServer(http.Dir(s.staticFileDir))
    http.Handle("/", fs)
    err := http.ListenAndServe(s.addr, nil)
    if err != nil {
        log.Fatalln(err)
    }
}

func NewNGServer(addr string, staticFileDir string, eventChan chan interface{}) *NGServer {
    s := new(NGServer)
    s.eventChan = eventChan
    s.addr = addr
    s.staticFileDir = staticFileDir
    s.connectedClient = make(map[*websocket.Conn]*NGClient)
    return s
}
