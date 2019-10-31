// Package `suc/quota` implements GRPC service `sucpb.SuCall` client
// `SetQuota()`.
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

// `SetQuota()` calls the server to set quota.  It uses a reasonable timeout
// internally without exposing a context, because setting quota should be
// relatively fast.
func (c *Client) SetQuota(req *pb.SetQuotaRequest) error {
	ctx, cancel := context.WithDeadline(
		context.Background(), time.Now().Add(10*time.Second),
	)
	defer cancel()

	_, err := c.rpc.SetQuota(ctx, req)
	return err
}
