package ngnet

import (
	"bytes"
	"errors"
	"time"

	"github.com/google/gopacket/tcpassembly"
)

type packetSource chan tcpassembly.Reassembly

// StreamReader read data from tcp stream
type StreamReader struct {
	src      packetSource
	stopCh   chan interface{}
	buffer   *bytes.Buffer
	lastSeen time.Time
}

// NewStreamReader create a new StreamReader
func NewStreamReader(s packetSource) *StreamReader {
	r := new(StreamReader)
	r.stopCh = make(chan interface{})
	r.src = s
	r.buffer = bytes.NewBuffer([]byte(""))
	return r
}

func (s *StreamReader) fillBuffer() error {
	if dataBlock, more := <-s.src; more {
		s.buffer.Write(dataBlock.Bytes)
		s.lastSeen = dataBlock.Seen
		return nil
	}
	return errors.New("EOF")
}

// ReadUntil read bytes until delim
func (s *StreamReader) ReadUntil(delim []byte) ([]byte, error) {
	var p int
	for {
		if p = bytes.Index(s.buffer.Bytes(), delim); p == -1 {
			if err := s.fillBuffer(); err != nil {
				return nil, err
			}
		} else {
			break
		}
	}
	return s.buffer.Next(p + len(delim)), nil
}

// Next read n bytes from stream
func (s *StreamReader) Next(n int) ([]byte, error) {
	for s.buffer.Len() < n {
		if err := s.fillBuffer(); err != nil {
			return nil, err
		}
	}
	return s.buffer.Next(n), nil
}
