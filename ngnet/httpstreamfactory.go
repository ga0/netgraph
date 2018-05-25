package ngnet

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
)

var (
	httpRequestFirtLine  *regexp.Regexp
	httpResponseFirtLine *regexp.Regexp
)

func init() {
	httpRequestFirtLine = regexp.MustCompile("([A-Z]+) (.+) (HTTP/.+)")
	httpResponseFirtLine = regexp.MustCompile("(HTTP/.+) (\\d{3}) (.+)")
}

type streamKey struct {
	net, tcp gopacket.Flow
}

// httpStreamPair is Bi-direction HTTP stream pair
type httpStreamPair struct {
	upStream   *httpStream
	downStream *httpStream
	requestSeq uint
	sem        chan byte
	connSeq    uint
	eventChan  chan interface{}
}

func (k streamKey) String() string {
	return fmt.Sprintf("{%v:%v} -> {%v:%v}", k.net.Src(), k.tcp.Src(), k.net.Dst(), k.tcp.Dst())
}

func (pair *httpStreamPair) handleTransaction() {
	upStream := pair.upStream
	method, uri, version := upStream.getRequestLine()
	reqHeaders := upStream.getHeaders()
	reqBody := upStream.getBody(method, reqHeaders, true)

	req := new(HTTPRequestEvent)
	req.Type = "HTTPRequest"
	req.Method = method
	req.Uri = uri
	req.Version = version
	req.Headers = reqHeaders
	req.Body = reqBody
	req.StreamSeq = pair.connSeq
	pair.eventChan <- req

	downStream := pair.downStream
	respVersion, code, reason := downStream.getResponseLine()
	respHeaders := downStream.getHeaders()
	respBody := downStream.getBody(method, respHeaders, false)

	resp := new(HTTPResponseEvent)
	resp.Type = "HTTPResponse"
	resp.Version = respVersion
	resp.Code = uint(code)
	resp.Reason = reason
	resp.Headers = respHeaders
	resp.Body = respBody
	resp.StreamSeq = pair.connSeq
	pair.eventChan <- resp
}

func (pair *httpStreamPair) run() {
	defer func() {
		if r := recover(); r != nil {
			if pair.upStream != nil {
				*pair.upStream.bad = true
			}
			if pair.downStream != nil {
				*pair.downStream.bad = true
			}
			//fmt.Printf("HTTPStream (#%d %v) error: %v\n", pair.connSeq, pair.upStream.key, r)
		}
	}()

	for {
		pair.handleTransaction()
		pair.requestSeq++
	}
}

func newHTTPStream(src packetSource, key streamKey) httpStream {
	var s httpStream
	s.reader = NewStreamReader(src)
	s.bytes = new(uint64)
	s.key = key
	s.bad = new(bool)
	return s
}

// HTTPStreamFactory implements StreamFactory interface for tcpassembly
type HTTPStreamFactory struct {
	runningStream *uint
	wg            *sync.WaitGroup
	seq           *uint
	uniStreams    *map[streamKey]*httpStreamPair
	eventChan     *chan interface{}
}

// NewHTTPStreamFactory create a NewHTTPStreamFactory
func NewHTTPStreamFactory(out chan interface{}) HTTPStreamFactory {
	var f HTTPStreamFactory
	f.seq = new(uint)
	*f.seq = 0
	f.wg = new(sync.WaitGroup)
	f.uniStreams = new(map[streamKey]*httpStreamPair)
	*f.uniStreams = make(map[streamKey]*httpStreamPair)
	f.eventChan = new(chan interface{})
	*f.eventChan = out
	f.runningStream = new(uint)
	return f
}

// Wait for all stream exit
func (f HTTPStreamFactory) Wait() {
	f.wg.Wait()
}

// RunningStreamCount get the running stream count
func (f *HTTPStreamFactory) RunningStreamCount() uint {
	return *f.runningStream
}

func (f *HTTPStreamFactory) runStreamPair(streamPair *httpStreamPair) {
	f.wg.Add(1)
	*f.runningStream++

	defer f.wg.Done()
	defer func() { *f.runningStream-- }()
	streamPair.run()
}

// New creates a HTTPStreamFactory
func (f HTTPStreamFactory) New(netFlow, tcpFlow gopacket.Flow) (ret tcpassembly.Stream) {
	revkey := streamKey{netFlow.Reverse(), tcpFlow.Reverse()}
	streamPair, ok := (*f.uniStreams)[revkey]
	src := make(packetSource, 32)
	if ok {
		if streamPair.upStream == nil {
			panic("unbelievable!?")
		}
		delete(*f.uniStreams, revkey)
		key := streamKey{netFlow, tcpFlow}
		s := newHTTPStream(src, key)
		streamPair.downStream = &s
		ret = s
	} else {
		streamPair = new(httpStreamPair)
		streamPair.connSeq = *f.seq
		streamPair.sem = make(chan byte, 1)
		streamPair.eventChan = *f.eventChan
		key := streamKey{netFlow, tcpFlow}
		s := newHTTPStream(src, key)
		streamPair.upStream = &s
		(*f.uniStreams)[key] = streamPair
		*f.seq++
		go f.runStreamPair(streamPair)
		ret = s
	}
	return
}
