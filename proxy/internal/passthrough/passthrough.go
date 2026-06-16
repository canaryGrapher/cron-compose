// Package passthrough copies a raw TCP connection to an upstream without
// inspecting or terminating it, preserving agent mTLS end-to-end.
package passthrough

import (
	"io"
	"net"
	"time"
)

// Handle proxies an already-accepted client connection to a TCP upstream,
// copying bytes both ways until either side closes. The client connection may
// carry replayed handshake bytes (see mux.peek); those are forwarded first, so
// the upstream sees a complete, untouched TLS stream.
func Handle(client net.Conn, upstreamAddr string, dialTimeout time.Duration) {
	up, err := net.DialTimeout("tcp", upstreamAddr, dialTimeout)
	if err != nil {
		client.Close()
		return
	}

	done := make(chan struct{}, 2)
	go func() { _, _ = io.Copy(up, client); done <- struct{}{} }()
	go func() { _, _ = io.Copy(client, up); done <- struct{}{} }()

	// As soon as one direction ends, close both so the other unblocks.
	<-done
	client.Close()
	up.Close()
	<-done
}
