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
	src           packetSource
	buffer        *bytes.Buffer
	lastTimeStamp float64
}

// NewStreamReader create a new StreamReader
func NewStreamReader(s packetSource) *StreamReader {
	f := new(StreamReader)
	f.src = s
	f.buffer = bytes.NewBuffer([]byte(""))
	return f
}

var (
	startTimestampSet bool
	startTimestamp    time.Time
)

func relativeTimestamp(t time.Time) float64 {
	if !startTimestampSet {
		startTimestamp = t
		startTimestampSet = true
		return 0.0
	}
	return t.Sub(startTimestamp).Seconds()
}

func (s *StreamReader) fillBuffer() error {
	if dataBlock, more := <-s.src; more {
		s.buffer.Write(dataBlock.Bytes)
		s.lastTimeStamp = relativeTimestamp(dataBlock.Seen)
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
