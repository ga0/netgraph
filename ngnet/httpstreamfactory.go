package ngnet

import (
	"sync"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
)

// HTTPStreamFactory implements StreamFactory interface for tcpassembly
type HTTPStreamFactory struct {
	runningStream *uint
	wg            *sync.WaitGroup
	seq           *uint
	uniStreams    *map[streamKey]*httpStreamPair
	eventChan     chan<- interface{}
}

// NewHTTPStreamFactory create a NewHTTPStreamFactory
func NewHTTPStreamFactory(out chan<- interface{}) HTTPStreamFactory {
	var f HTTPStreamFactory
	f.seq = new(uint)
	*f.seq = 0
	f.wg = new(sync.WaitGroup)
	f.uniStreams = new(map[streamKey]*httpStreamPair)
	*f.uniStreams = make(map[streamKey]*httpStreamPair)
	f.eventChan = out
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
		streamPair = newHTTPStreamPair(*f.seq, f.eventChan)
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
