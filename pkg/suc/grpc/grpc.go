// Package `grpc` provides GRPC details for sucd clients.
package grpc

import (
	"context"
	"fmt"
	"net"
	"time"

	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
	"google.golang.org/grpc"
)

func unixDialer(addr string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", addr, timeout)
}

// `Dial(sock)` connects to a Unix domain socket.
func Dial(sock string) (*grpc.ClientConn, error) {
	conn, err := grpc.Dial(
		sock, grpc.WithInsecure(), grpc.WithDialer(unixDialer),
	)
	if err != nil {
		return nil, err
	}

	// Ping server to discover connection problems early.
	rpc := pb.NewPingClient(conn)
	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(50*time.Millisecond),
	)
	defer cancel()
	if _, err := rpc.Ping(ctx, &pb.PingRequest{}); err != nil {
		_ = conn.Close() // Try to cleanup, but ignore errors.
		return nil, fmt.Errorf("ping failed: %s", err)
	}

	return conn, nil
}
