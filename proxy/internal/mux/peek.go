package mux

import (
	"bytes"
	"io"
	"net"
	"time"
)

// peeked wraps a net.Conn whose first bytes were read for protocol detection;
// it replays those bytes before resuming the live stream.
type peeked struct {
	net.Conn
	r io.Reader
}

func (p *peeked) Read(b []byte) (int, error) { return p.r.Read(b) }

// clientHello is the result of sniffing a new connection.
type clientHello struct {
	isTLS bool
	sni   string
}

// peek reads just enough of conn to classify it: whether it opens a TLS
// handshake and, if so, the SNI server name. It returns a connection that
// replays everything consumed, so the caller can hand it to a TLS terminator,
// an HTTP server, or a raw passthrough unchanged.
func peek(conn net.Conn, timeout time.Duration) (clientHello, net.Conn) {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	defer func() { _ = conn.SetReadDeadline(time.Time{}) }()

	var buf bytes.Buffer
	tee := io.TeeReader(conn, &buf)
	replay := func() net.Conn {
		return &peeked{Conn: conn, r: io.MultiReader(bytes.NewReader(buf.Bytes()), conn)}
	}

	hdr := make([]byte, 1)
	if _, err := io.ReadFull(tee, hdr); err != nil {
		return clientHello{}, replay()
	}
	// 0x16 marks a TLS handshake record; anything else is treated as cleartext.
	if hdr[0] != 0x16 {
		return clientHello{isTLS: false}, replay()
	}

	// Remainder of the 5-byte TLS record header: version(2) + length(2).
	rest := make([]byte, 4)
	if _, err := io.ReadFull(tee, rest); err != nil {
		return clientHello{isTLS: true}, replay()
	}
	recLen := int(rest[2])<<8 | int(rest[3])
	if recLen <= 0 || recLen > 1<<14 {
		return clientHello{isTLS: true}, replay()
	}
	body := make([]byte, recLen)
	if _, err := io.ReadFull(tee, body); err != nil {
		return clientHello{isTLS: true}, replay()
	}
	return clientHello{isTLS: true, sni: parseSNI(body)}, replay()
}

// parseSNI extracts the host_name from a TLS ClientHello handshake body, or ""
// if absent or malformed. Every length field is bounds-checked before use.
func parseSNI(b []byte) string {
	// Handshake header: msg_type(1)=0x01 client_hello, length(3).
	if len(b) < 4 || b[0] != 0x01 {
		return ""
	}
	b = b[4:]
	// client_version(2) + random(32).
	if len(b) < 34 {
		return ""
	}
	b = b[34:]
	// session_id.
	if len(b) < 1 || len(b) < 1+int(b[0]) {
		return ""
	}
	b = b[1+int(b[0]):]
	// cipher_suites.
	if len(b) < 2 {
		return ""
	}
	csLen := int(b[0])<<8 | int(b[1])
	if len(b) < 2+csLen {
		return ""
	}
	b = b[2+csLen:]
	// compression_methods.
	if len(b) < 1 || len(b) < 1+int(b[0]) {
		return ""
	}
	b = b[1+int(b[0]):]
	// extensions.
	if len(b) < 2 {
		return ""
	}
	extLen := int(b[0])<<8 | int(b[1])
	b = b[2:]
	if len(b) > extLen {
		b = b[:extLen]
	}
	for len(b) >= 4 {
		etype := int(b[0])<<8 | int(b[1])
		elen := int(b[2])<<8 | int(b[3])
		b = b[4:]
		if len(b) < elen {
			return ""
		}
		ext := b[:elen]
		b = b[elen:]
		if etype != 0x0000 { // server_name extension
			continue
		}
		// server_name_list: list_len(2), entries of type(1)+len(2)+name.
		if len(ext) < 2 {
			return ""
		}
		listLen := int(ext[0])<<8 | int(ext[1])
		ext = ext[2:]
		if len(ext) < listLen {
			return ""
		}
		ext = ext[:listLen]
		for len(ext) >= 3 {
			nameType := ext[0]
			nameLen := int(ext[1])<<8 | int(ext[2])
			ext = ext[3:]
			if len(ext) < nameLen {
				return ""
			}
			if nameType == 0x00 { // host_name
				return string(ext[:nameLen])
			}
			ext = ext[nameLen:]
		}
	}
	return ""
}
