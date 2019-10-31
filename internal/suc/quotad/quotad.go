// Package `suc/quotad` implements GRPC service `sucpb.SuCall` server
// `SetQuota()`.
package quotad

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/nogproject/bcpfs/pkg/execx"
	"github.com/nogproject/bcpfs/pkg/grpc/ucred"
	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
	xcontext "golang.org/x/net/context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var ErrUnimplemented = status.Error(codes.Unimplemented, "unimplemented")

type QuotaServer interface {
	SetQuota(
		xcontext.Context, *pb.SetQuotaRequest,
	) (*pb.SetQuotaResponse, error)
}

type quotaServerUnimplemented struct{}

func NewServerUnimplemented() QuotaServer {
	return &quotaServerUnimplemented{}
}

func (*quotaServerUnimplemented) SetQuota(
	xcontext.Context, *pb.SetQuotaRequest,
) (*pb.SetQuotaResponse, error) {
	return nil, ErrUnimplemented
}

type Authorizer interface {
	Authorize(context.Context) error
}

type Logger interface {
	WarnS(string)
	InfoS(string)
}

func logf(format string, a ...interface{}) string {
	return fmt.Sprintf("[suc/quotad] "+format, a...)
}

type quotaServer struct {
	authorizer Authorizer
	logger     Logger
	filesystem string
	dryRun     bool
}

// `NewServer(authz, l, filesystem)` returns a quota server that uses
// `authz.Authorize()` for per-request authorization.  Quota operations are
// only allowed for `filesystem`.
func NewServer(
	authz Authorizer, l Logger, filesystem string, dryRun bool,
) QuotaServer {
	return &quotaServer{
		authorizer: authz,
		logger:     l,
		filesystem: filesystem,
		dryRun:     dryRun,
	}
}

var cat = execx.MustLookTool(execx.ToolSpec{
	Program:   "cat",
	CheckArgs: []string{"--version"},
	CheckText: "cat",
})

var setquota = execx.MustLookTool(execx.ToolSpec{
	Program:   "setquota",
	CheckArgs: []string{"--version"},
	CheckText: "Quota utilities version 4",
})

func (s *quotaServer) SetQuota(
	ctx xcontext.Context, req *pb.SetQuotaRequest,
) (*pb.SetQuotaResponse, error) {
	if err := s.authorizer.Authorize(ctx); err != nil {
		s.logger.WarnS(logf("SetQuota(): %s", err))
		return nil, err
	}

	info, ok := ucred.FromContext(ctx)
	if !ok {
		panic("authorized context without auth info")
	}
	uid := info.Ucred.Uid

	if req.Filesystem != s.filesystem {
		err := status.Error(codes.InvalidArgument, "wrong filesystem")
		s.logger.WarnS(logf("uid:%d SetQuota(): %s", uid, err))
		return nil, err
	}

	s.logger.InfoS(logf(
		"uid:%d rpc SetQuota(scope:%s, len(limits):%d)",
		uid, req.Scope, len(req.Limits),
	))

	var cmd *exec.Cmd
	switch {
	case s.dryRun:
		cmd = exec.Command(cat.Path)
	case req.Scope == pb.QuotaScope_USER_QUOTA:
		cmd = exec.Command(
			setquota.Path, "--user", "--batch", s.filesystem,
		)
	case req.Scope == pb.QuotaScope_GROUP_QUOTA:
		cmd = exec.Command(
			setquota.Path, "--group", "--batch", s.filesystem,
		)
	default:
		return nil, status.Error(
			codes.InvalidArgument, "invalid quota scope",
		)
	}

	var buf bytes.Buffer
	for _, l := range req.Limits {
		buf.Write([]byte(fmt.Sprintf(
			"%s %d %d %d %d\n",
			l.Xid,
			l.BlockSoftLimit, l.BlockHardLimit,
			l.InodeSoftLimit, l.InodeHardLimit,
		)))
	}

	cmd.Stdin = &buf
	out, err := cmd.CombinedOutput()
	if err != nil {
		err := status.Error(
			codes.Unknown,
			fmt.Sprintf("%s, output: %s", err, string(out)),
		)
		s.logger.WarnS(logf("uid:%d SetQuota(): %s", uid, err))
		return nil, err
	}

	if s.dryRun {
		s.logger.InfoS(logf(
			"uid:%d setquota --dry-run: %s", uid, string(out),
		))
	}

	return &pb.SetQuotaResponse{}, nil
}
