// Package `status` implements GRPC service `sucpb.SuCall` client `Status()`.
package status

import (
	"context"
	"time"

	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
	"google.golang.org/grpc"
)

type Client struct {
	rpc pb.SuCallClient
}

func NewClient(conn *grpc.ClientConn) *Client {
	return &Client{rpc: pb.NewSuCallClient(conn)}
}

// `Status()` returns the server status.  It uses a timeout internally without
// exposing a context, because status should always be quick.
func (c *Client) Status() (string, error) {
	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(100*time.Millisecond),
	)
	defer cancel()

	rsp, err := c.rpc.Status(ctx, &pb.StatusRequest{})
	if err != nil {
		return "", err
	}
	return rsp.Text, nil
}
