package ngnet

import (
    "fmt"
    "github.com/google/gopacket"
    "github.com/google/gopacket/layers"
    "github.com/google/gopacket/pcap"
    "github.com/google/gopacket/tcpassembly"
    "testing"
)

func TestNgnet(t *testing.T) {
    eventChan := make(chan interface{}, 1024)
    f := NewHttpStreamFactory(eventChan)
    pool := tcpassembly.NewStreamPool(f)
    assembler := tcpassembly.NewAssembler(pool)
    packetCount := 0
    fmt.Println("Run")
    if handle, err := pcap.OpenOffline("dump.pcapng"); err != nil {
        panic(err)
    } else {
        packetSource := gopacket.NewPacketSource(handle, handle.LinkType())
        for packet := range packetSource.Packets() {
            net_layer := packet.NetworkLayer()
            trans_layer := packet.TransportLayer()

            if net_layer == nil {
                continue
            }
            if trans_layer == nil {
                continue
            }
            packetCount++
            tcp, _ := trans_layer.(*layers.TCP)
            assembler.AssembleWithTimestamp(net_layer.NetworkFlow(), tcp, packet.Metadata().CaptureInfo.Timestamp)
        }
    }
    assembler.FlushAll()
    f.Wait()
    fmt.Println("packet:", packetCount, "http:", len(eventChan))
}
