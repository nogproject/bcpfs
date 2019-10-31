// Package `ucred` provides `SO_PEERCRED` auth for GRPC over a Unix domain
// socket.
package ucred

import (
	"context"
	"syscall"

	"google.golang.org/grpc/peer"
)

// `AuthInfo.Ucred` is a field, so that further information, like TLS, could be
// added to `AuthInfo`.
type AuthInfo struct {
	Ucred syscall.Ucred
}

// `AuthType()` makes it an `grpc/credentials.AuthInfo` interface.
func (AuthInfo) AuthType() string {
	return "ucred"
}

// `FromContext()` returns the `ucred.AuthInfo` if it exists in `ctx`.  It uses
// `grpc/peer.FromContext()`.
func FromContext(ctx context.Context) (*AuthInfo, bool) {
	pr, ok := peer.FromContext(ctx)
	if !ok {
		return nil, false
	}
	info, ok := pr.AuthInfo.(AuthInfo)
	if !ok {
		return nil, false
	}
	return &info, true
}
