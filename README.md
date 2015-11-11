# netgraph
Capture and analyze http and tcp streams

一个B/S架构的HTTP抓包工具。
依赖 github.com/google/gopacket.
![截图](https://raw.githubusercontent.com/ga0/netgraph/master/screenshot.png)

# 选项
    -bpf string
          Berkeley Packet Filter (default "tcp port 80")
    -f string
          Open pcap file
    -i string
          Device to capture, auto select one if no device provided
    -o string
          Output captured packet to pcap file
    -p int
          Web server port (default 9000)
    -s    save network event in server