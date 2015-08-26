package ngnet

import (
    "bytes"
    "errors"
    "fmt"
    "github.com/google/gopacket"
    "github.com/google/gopacket/tcpassembly"
    "regexp"
    "strconv"
    "strings"
    "sync"
    "time"
)

type Direction byte

const (
    UpDirection   Direction = 1
    DownDirection Direction = 2
)

var (
    httpRequestFirtLine  *regexp.Regexp
    httpResponseFirtLine *regexp.Regexp
)

func init() {
    httpRequestFirtLine = regexp.MustCompile("([A-Z]*) (.*) (.*)")
    httpResponseFirtLine = regexp.MustCompile("(.*) (\\d*) (.*)")
}

type Packet struct {
    Meta string
    Data []byte
}

type PacketSource chan tcpassembly.Reassembly

type HttpStreamReader struct {
    src           PacketSource
    buffer        *bytes.Buffer
    seq           uint
    lastTimeStamp float64
}

func NewHttpStreamReader(s PacketSource, seq uint) *HttpStreamReader {
    f := new(HttpStreamReader)
    f.src = s
    f.buffer = bytes.NewBuffer([]byte(""))
    f.seq = seq
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
    } else {
        return t.Sub(startTimestamp).Seconds()
    }
}

func (s *HttpStreamReader) fillBuffer() error {
    if dataBlock, more := <-s.src; more {
        s.buffer.Write(dataBlock.Bytes)
        s.lastTimeStamp = relativeTimestamp(dataBlock.Seen)
        return nil
    } else {
        return errors.New("EOF")
    }
}

func (f *HttpStreamReader) ReadUntil(delim []byte) ([]byte, error) {
    var p int
    for {
        if p = bytes.Index(f.buffer.Bytes(), delim); p == -1 {
            if err := f.fillBuffer(); err != nil {
                return nil, err
            }
        } else {
            break
        }
    }
    return f.buffer.Next(p + len(delim)), nil
}

func (f HttpStreamReader) Next(n int) ([]byte, error) {
    for f.buffer.Len() < n {
        if err := f.fillBuffer(); err != nil {
            return nil, errors.New("EOF")
        }
    }
    return f.buffer.Next(n), nil
}

type HttpStream struct {
    reader     *HttpStreamReader
    streamSeq  uint
    requestSeq uint
    wg         *sync.WaitGroup
    eventChan  chan interface{}
    sem        chan byte
    direction  Direction
    bad        *bool
}

func (s HttpStream) getHeader() (m HttpMessage, more bool) {
    d, err := s.reader.ReadUntil([]byte("\r\n\r\n"))
    if err != nil {
        more = false
        return
    }
    data := string(d[:len(d)-4])
    more = true
    for i, line := range strings.Split(data, "\r\n") {
        if i == 0 {
            if s.direction == DownDirection {
                if line[:5] != "HTTP/" {
                    panic("Bad HTTP Response: \n" + line)
                }
                r := httpResponseFirtLine.FindStringSubmatch(line)
                if len(r) != 4 {
                    panic("Bad HTTP Response: \n" + line)
                }
                e := NewHttpResponseEvent()
                e.Type = "HttpResponse"
                e.Version = r[1]
                e.Code = r[2]
                e.Reason = r[3]
                m = e
            } else if s.direction == UpDirection {
                r := httpRequestFirtLine.FindStringSubmatch(line)
                if len(r) != 4 {
                    panic("Bad HTTP Request: \n" + line)
                }
                e := NewHttpRequestEvent()
                e.Type = "HttpRequest"
                e.Method = r[1]
                e.Uri = r[2]
                e.Version = r[3]
                m = e
            }
            m.SetStreamSeq(s.streamSeq)
            m.SetTimestamp(s.reader.lastTimeStamp)
            m.SetEndTimestamp(s.reader.lastTimeStamp)
            continue
        }
        p := strings.Index(line, ":")
        if p == -1 {
            panic(fmt.Sprintf("bad http header (line %d): %s", i, data))
        }
        var h HttpHeaderItem
        h.Name = line[:p]
        h.Value = strings.Trim(line[p+1:], " ")
        m.AddHeader(h)
    }
    return
}

func (s HttpStream) processChunked() ([]byte, bool) {
    var body []byte
    for {
        buf, err := s.reader.ReadUntil([]byte("\r\n"))
        if err != nil {
            return body, false
        }
        l := string(buf)
        l = strings.Trim(l[:len(l)-2], " ")
        blockLength, err := strconv.ParseInt(l, 16, 32)

        if err != nil {
            panic("bad chunked block length: " + l + "\n" + err.Error())
        }

        buf, err = s.reader.Next(int(blockLength))
        body = append(body, buf...)
        if err != nil {
            return body, false
        }
        buf, err = s.reader.Next(2)
        if err != nil {
            return body, false
        }
        CRLF := string(buf)
        if CRLF != "\r\n" {
            panic("bad chunked block data")
        }

        if blockLength == 0 {
            break
        }
    }
    return body, true
}

func (f HttpStream) processContentLength(contentLength int) ([]byte, bool) {
    body, err := f.reader.Next(contentLength)
    return body, err == nil
}

func (f HttpStream) processBody(contentLength int, chunked bool) (body []byte, more bool) {
    if chunked {
        body, more = f.processChunked()
    } else {
        body, more = f.processContentLength(contentLength)
    }
    return
}

func GetHttpSize(hs []HttpHeaderItem) (contentLength int, chunked bool) {
    for _, h := range hs {
        if h.Name == "Content-Length" {
            var err error
            contentLength, err = strconv.Atoi(h.Value)
            if err != nil {
                panic("Content-Length error: " + h.Value)
            }
        } else if h.Name == "Transfer-Encoding" && h.Value == "chunked" {
            chunked = true
        }
    }
    return
}

func (s HttpStream) Process() {
    defer func() {
        if r := recover(); r != nil {
            *s.bad = true
            //close(s.reader.src)
            fmt.Println("HttpStream error: ", r)
        }
    }()
    defer s.wg.Done()
    for {
        if s.direction == DownDirection {
            <-s.sem
        } else if s.direction == UpDirection {
            s.sem <- 1
        }

        m, more := s.getHeader()
        if !more {
            break
        }

        contentLength, chunked := GetHttpSize(m.Header())
        if contentLength == 0 && !chunked {
            s.eventChan <- m
            s.requestSeq++
            continue
        }

        body, more := s.processBody(contentLength, chunked)
        if !more {
            break
        }
        m.SetBody(string(body))
        m.SetEndTimestamp(s.reader.lastTimeStamp)
        s.eventChan <- m
        s.requestSeq++
    }
}

func NewHttpStream(src PacketSource, meta string, seq uint, wg *sync.WaitGroup, eventChan chan interface{}, sem chan byte, dir Direction) HttpStream {
    var s HttpStream
    s.reader = NewHttpStreamReader(src, seq)
    s.streamSeq = seq
    s.wg = wg
    s.eventChan = eventChan
    s.sem = sem
    s.direction = dir
    s.bad = new(bool)
    return s
}

func (s HttpStream) Reassembled(rs []tcpassembly.Reassembly) {
    for _, r := range rs {
        if *s.bad {
            break
        }
        s.reader.src <- r
    }
}

func (s HttpStream) ReassemblyComplete() {
    close(s.reader.src)
}

type HttpStreamPair struct {
    upStream   *HttpStream
    downStream *HttpStream
    sem        chan byte
    connSeq    uint
}

type HttpStreamFactory struct {
    wg         *sync.WaitGroup
    seq        *uint
    uniStreams *map[string]*HttpStreamPair
    eventChan  *chan interface{}
}

func NewHttpStreamFactory(out chan interface{}) HttpStreamFactory {
    var f HttpStreamFactory
    f.seq = new(uint)
    *f.seq = 0
    f.wg = new(sync.WaitGroup)
    f.uniStreams = new(map[string]*HttpStreamPair)
    *f.uniStreams = make(map[string]*HttpStreamPair)
    f.eventChan = new(chan interface{})
    *f.eventChan = out
    return f
}

func (f HttpStreamFactory) Wait() {
    f.wg.Wait()
}

func (f HttpStreamFactory) New(netFlow, tcpFlow gopacket.Flow) (ret tcpassembly.Stream) {
    revkey := fmt.Sprintf("%v:%v->%v:%v",
        netFlow.Dst(),
        tcpFlow.Dst(),
        netFlow.Src(),
        tcpFlow.Src())
    streamPair, ok := (*f.uniStreams)[revkey]
    src := make(PacketSource)
    if ok {
        if streamPair.upStream == nil {
            panic("fuck")
        }
        delete(*f.uniStreams, revkey)
        s := NewHttpStream(src, "", streamPair.connSeq, f.wg, *f.eventChan, streamPair.sem, DownDirection)
        streamPair.downStream = &s
        ret = s
        go s.Process()
    } else {
        streamPair = new(HttpStreamPair)
        streamPair.connSeq = *f.seq
        streamPair.sem = make(chan byte, 1)
        key := fmt.Sprintf("%v:%v->%v:%v",
            netFlow.Src(),
            tcpFlow.Src(),
            netFlow.Dst(),
            tcpFlow.Dst())
        s := NewHttpStream(src, "", streamPair.connSeq, f.wg, *f.eventChan, streamPair.sem, UpDirection)
        streamPair.upStream = &s
        (*f.uniStreams)[key] = streamPair
        *f.seq++
        //fmt.Printf("#%d Connect %s\n", streamPair.connSeq, key)
        go s.Process()
        ret = s
    }
    f.wg.Add(1)
    return
}
