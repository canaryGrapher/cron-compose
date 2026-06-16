// Package mux is the single-port demultiplexer. Every inbound connection is
// classified and dispatched: browser HTTP(S) to the reverse-proxy handler, agent
// TLS (mTLS gRPC) straight through to the control plane.
package mux

import (
	"context"
	"crypto/tls"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/croncompose/croncompose/proxy/internal/config"
	"github.com/croncompose/croncompose/proxy/internal/passthrough"
)

// Server listens on one address and routes by protocol.
type Server struct {
	cfg     config.Config
	log     *slog.Logger
	handler http.Handler
	tlsCfg  *tls.Config // nil when the proxy does not terminate browser TLS
}

// New builds the mux. When TLS is configured it loads the server certificate
// used to terminate browser HTTPS.
func New(cfg config.Config, handler http.Handler, log *slog.Logger) (*Server, error) {
	s := &Server{cfg: cfg, log: log, handler: handler}
	if cfg.TLSEnabled() {
		cert, err := tls.LoadX509KeyPair(cfg.TLSCertFile, cfg.TLSKeyFile)
		if err != nil {
			return nil, err
		}
		s.tlsCfg = &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}
	}
	return s, nil
}

// Serve listens on cfg.ListenAddr and blocks until ctx is canceled.
func (s *Server) Serve(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.cfg.ListenAddr)
	if err != nil {
		return err
	}
	defer ln.Close()

	// HTTP connections are funneled into an in-memory listener consumed by a
	// single http.Server, which keeps normal keep-alive handling intact.
	hl := newChanListener(ln.Addr())
	httpSrv := &http.Server{
		Handler:           s.handler,
		ReadHeaderTimeout: s.cfg.ReadHeaderTimeout,
		IdleTimeout:       s.cfg.IdleTimeout,
	}
	go func() { _ = httpSrv.Serve(hl) }()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
		_ = hl.Close()
		shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpSrv.Shutdown(shutCtx)
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			s.log.Warn("accept failed", "err", err)
			time.Sleep(10 * time.Millisecond)
			continue
		}
		go s.dispatch(conn, hl)
	}
}

// dispatch classifies a single connection and routes it.
func (s *Server) dispatch(conn net.Conn, hl *chanListener) {
	info, replay := peek(conn, 5*time.Second)

	// Cleartext HTTP (browser over http://) -> reverse proxy.
	if !info.isTLS {
		hl.push(replay)
		return
	}

	// TLS. If we terminate browser TLS and this is not the agent SNI, decrypt
	// and serve HTTP. Otherwise pass the raw TLS stream to the gRPC upstream so
	// agent mTLS stays end-to-end.
	if s.tlsCfg != nil && info.sni != s.cfg.AgentSNI {
		hl.push(tls.Server(replay, s.tlsCfg))
		return
	}
	passthrough.Handle(replay, s.cfg.GRPCUpstream, s.cfg.DialTimeout)
}

// chanListener adapts pushed connections into a net.Listener for http.Server.
type chanListener struct {
	conns  chan net.Conn
	addr   net.Addr
	closed chan struct{}
	once   sync.Once
}

func newChanListener(addr net.Addr) *chanListener {
	return &chanListener{conns: make(chan net.Conn), addr: addr, closed: make(chan struct{})}
}

func (l *chanListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *chanListener) Close() error {
	l.once.Do(func() { close(l.closed) })
	return nil
}

func (l *chanListener) Addr() net.Addr { return l.addr }

func (l *chanListener) push(c net.Conn) {
	select {
	case l.conns <- c:
	case <-l.closed:
		_ = c.Close()
	}
}
