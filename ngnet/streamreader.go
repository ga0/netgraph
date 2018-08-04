package ngnet

import (
	"bytes"
	"errors"
	"time"
)

// StreamDataBlock is copied from tcpassembly.Reassembly
type StreamDataBlock struct {
	Bytes []byte
	Seen  time.Time
}

// NewStreamDataBlock create a new StreamDataBlock
func NewStreamDataBlock(bytes []byte, seen time.Time) *StreamDataBlock {
	b := new(StreamDataBlock)
	b.Bytes = make([]byte, len(bytes))
	copy(b.Bytes, bytes[:])
	b.Seen = seen
	return b
}

// StreamReader read data from tcp stream
type StreamReader struct {
	src      chan *StreamDataBlock
	stopCh   chan interface{}
	buffer   *bytes.Buffer
	lastSeen time.Time
}

// NewStreamReader create a new StreamReader
func NewStreamReader() *StreamReader {
	r := new(StreamReader)
	r.stopCh = make(chan interface{})
	r.buffer = bytes.NewBuffer([]byte(""))
	r.src = make(chan *StreamDataBlock, 32)
	return r
}

func (s *StreamReader) fillBuffer() error {
	if dataBlock, ok := <-s.src; ok {
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
	dst := make([]byte, n)
	copy(dst, s.buffer.Next(n))
	return dst, nil
}
