package handlers

import (
	"fmt"
	"log"
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

func (p *DnsEventPrinterHandler) printDNSRequestEvent(req ngdns.DNSRequestEvent) {
	fmt.Fprintf(p.file, "[%s] #%d Request %s->%s\r\n",
		req.Start.Format("2006-01-02 15:04:05.000"), req.StreamSeq, req.ClientAddr, req.ServerAddr)
	fmt.Fprintf(p.file, "%s %s %s\r\n", req.Method, req.URI, req.Version)
	for _, h := range req.Headers {
		fmt.Fprintf(p.file, "%s: %s\r\n", h.Name, h.Value)
	}

	fmt.Fprintf(p.file, "\r\ncontent(%d)", len(req.Body))
	if len(req.Body) > 0 {
		fmt.Fprintf(p.file, "%s", req.Body)
	}
	fmt.Fprintf(p.file, "\r\n\r\n")
}

func (p *DnsEventPrinterHandler) printDNSResponseEvent(resp ngdns.DNSResponseEvent) {
	fmt.Fprintf(p.file, "[%s] #%d Response %s<-%s\r\n",
		resp.Start.Format("2006-01-02 15:04:05.000"), resp.StreamSeq, resp.ClientAddr, resp.ServerAddr)
	fmt.Fprintf(p.file, "%s %d %s\r\n", resp.Version, resp.Code, resp.Reason)
	for _, h := range resp.Headers {
		fmt.Fprintf(p.file, "%s: %s\r\n", h.Name, h.Value)
	}

	fmt.Fprintf(p.file, "\r\ncontent(%d)", len(resp.Body))
	if len(resp.Body) > 0 {
		fmt.Fprintf(p.file, "%s", resp.Body)
	}
	fmt.Fprintf(p.file, "\r\n\r\n")
}

// PushEvent implements the function of interface Handlers
func (p *DnsEventPrinterHandler) PushEvent(e interface{}) {
	switch v := e.(type) {
	case ngdns.DNSRequestEvent:
		p.printDNSRequestEvent(v)
	case ngdns.DNSResponseEvent:
		p.printDNSResponseEvent(v)
	default:
		log.Printf("Unknown event: %v", e)
	}
}

// Wait implements the function of interface
func (p *DnsEventPrinterHandler) Wait() {}
