package grpcd

import (
	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
	xcontext "golang.org/x/net/context"
)

type pingServer struct{}

func (s *pingServer) Ping(
	ctx xcontext.Context, request *pb.PingRequest,
) (*pb.PingResponse, error) {
	// No authz: Valid transport implies permission to ping.
	return &pb.PingResponse{}, nil
}
