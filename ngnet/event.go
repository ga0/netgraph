package ngnet

import ()

type NetEvent interface {
    SetTimestamp(float64)
    SetType(string)
    SetStreamSeq(uint)
    SetEndTimestamp(float64)
}

type NetEventBase struct {
    Timestamp    float64
    StreamSeq    uint
    Type         string
    EndTimestamp float64
}

func (ne *NetEventBase) SetTimestamp(t float64)    { ne.Timestamp = t }
func (ne *NetEventBase) SetType(t string)          { ne.Type = t }
func (ne *NetEventBase) SetStreamSeq(seq uint)     { ne.StreamSeq = seq }
func (ne *NetEventBase) SetEndTimestamp(t float64) { ne.EndTimestamp = t }

type HttpMessage interface {
    NetEvent
    SetVersion(string)
    SetBody(string)
    AddHeader(HttpHeaderItem)
    Header() []HttpHeaderItem
}

type HttpMessageBase struct {
    NetEventBase
    Version string
    Headers []HttpHeaderItem
    Body    string
}

func (hm *HttpMessageBase) SetVersion(v string)        { hm.Version = v }
func (hm *HttpMessageBase) SetBody(body string)        { hm.Body = body }
func (hm *HttpMessageBase) GetBody() string            { return hm.Body }
func (hm *HttpMessageBase) AddHeader(h HttpHeaderItem) { hm.Headers = append(hm.Headers, h) }
func (hm *HttpMessageBase) Header() []HttpHeaderItem   { return hm.Headers }

type HttpHeaderItem struct {
    Name  string
    Value string
}

type HttpRequestEvent struct {
    HttpMessageBase
    Method string
    Uri    string
}

type HttpResponseEvent struct {
    HttpMessageBase
    Code   string
    Reason string
}
