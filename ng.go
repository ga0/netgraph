package main

import (
    "flag"
    "fmt"
    "github.com/ga0/ng/ngnet"
    "github.com/ga0/ng/ngserver"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"
    "github.com/google/gopacket/tcpassembly"
    "os"
)

var eventChan chan interface{}
var pcapfile = flag.String("f", "", "Open pcap file")
var device = flag.String("i", "", "Device to capture")
var bindingPort = flag.Int("p", 9000, "Web server port")
var bpf = flag.String("bpf", "tcp port 80", "Berkeley Packet Filter")

func init() {
    eventChan = make(chan interface{}, 1024)
}

func packetSource() *gopacket.PacketSource {
    if *pcapfile != "" {
        fmt.Println("offline")
        if handle, err := pcap.OpenOffline(*pcapfile); err != nil {
            panic(err)
        } else {
            return gopacket.NewPacketSource(handle, handle.LinkType())
        }
    } else if *device != "" {
        handle, err := pcap.OpenLive(*device, 1024*1024, true, pcap.BlockForever)
        if err != nil {
            fmt.Println(err)
            os.Exit(-1)
        }
        if *bpf != "" {
            if err = handle.SetBPFFilter(*bpf); err != nil {
                fmt.Println("Failed to set BPF filter:", err)
                os.Exit(-1)
            }
        }
        return gopacket.NewPacketSource(handle, handle.LinkType())
    } else {
        fmt.Println("no packet to capture")
        os.Exit(-1)
    }
    return nil
}

func runNGNet(packetSource *gopacket.PacketSource) {
    f := ngnet.NewHttpStreamFactory(eventChan)
    pool := tcpassembly.NewStreamPool(f)
    assembler := tcpassembly.NewAssembler(pool)

    var count int = 0
    for packet := range packetSource.Packets() {
        count++

        net_layer := packet.NetworkLayer()
        trans_layer := packet.TransportLayer()

        if net_layer == nil {
            continue
        }
        if trans_layer == nil {
            continue
        }
        tcp, _ := trans_layer.(*layers.TCP)
        if tcp == nil {
            continue
        }

        //fmt.Println(packet.Metadata(), tcp, net_layer)

        assembler.AssembleWithTimestamp(
            net_layer.NetworkFlow(),
            tcp,
            packet.Metadata().CaptureInfo.Timestamp)
    }

    assembler.FlushAll()
    f.Wait()
    fmt.Println("Packet count: ", count)
}

func main() {
    flag.Parse()

    go runNGNet(packetSource())
    addr := fmt.Sprintf(":%d", *bindingPort)
    fmt.Println(addr)
    s := ngserver.NewNGServer(addr, "client", eventChan)
    s.Serve()
}
