package ngdns

import "time"

// DNSEvent is DNS request or response
type DNSEvent struct {
	Type         string
	Start        time.Time
	End          time.Time
	ID           uint16
	QR           bool
	OpCode       int
	ResponseCose string
	Questions    []DNSQuestion
	Answers      []DNSAnswer
	StreamSeq    string
}

type DNSQuestion struct {
	Name  string
	Type  string
	Class string
}

type DNSAnswer struct {
	Name       string
	Type       string
	Class      string
	DataLength uint16
}
