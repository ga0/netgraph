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
    SetBody([]byte)
    AddHeader(HttpHeaderItem)
    Header() []HttpHeaderItem
}

type HttpMessageBase struct {
    Version string
    Headers []HttpHeaderItem
    Body    []byte
}

func (hm *HttpMessageBase) SetVersion(v string)        { hm.Version = v }
func (hm *HttpMessageBase) SetBody(body []byte)        { hm.Body = body }
func (hm *HttpMessageBase) AddHeader(h HttpHeaderItem) { hm.Headers = append(hm.Headers, h) }
func (hm *HttpMessageBase) Header() []HttpHeaderItem   { return hm.Headers }

type HttpHeaderItem struct {
    Name  string
    Value string
}

type HttpRequestEvent struct {
    *NetEventBase
    *HttpMessageBase
    Method string
    Uri    string
}

func NewHttpRequestEvent() (e HttpRequestEvent) {
    e.NetEventBase = new(NetEventBase)
    e.HttpMessageBase = new(HttpMessageBase)
    return
}

type HttpResponseEvent struct {
    *NetEventBase
    *HttpMessageBase
    Code   string
    Reason string
}

func NewHttpResponseEvent() (e HttpResponseEvent) {
    e.NetEventBase = new(NetEventBase)
    e.HttpMessageBase = new(HttpMessageBase)
    return
}
