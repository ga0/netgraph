# netgraph
Capture and analyze http and tcp streams

一个B/S架构的HTTP抓包工具。
抓包和组包使用 github.com/google/gopacket
前后端通信使用 golang.org/x/net/websocket

![截图](https://raw.githubusercontent.com/ga0/netgraph/master/screenshot.png)

请确保你的浏览器支持 websocket。

## 编译,安装,运行

      1. go get github.com/ga0/netgraph
      2. 进入 netgraph 项目的根目录(netgraph会从client中取前端页面);
      3. 执行 go build
      4. 执行 ./netgraph -e 网卡名称(比如eth0) -p 服务器端口(默认9000);
      5. 用浏览器打开运行 netgraph 的服务器地址（比如 http://localhost:9000）。

## 选项
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

## 说明
有任何疑问请及时联系我，期待您的反馈。

