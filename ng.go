package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/ga0/netgraph/ngnet"
	"github.com/ga0/netgraph/ngserver"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
	"github.com/google/gopacket/pcapgo"
	"github.com/google/gopacket/tcpassembly"
)

var eventChan chan interface{}
var inputPcap = flag.String("f", "", "Open pcap file")
var device = flag.String("i", "", "Device to capture, auto select one if no device provided")
var bindingPort = flag.Int("p", 9000, "Web server port")
var bpf = flag.String("bpf", "tcp port 80", "Berkeley Packet Filter")
var outputPcap = flag.String("o", "", "Output captured packet to pcap file")
var saveEvent = flag.Bool("s", false, "save network event in server")

func init() {
	flag.Parse()
	if *inputPcap != "" && *outputPcap != "" {
		log.Fatalln("set -f and -o at the same time")
	}
	if *inputPcap != "" && *device != "" {
		log.Fatalln("set -f and -i at the same time")
	}
	if *inputPcap != "" {
		*saveEvent = true
	}
	eventChan = make(chan interface{}, 1024)
}

func autoSelectDev() string {
	ifs, err := pcap.FindAllDevs()
	if err != nil {
		log.Fatalln(err)
	}
	var available []string
	for _, i := range ifs {
		addrFound := false
		var addrs []string
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
		return available[0]
	}
	return ""
}

func packetSource() *gopacket.PacketSource {
	if *inputPcap != "" {
		handle, err := pcap.OpenOffline(*inputPcap)
		if err != nil {
			log.Fatalln(err)
		}
		fmt.Printf("open pcap file \"%s\"\n", *inputPcap)
		return gopacket.NewPacketSource(handle, handle.LinkType())
	}

	if *device == "" {
		*device = autoSelectDev()
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
	fmt.Printf("open live on device \"%s\", bpf \"%s\", serves on port %d\n", *device, *bpf, *bindingPort)
	return gopacket.NewPacketSource(handle, handle.LinkType())
}

func runNGNet(packetSource *gopacket.PacketSource) {
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
	for packet := range packetSource.Packets() {
		if packet == nil {
			break
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
	}

	assembler.FlushAll()
	log.Println("Read pcap file complete")
	streamFactory.Wait()
	log.Println("Parse complete, packet count: ", count)
}

/*
   create client.go
*/
//go:generate python embed_html.py

func main() {
	go runNGNet(packetSource())
	addr := fmt.Sprintf(":%d", *bindingPort)
	s := ngserver.NewNGServer(addr, eventChan, *saveEvent)
	s.Serve()
}
