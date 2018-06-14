package ngnet

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/gopacket"
	"github.com/google/gopacket/tcpassembly"
)

var (
	httpRequestFirtLine  *regexp.Regexp
	httpResponseFirtLine *regexp.Regexp
)

func init() {
	httpRequestFirtLine = regexp.MustCompile(`([A-Z]+) (.+) (HTTP/.+)\r\n`)
	httpResponseFirtLine = regexp.MustCompile(`(HTTP/.+) (\d{3}) (.+)\r\n`)
}

type streamKey struct {
	net, tcp gopacket.Flow
}

func (k streamKey) String() string {
	return fmt.Sprintf("{%v:%v} -> {%v:%v}", k.net.Src(), k.tcp.Src(), k.net.Dst(), k.tcp.Dst())
}

type httpStream struct {
	reader *StreamReader
	bytes  *uint64
	key    streamKey
	bad    *bool
}

func newHTTPStream(src packetSource, key streamKey) httpStream {
	var s httpStream
	s.reader = NewStreamReader(src)
	s.bytes = new(uint64)
	s.key = key
	s.bad = new(bool)
	return s
}

// Reassembled is called by tcpassembly
func (s httpStream) Reassembled(rs []tcpassembly.Reassembly) {
	for _, r := range rs {
		if *s.bad {
			break
		}
		if r.Skip != 0 {
			*s.bad = true
			break
		}
		*s.bytes += uint64(len(r.Bytes))
		select {
		case <-s.reader.stopCh:
			*s.bad = true
			return
		case s.reader.src <- r:
		}
	}
}

// ReassemblyComplete is called by tcpassembly
func (s httpStream) ReassemblyComplete() {
	close(s.reader.src)
}

func (s *httpStream) getRequestLine() (method string, uri string, version string) {
	bytes, err := s.reader.ReadUntil([]byte("\r\n"))
	if err != nil {
		panic("Cannot read request line, err=" + err.Error())
	}
	line := string(bytes)
	r := httpRequestFirtLine.FindStringSubmatch(line)
	if len(r) != 4 {
		panic("Bad HTTP Request: " + line)
	}

	method = r[1]
	uri = r[2]
	version = r[3]
	return
}

func (s *httpStream) getResponseLine() (version string, code uint, reason string) {
	bytes, err := s.reader.ReadUntil([]byte("\r\n"))
	if err != nil {
		panic("Cannot read response line, err=" + err.Error())
	}
	line := string(bytes)
	r := httpResponseFirtLine.FindStringSubmatch(line)
	if len(r) != 4 {
		panic("Bad HTTP Response: " + line)
	}

	version = r[1]
	var code64 uint64
	code64, err = strconv.ParseUint(r[2], 10, 32)
	if err != nil {
		panic("Bad HTTP Response: " + line + ", err=" + err.Error())
	}
	code = uint(code64)
	reason = r[3]
	return
}

func (s *httpStream) getHeaders() (headers []HTTPHeaderItem) {
	d, err := s.reader.ReadUntil([]byte("\r\n\r\n"))
	if err != nil {
		panic("Cannot read headers, err=" + err.Error())
	}
	data := string(d[:len(d)-4])
	for i, line := range strings.Split(data, "\r\n") {
		p := strings.Index(line, ":")
		if p == -1 {
			panic(fmt.Sprintf("Bad http header (line %d): %s", i, data))
		}
		var h HTTPHeaderItem
		h.Name = line[:p]
		h.Value = strings.Trim(line[p+1:], " ")
		headers = append(headers, h)
	}
	return
}

func (s *httpStream) getChunked() []byte {
	var body []byte
	for {
		buf, err := s.reader.ReadUntil([]byte("\r\n"))
		if err != nil {
			panic("Cannot read chuncked content, err=" + err.Error())
		}
		l := string(buf)
		l = strings.Trim(l[:len(l)-2], " ")
		blockSize, err := strconv.ParseInt(l, 16, 32)
		if err != nil {
			panic("bad chunked block length: " + l + ", err=" + err.Error())
		}

		buf, err = s.reader.Next(int(blockSize))
		body = append(body, buf...)
		if err != nil {
			panic("Cannot read chuncked content, err=" + err.Error())
		}
		buf, err = s.reader.Next(2)
		if err != nil {
			panic("Cannot read chuncked content, err=" + err.Error())
		}
		CRLF := string(buf)
		if CRLF != "\r\n" {
			panic("Bad chunked block data")
		}

		if blockSize == 0 {
			break
		}
	}
	return body
}

func (s *httpStream) getFixedLengthContent(contentLength int) []byte {
	body, err := s.reader.Next(contentLength)
	if err != nil {
		panic("Cannot read content, err=" + err.Error())
	}
	return body
}

func getContentInfo(hs []HTTPHeaderItem) (contentLength int, contentEncoding string, contentType string, chunked bool) {
	for _, h := range hs {
		lowerName := strings.ToLower(h.Name)
		if lowerName == "content-length" {
			var err error
			contentLength, err = strconv.Atoi(h.Value)
			if err != nil {
				panic("Content-Length error: " + h.Value + ", err=" + err.Error())
			}
		} else if lowerName == "transfer-encoding" && h.Value == "chunked" {
			chunked = true
		} else if lowerName == "content-encoding" {
			contentEncoding = h.Value
		} else if lowerName == "content-type" {
			contentType = h.Value
		}
	}
	return
}

func (s *httpStream) getBody(method string, headers []HTTPHeaderItem, isRequest bool) (body []byte) {
	contentLength, contentEncoding, _, chunked := getContentInfo(headers)
	if (contentLength == 0 && !chunked) || (!isRequest && method == "HEAD") {
		return
	}

	if chunked {
		body = s.getChunked()
	} else {
		body = s.getFixedLengthContent(contentLength)
	}

	var uncompressedBody []byte
	var err error
	// TODO: more compress type should be supported
	if contentEncoding == "gzip" {
		buffer := bytes.NewBuffer(body)
		zipReader, _ := gzip.NewReader(buffer)
		uncompressedBody, err = ioutil.ReadAll(zipReader)
		defer zipReader.Close()
		if err != nil {
			body = []byte("(gzip data uncompress error)")
		} else {
			body = uncompressedBody
		}
	}
	return
}
