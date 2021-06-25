package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"nggraph/handlers"
	"nggraph/ngdns"

	"github.com/ga0/netgraph/ngnet"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/google/gopacket/tcpassembly"
)

var device = flag.String("i", "", "Device to capture, auto select one if no device provided")
var bpf = flag.String("bpf", "tcp port 80", "Set berkeley packet filter")

var outputHTTP = flag.String("o", "", "Write HTTP request/response to file")
var inputPcap = flag.String("input-pcap", "", "Open pcap file")
var outputPcap = flag.String("output-pcap", "", "Write captured packet to a pcap file")
var requestOnly = flag.Bool("output-request-only", true, "Write HTTP request only, drop response")

var bindingPort = flag.Int("p", 9000, "Web server port. If the port is set to '0', the server will not run.")
var saveEvent = flag.Bool("s", false, "Save HTTP event in server")

var verbose = flag.Bool("v", true, "Show more message")

var dnsDevice = flag.String("dnsDevice", "", "Set DNS device to monitoring")

var (
	devName    string
	es_index   string
	es_docType string
	es_server  string
	err        error
	handle     *pcap.Handle
	InetAddr   string
	SrcIP      string
	DstIP      string
)

// NGHTTPEventHandler handle HTTP events
type NGHTTPEventHandler interface {
	PushEvent(interface{})
	Wait()
}

var localHandlers []NGHTTPEventHandler

func init() {
	flag.Parse()
	if *inputPcap != "" && *outputPcap != "" {
		log.Fatalln("ERROR: set -input-pcap and -output-pcap at the same time")
	}
	if *inputPcap != "" && *device != "" {
		log.Fatalln("ERROR: set -input-pcap and -i at the same time")
	}
	if !*verbose {
		log.SetOutput(ioutil.Discard)
	}
	if *inputPcap != "" {
		*saveEvent = true
	}
}

func initEventHandlers() {
	if *bindingPort != 0 {
		addr := fmt.Sprintf(":%d", *bindingPort)
		ngserver := NewNGServer(addr, *saveEvent)
		ngserver.Serve()
		localHandlers = append(localHandlers, ngserver)
	}

	if *outputHTTP != "" {
		p := NewEventPrinter(*outputHTTP)
		localHandlers = append(localHandlers, p)
	}

	dnsEvent := handlers.NewDnsEventPrinterHandler("")
	localHandlers = append(localHandlers, dnsEvent)
}

func autoSelectDev() (string, string) {
	ifs, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatalln(err)
	}
	var available []string
	var addrs []string
	for _, i := range ifs {
		addrFound := false
		for _, addr := range i.Addresses {
			if addr.IP.IsLoopback() ||
				addr.IP.IsMulticast() ||
				addr.IP.IsUnspecified() ||
				addr.IP.IsLinkLocalUnicast() {
				continue
			}
			addrFound = true
			addrs = append(addrs, addr.IP.String())
		}
		if addrFound {
			available = append(available, i.Name)
		}
	}
	if len(available) > 0 {
		return available[1], addrs[1]
	}
	return "", ""
}

func packetSource() *gopacket.PacketSource {
	if *inputPcap != "" {
		handle, err := pcap.OpenOffline(*inputPcap)
		if err != nil {
			log.Fatalln(err)
		}
		log.Printf("open pcap file \"%s\"\n", *inputPcap)
		return gopacket.NewPacketSource(handle, handle.LinkType())
	}

	if *device == "" {
		*device, _ = autoSelectDev()
		if *device == "" {
			log.Fatalln("no device to capture")
		}
	}

	handle, err := pcap.OpenLive(*device, 1024*1024, true, pcap.BlockForever)
	if err != nil {
		log.Fatalln(err)
	}
	if *bpf != "" {
		if err = handle.SetBPFFilter(*bpf); err != nil {
			log.Fatalln("Failed to set BPF filter:", err)
		}
	}
	log.Printf("open live on device \"%s\", bpf \"%s\"\n", *device, *bpf)
	return gopacket.NewPacketSource(handle, handle.LinkType())
}

func runNGNet(packetSource *gopacket.PacketSource, eventChan chan<- interface{}) {
	streamFactory := ngnet.NewHTTPStreamFactory(eventChan)
	pool := tcpassembly.NewStreamPool(streamFactory)
	assembler := tcpassembly.NewAssembler(pool)

	var pcapWriter *pcapgo.Writer
	if *outputPcap != "" {
		outPcapFile, err := os.Create(*outputPcap)
		if err != nil {
			log.Fatalln(err)
		}
		defer outPcapFile.Close()
		pcapWriter = pcapgo.NewWriter(outPcapFile)
		pcapWriter.WriteFileHeader(65536, layers.LinkTypeEthernet)
	}

	var count uint
	ticker := time.Tick(time.Minute)
	var lastPacketTimestamp time.Time

LOOP:
	for {
		select {
		case packet := <-packetSource.Packets():
			if packet == nil {
				break LOOP
			}

			count++
			netLayer := packet.NetworkLayer()
			if netLayer == nil {
				continue
			}
			transLayer := packet.TransportLayer()
			if transLayer == nil {
				continue
			}
			tcp, _ := transLayer.(*layers.TCP)
			if tcp == nil {
				continue
			}

			if pcapWriter != nil {
				pcapWriter.WritePacket(packet.Metadata().CaptureInfo, packet.Data())
			}

			assembler.AssembleWithTimestamp(
				netLayer.NetworkFlow(),
				tcp,
				packet.Metadata().CaptureInfo.Timestamp)

			lastPacketTimestamp = packet.Metadata().CaptureInfo.Timestamp
		case <-ticker:
			assembler.FlushOlderThan(lastPacketTimestamp.Add(time.Minute * -2))
		}
	}

	assembler.FlushAll()
	log.Println("Read pcap file complete")
	streamFactory.Wait()
	log.Println("Parse complete, packet count: ", count)

	close(eventChan)
}

// EventPrinter print HTTP events to file or stdout
type EventPrinter struct {
	file *os.File
}

// NewEventPrinter creates EventPrinter
func NewEventPrinter(name string) *EventPrinter {
	p := new(EventPrinter)
	var err error
	if name == "stdout" {
		p.file = os.Stdout
	} else {
		p.file, err = os.OpenFile(name, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			log.Fatalln("Cannot open file ", name)
		}
	}

	return p
}

func (p *EventPrinter) printHTTPRequestEvent(req ngnet.HTTPRequestEvent) {
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

func (p *EventPrinter) printHTTPResponseEvent(resp ngnet.HTTPResponseEvent) {
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

// PushEvent implements the function of interface NGHTTPEventHandler
func (p *EventPrinter) PushEvent(e interface{}) {
	switch v := e.(type) {
	case ngnet.HTTPRequestEvent:
		p.printHTTPRequestEvent(v)
	case ngnet.HTTPResponseEvent:
		if !*requestOnly {
			p.printHTTPResponseEvent(v)
		}
	default:
		log.Printf("Unknown event: %v", e)
	}
}

// Wait implements the function of interface NGHTTPEventHandler
func (p *EventPrinter) Wait() {}

func runEventHandler(eventChan <-chan interface{}) {
	for e := range eventChan {
		if e == nil {
			break
		}
		for _, h := range localHandlers {
			h.PushEvent(e)
		}
	}

	for _, h := range localHandlers {
		h.Wait()
	}
}

func runNGDns(eventChan chan<- interface{}) {
	var eth layers.Ethernet
	var ip4 layers.IPv4
	var ip6 layers.IPv6
	var tcp layers.TCP
	var udp layers.UDP
	var dns layers.DNS

	var payload gopacket.Payload

	devName, InetAddr = autoSelectDev()

	handle, err = pcap.OpenLive(devName, 1600, false, pcap.BlockForever)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	// Set filter
	var filter string = "port 53 and src host " + InetAddr
	fmt.Println("    Filter: ", filter)
	err := handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	parser := gopacket.NewDecodingLayerParser(layers.LayerTypeEthernet, &eth, &ip4, &ip6, &tcp, &udp, &dns, &payload)

	decodedLayers := make([]gopacket.LayerType, 0, 10)
	for {
		data, _, err := handle.ReadPacketData()
		if err != nil {
			fmt.Println("Error reading packet data: ", err)
			continue
		}

		err = parser.DecodeLayers(data, &decodedLayers)
		for _, typ := range decodedLayers {
			switch typ {
			case layers.LayerTypeIPv4:
				SrcIP = ip4.SrcIP.String()
				DstIP = ip4.DstIP.String()
			case layers.LayerTypeIPv6:
				SrcIP = ip6.SrcIP.String()
				DstIP = ip6.DstIP.String()
			case layers.LayerTypeDNS:
				dnsOpCode := int(dns.OpCode)
				dnsResponseCode := int(dns.ResponseCode)
				dnsANCount := int(dns.ANCount)

				if (dnsANCount == 0 && dnsResponseCode == 0) || (dnsANCount > 0) {

					dnsEvent := ngdns.DNSEvent{
						Type:         "DNSEvent",
						ID:           dns.ID,
						QR:           dns.QR,
						OpCode:       dnsOpCode,
						ResponseCose: dns.ResponseCode.String(),
						Questions:    make([]ngdns.DNSQuestion, 0),
						Answers:      make([]ngdns.DNSAnswer, 0),
					}

					for _, dnsQuestion := range dns.Questions {
						dnsEvent.Questions = append(dnsEvent.Questions, ngdns.DNSQuestion{
							Name:  string(dnsQuestion.Name),
							Type:  dnsQuestion.Type.String(),
							Class: dnsQuestion.Class.String(),
						})
					}

					for _, dnsAnswer := range dns.Answers {
						dnsEvent.Answers = append(dnsEvent.Answers, ngdns.DNSAnswer{
							Name:       string(dnsAnswer.Name),
							Type:       dnsAnswer.Type.String(),
							Class:      dnsAnswer.Class.String(),
							DataLength: dnsAnswer.DataLength,
						})
					}

					eventChan <- dnsEvent
				}

			}
		}

		if err != nil {
			fmt.Println("  Error encountered:", err)
		}
	}
}

func main() {
	initEventHandlers()
	eventChan := make(chan interface{}, 1024)
	go runNGNet(packetSource(), eventChan)
	go runNGDns(eventChan)
	runEventHandler(eventChan)
}
