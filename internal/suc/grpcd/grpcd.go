// Package `grpcd` provides sucd-specific GRPC details.
package grpcd

import (
	"net"
	"os"

	"github.com/nogproject/bcpfs/pkg/grpc/ucred"
	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
	"google.golang.org/grpc"
)

// `Server` is a GRPC server listening on a Unix domain socket.
type Server struct {
	listener net.Listener
	gServer  *grpc.Server
}

// `NewServer()` returns a server that listens on the Unix domain `socket`,
// which is removed before listen, and set to file `mode` right after listen.
// There is a short window between listen and chmod where the socket mode may
// be different.  To ensure full protection at all times, place the socket in a
// directory with the desired mode.
//
// Use `Serve()` to start serving requests.
//
// `authz.AuthorizeInfo()` is called to authorize a connection ucred before the
// connection is used as GRPC transport.  Connections that `authz` rejects will
// be closed by GRCP.
//
// `srv` must implement the server side of GRPC service `sucpb.SuCall`.
func NewServer(
	socket string,
	mode os.FileMode,
	authz ucred.ConnAuthorizer,
	logger ucred.Logger,
	srv pb.SuCallServer,
) (*Server, error) {
	_ = os.Remove(socket) // Avoid `bind: address already in use`.
	lis, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(socket, mode); err != nil {
		return nil, err
	}

	creds := &ucred.SoPeerCred{
		Authorizer: authz,
		Logger:     logger,
	}
	gs := grpc.NewServer(grpc.Creds(creds))
	pb.RegisterPingServer(gs, &pingServer{})
	pb.RegisterSuCallServer(gs, srv)

	return &Server{
		listener: lis,
		gServer:  gs,
	}, nil
}

// `Serve()` handles requests forever unless there is an error.
func (s *Server) Serve() error {
	return s.gServer.Serve(s.listener)
}

func (s *Server) GracefulStop() {
	s.gServer.GracefulStop()
}

func (s *Server) Stop() {
	s.gServer.GracefulStop()
}
