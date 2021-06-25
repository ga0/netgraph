package handlers

import (
	"fmt"
	"os"

	"nggraph/ngdns"
)

// DnsEventPrinterHandler print DNS events to file or stdout
type DnsEventPrinterHandler struct {
	file *os.File
}

// NewDnsEventPrinterHandler creates DnsEventPrinterHandler
func NewDnsEventPrinterHandler(name string) *DnsEventPrinterHandler {
	p := new(DnsEventPrinterHandler)
	p.file = os.Stdout

	return p
}

func (p *DnsEventPrinterHandler) printDNSEvent(req ngdns.DNSEvent) {
	fmt.Fprintf(p.file, "[%s] #%s Request %s->%d\r\n",
		req.Start.Format("2006-01-02 15:04:05.000"), req.StreamSeq, req.Type, req.ID)
	// fmt.Fprintf(p.file, "%s %s %s\r\n", req.Method, req., req.QR)
	// for _, h := range req.Headers {
	// 	fmt.Fprintf(p.file, "%s: %s\r\n", h.Name, h.Value)
	// }

	// fmt.Fprintf(p.file, "\r\ncontent(%d)", len(req.Body))
	// if len(req.Body) > 0 {
	// 	fmt.Fprintf(p.file, "%s", req.Body)
	// }
	fmt.Fprintf(p.file, "\r\n\r\n")
}

// PushEvent implements the function of interface Handlers
func (p *DnsEventPrinterHandler) PushEvent(e interface{}) {
	switch v := e.(type) {
	case ngdns.DNSEvent:
		p.printDNSEvent(v)
	}
}

// Wait implements the function of interface
func (p *DnsEventPrinterHandler) Wait() {}
