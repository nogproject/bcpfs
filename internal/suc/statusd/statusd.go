// Package `statusd` implements GRPC service `sucpb.SuCall` server `Status()`.
package statusd

import (
	"context"
	"fmt"

	"github.com/nogproject/bcpfs/pkg/grpc/ucred"
	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
	xcontext "golang.org/x/net/context"
)

type Authorizer interface {
	Authorize(context.Context) error
}

type Logger interface {
	WarnS(string)
	InfoS(string)
}

func logf(format string, a ...interface{}) string {
	return fmt.Sprintf("[suc/statusd] "+format, a...)
}

type StatusServer struct {
	authorizer Authorizer
	logger     Logger
	Version    string
}

// `NewServer(authz)` returns a status server that uses `authz.Authorize()` for
// per-request authorization.
func NewServer(authz Authorizer, l Logger) *StatusServer {
	return &StatusServer{authorizer: authz, logger: l}
}

// `Status()` returns a placeholder status text.
func (s *StatusServer) Status(
	ctx xcontext.Context, request *pb.StatusRequest,
) (*pb.StatusResponse, error) {
	if err := s.authorizer.Authorize(ctx); err != nil {
		s.logger.WarnS(logf("Status(): %s", err))
		return nil, err
	}

	info, ok := ucred.FromContext(ctx)
	if !ok {
		panic("authorized context without auth info")
	}
	s.logger.InfoS(logf("uid:%d rpc Status()", info.Ucred.Uid))

	txt := fmt.Sprintf(`version: %s
clientUid: %d
clientGid: %d
`,
		s.version(),
		info.Ucred.Uid, info.Ucred.Gid,
	)

	rsp := &pb.StatusResponse{Text: txt}
	return rsp, nil
}

func (s *StatusServer) version() string {
	if s.Version == "" {
		return "unspecified"
	}
	return s.Version
}
