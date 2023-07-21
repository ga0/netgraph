FROM golang:1.18.3
RUN apt update -y &&\
    apt install libpcap-dev -y &&\
    go install github.com/ga0/netgraph@latest