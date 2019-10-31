package ucred

import (
	"errors"
	"fmt"
	"net"
	"syscall"

	xcontext "golang.org/x/net/context"

	"google.golang.org/grpc/credentials"
)

type Logger interface {
	WarnS(string)
}

func logf(format string, a ...interface{}) string {
	return fmt.Sprintf("[ucred] "+format, a...)
}

var errUnimplemented = errors.New("unimplemented")

// A `ConnAuthorizer` must be stored on `SoPeerCred` to accept connection
// during `ServerHandshake()`.
type ConnAuthorizer interface {
	AuthorizeInfo(*AuthInfo) error
}

// `SoPeerCred` implements `grpc/credentials.TransportCredentials` for use as a
// `grpc.Creds()` server option.  DO NOT use it as a client option.
type SoPeerCred struct {
	Authorizer ConnAuthorizer
	Logger     Logger
}

func (cr *SoPeerCred) warnS(msg string) {
	if cr.Logger != nil {
		cr.Logger.WarnS(msg)
	}
}

// Dummy that returns an error.
func (creds *SoPeerCred) ClientHandshake(
	xcontext.Context, string, net.Conn,
) (net.Conn, credentials.AuthInfo, error) {
	return nil, nil, errUnimplemented
}

// `ServerHandshake()` uses SO_PEERCRED to get the client ucred.  It then
// checks that the ucred is authorized and stores it as `AuthInfo`, which can
// be retrieved with `FromContext(ctx)` to authorize individual GRPC
// operations.
//
// If ucred is not authorized, `ServerHandshake()` returns an error to GRPC,
// which will close the connection.  The server logs a warning.  The client
// receives `code = Unavailable desc = transport`.
func (creds *SoPeerCred) ServerHandshake(
	conn net.Conn,
) (net.Conn, credentials.AuthInfo, error) {
	uconn, ok := conn.(*net.UnixConn)
	if !ok {
		err := fmt.Errorf("not a Unix connection")
		creds.warnS(logf("handshake denied: %s", err))
		return nil, nil, err
	}

	fp, err := uconn.File()
	if err != nil {
		err = fmt.Errorf("failed to get fd: %s", err)
		creds.warnS(logf("handshake denied: %s", err))
		return nil, nil, err
	}
	defer func() {
		_ = fp.Close()
	}()

	cred, err := syscall.GetsockoptUcred(
		int(fp.Fd()), syscall.SOL_SOCKET, syscall.SO_PEERCRED,
	)
	if err != nil {
		err = fmt.Errorf("failed to SO_PEERCRED: %s", err)
		creds.warnS(logf("handshake denied: %s", err))
		return nil, nil, err
	}

	auth := AuthInfo{Ucred: *cred}
	if creds.Authorizer == nil {
		err := errors.New("missing authorizer")
		creds.warnS(logf("handshake denied: %s", err))
		return nil, nil, err
	}
	if err := creds.Authorizer.AuthorizeInfo(&auth); err != nil {
		creds.warnS(logf("handshake denied: %s", err))
		return nil, nil, err
	}

	return conn, auth, nil
}

// `Info()` returns something moderately useful.  It was not obvious that it
// needs to do more.
func (creds *SoPeerCred) Info() credentials.ProtocolInfo {
	return credentials.ProtocolInfo{
		SecurityProtocol: "peercred",
		ServerName:       "localhost",
	}
}

// Dummy implementation that returns self, which should be ok, since
// `SoPeerCred` is immutable.
func (creds *SoPeerCred) Clone() credentials.TransportCredentials {
	return creds
}

// Dummy implementation.
func (creds *SoPeerCred) OverrideServerName(string) error {
	return errUnimplemented
}
