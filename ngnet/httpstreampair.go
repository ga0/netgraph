package ngnet

import (
	"time"
)

// HTTPHeaderItem is HTTP header key-value pair
type HTTPHeaderItem struct {
	Name  string
	Value string
}

// HTTPEvent is HTTP request or response
type HTTPEvent struct {
	Type      string
	Start     time.Time
	End       time.Time
	StreamSeq uint
}

// HTTPRequestEvent is HTTP request
type HTTPRequestEvent struct {
	HTTPEvent
	ClientAddr string
	Method     string
	URI        string
	Version    string
	Headers    []HTTPHeaderItem
	Body       []byte
}

// HTTPResponseEvent is HTTP response
type HTTPResponseEvent struct {
	HTTPEvent
	Version string
	Code    uint
	Reason  string
	Headers []HTTPHeaderItem
	Body    []byte
}

// httpStreamPair is Bi-direction HTTP stream pair
type httpStreamPair struct {
	upStream   *httpStream
	downStream *httpStream

	requestSeq uint
	sem        chan byte
	connSeq    uint
	eventChan  chan<- interface{}
}

func newHTTPStreamPair(seq uint, eventChan chan<- interface{}) *httpStreamPair {
	pair := new(httpStreamPair)
	pair.connSeq = seq
	pair.sem = make(chan byte, 1)
	pair.eventChan = eventChan

	return pair
}

func (pair *httpStreamPair) run() {
	defer func() {
		if r := recover(); r != nil {
			if pair.upStream != nil {
				close(pair.upStream.reader.stopCh)
			}
			if pair.downStream != nil {
				close(pair.downStream.reader.stopCh)
			}
			//fmt.Printf("HTTPStream (#%d %v) error: %v\n", pair.connSeq, pair.upStream.key, r)
		}
	}()

	for {
		pair.handleTransaction()
		pair.requestSeq++
	}
}

func (pair *httpStreamPair) handleTransaction() {
	upStream := pair.upStream
	method, uri, version := upStream.getRequestLine()
	reqStart := upStream.reader.lastSeen
	reqHeaders := upStream.getHeaders()
	reqBody := upStream.getBody(method, reqHeaders, true)

	var req HTTPRequestEvent
	req.ClientAddr = pair.upStream.key.net.Src().String() + ":" + pair.upStream.key.tcp.Src().String()
	req.Type = "HTTPRequest"
	req.Method = method
	req.URI = uri
	req.Version = version
	req.Headers = reqHeaders
	req.Body = reqBody
	req.StreamSeq = pair.connSeq
	req.Start = reqStart
	req.End = upStream.reader.lastSeen
	pair.eventChan <- req

	downStream := pair.downStream
	respVersion, code, reason := downStream.getResponseLine()
	respStart := downStream.reader.lastSeen
	respHeaders := downStream.getHeaders()
	respBody := downStream.getBody(method, respHeaders, false)

	var resp HTTPResponseEvent
	resp.Type = "HTTPResponse"
	resp.Version = respVersion
	resp.Code = uint(code)
	resp.Reason = reason
	resp.Headers = respHeaders
	resp.Body = respBody
	resp.StreamSeq = pair.connSeq
	resp.Start = respStart
	resp.End = downStream.reader.lastSeen
	pair.eventChan <- resp
}
