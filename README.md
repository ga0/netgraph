# netgraph
Capture and analyze http and tcp streams

一个B/S架构的HTTP抓包工具。
抓包和组包使用 github.com/google/gopacket
前后端通信使用 golang.org/x/net/websocket

![截图](https://raw.githubusercontent.com/ga0/netgraph/master/screenshot.png)

请确保你的浏览器支持 websocket。

## 编译,安装,运行

      1. go get github.com/ga0/netgraph
      2. 执行 $GOPATH/bin/netgraph -i 网卡名称(比如eth0) -p 服务器端口(默认9000)
      3. 用浏览器打开运行 netgraph 的服务器地址(比如 http://localhost:9000)

windows下需要先安装winpcap库。

如果你修改过client下的前端文件：

      1. 在源码根目录下执行 go generate
      2. go build
      3. 运行 netgraph

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

This project is licensed under the terms of the MIT license.

有任何疑问请及时联系我，期待您的反馈。

