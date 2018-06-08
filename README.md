![](https://travis-ci.com/ga0/netgraph.svg?branch=master)

# netgraph
Capture and analyze http and tcp streams

一个B/S架构的HTTP抓包工具。
抓包和组包使用 github.com/google/gopacket
前后端通信使用 golang.org/x/net/websocket

![截图](https://raw.githubusercontent.com/ga0/netgraph/master/screenshot.png)

请确保你的浏览器支持 websocket。

## 编译,安装,运行 / Compile, Install, Run

      1. go get github.com/ga0/netgraph
      2. run $GOPATH/bin/netgraph -i INTERFACE -p PORT
      3. open the netgraph web page in your browser (for example: http://localhost:9000, 9000 is the PORT set in step 2)

windows下需要先安装 winpcap 库。

如果你修改过client下的前端文件：

      1. 在源码根目录下执行 go generate
      2. go build
      3. 运行 netgraph

## 选项 / Options
      -assembly_debug_log
            If true, the github.com/google/gopacket/tcpassembly library will log verbose debugging information (at least one line per packet)
      -assembly_memuse_log
            If true, the github.com/google/gopacket/tcpassembly library will log information regarding its memory use every once in a while.
      -bpf string
            Set berkeley packet filter (default "tcp port 80")
      -i string
            Device to capture, auto select one if no device provided
      -input-pcap string
            Open pcap file
      -o string
            Write HTTP request/response to file
      -output-pcap string
            Write captured packet to a pcap file
      -p int
            Web server port. If the port is set to '0', the server will not run. (default 9000)
      -s	Save HTTP event in server
      -v	Show more message (default true)

## 说明 / License

This project is licensed under the terms of the MIT license.

有任何疑问请及时联系我，期待您的反馈。

