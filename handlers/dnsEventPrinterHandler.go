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
	fmt.Fprintf(p.file, "\n\n  DNS Event\n")
	fmt.Fprintf(p.file, "    ID: %d\n", req.ID)
	fmt.Fprintf(p.file, "    QR: %t\n", req.QR)
	fmt.Fprintf(p.file, "    OpCode: %d\n", req.OpCode)
	fmt.Fprintf(p.file, "    ResponseCode: %s\n", req.ResponseCode)

	for _, dnsQuestion := range req.Questions {
		fmt.Fprintf(p.file, "  DNS Question\n")
		fmt.Fprintf(p.file, "    Name: %s\n", dnsQuestion.Name)
		fmt.Fprintf(p.file, "    Type: %s\n", dnsQuestion.Type)
		fmt.Fprintf(p.file, "    Class: %s\n", dnsQuestion.Class)
	}

	for _, dnsAnswer := range req.Answers {
		fmt.Fprintf(p.file, "  DNS Answer\n")
		fmt.Fprintf(p.file, "    Name: %s\n", string(dnsAnswer.Name))
		fmt.Fprintf(p.file, "    Type: %s\n", dnsAnswer.Type)
		fmt.Fprintf(p.file, "    Class: %s\n", dnsAnswer.Class)
		fmt.Fprintf(p.file, "    Data length: %d\n", dnsAnswer.DataLength)
	}
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
