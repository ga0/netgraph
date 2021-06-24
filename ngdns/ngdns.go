package ngdns

import "time"

// HTTPHeaderItem is HTTP header key-value pair
type HTTPHeaderItem struct {
	Name  string
	Value string
}

// DNSEvent is DNS request or response
type DNSEvent struct {
	Type      string
	Start     time.Time
	End       time.Time
	StreamSeq uint
}

// DNSRequestEvent is HTTP request
type DNSRequestEvent struct {
	DNSEvent
	ClientAddr string
	ServerAddr string
	Method     string
	URI        string
	Version    string
	Headers    []HTTPHeaderItem
	Body       []byte
}

// DNSResponseEvent is HTTP response
type DNSResponseEvent struct {
	DNSEvent
	ClientAddr string
	ServerAddr string
	Version    string
	Code       uint
	Reason     string
	Headers    []HTTPHeaderItem
	Body       []byte
}
