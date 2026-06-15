// Package transport wraps the gRPC client to the control plane (mTLS).
package transport

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	agentv1 "github.com/croncompose/croncompose/proto/agent/v1"
)

// Client is a thin wrapper around the generated AgentService client.
type Client struct {
	addr string
	conn *grpc.ClientConn
	svc  agentv1.AgentServiceClient
}

// Dial opens a mTLS connection to the control plane.
func Dial(ctx context.Context, addr string, tlsCfg *tls.Config) (*Client, error) {
	dialCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(dialCtx, addr,
		grpc.WithTransportCredentials(credentials.NewTLS(tlsCfg)),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", addr, err)
	}
	return &Client{
		addr: addr,
		conn: conn,
		svc:  agentv1.NewAgentServiceClient(conn),
	}, nil
}

// Close tears down the connection.
func (c *Client) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

// OpenStream attaches to AgentStream. Authentication is the mTLS client cert.
func (c *Client) OpenStream(ctx context.Context) (agentv1.AgentService_AgentStreamClient, error) {
	return c.svc.AgentStream(ctx)
}
