// vim: sw=8

// `bcpsucd` is the root daemon for privilege separation; see NOE-12.
//
// This is a root server.  Do not add dependencies lightly.  Standard lib and
// widely used packages are ok.
//
// Use package `flag`, not `docopt`, to limit dependencies.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/nogproject/bcpfs/cmd/bcpsucd/v"
	"github.com/nogproject/bcpfs/internal/suc/grpcd"
	"github.com/nogproject/bcpfs/internal/suc/quotad"
	"github.com/nogproject/bcpfs/internal/suc/statusd"
	"github.com/nogproject/bcpfs/pkg/grpc/ucred"
	"github.com/nogproject/bcpfs/pkg/zap"
)

// Double single quote is backtick substitute.
func qqBackticks(s string) string {
	return strings.Replace(s, "''", "`", -1)
}

var usageIntro = qqBackticks(`Usage:
  bcpsucd [options]
  bcpsucd --conn-allow-uids=<uids> --status-allow-uids=<uids>

Options:
`)

func init() {
	flag.Usage = func() {
		fmt.Print(usageIntro)
		flag.PrintDefaults()
		fmt.Print(usageDetails)
	}
}

var usageDetails = qqBackticks(`
''bcpsucd'' changes the socket mode to 0777, so that any uid can connect,
assuming that access is controlled via ''--*-allow-uid'' flags.  To restrict
access to the socket, place it into a directory with the desired access mode.
`)

var (
	optVersion = flag.Bool("version", false, "print the version")
	argSocket  = flag.String(
		"socket", "/var/run/bcpsucd/socket",
		"Unix domain socket, listening mode 0777",
	)
)

type uint32List []uint32

func (l *uint32List) String() string {
	return fmt.Sprint(*l)
}

func (l *uint32List) Set(value string) error {
	for _, v := range strings.Split(value, ",") {
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		if i < 0 {
			return errors.New("negative int")
		}
		*l = append(*l, uint32(i))
	}
	return nil
}

var argConnAllowUids uint32List
var argStatusAllowUids uint32List
var argQuotaAllowUids uint32List

func init() {
	flag.Var(
		&argConnAllowUids, "conn-allow-uids",
		"comma-separated list of uids allowed to connect",
	)
	flag.Var(
		&argStatusAllowUids, "status-allow-uids",
		"comma-separated list of uids allowed to call status service",
	)
	flag.Var(
		&argQuotaAllowUids, "quota-allow-uids",
		"comma-separated list of uids allowed to call quota service",
	)
}

var argQuotaFilesystem = flag.String(
	"quota-filesystem", "/nonexistent",
	"filesystem to which quota operations are restricted",
)

var argQuotaDryRun = flag.Bool(
	"quota-dry-run", false, "skip actual quota operations",
)

type suCallServer struct {
	*statusd.StatusServer
	quotad.QuotaServer
}

func logf(format string, a ...interface{}) string {
	return fmt.Sprintf("[bcpsucd] "+format, a...)
}

func main() {
	flag.Parse()

	version := fmt.Sprintf("bcpsucd-%s+%s", v.Version, v.Build)
	if *optVersion {
		fmt.Println(version)
		return
	}

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatal("failed to create logger", err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.InfoS(logf("Started."))

	connAllow := ucred.NewUidAuthorizer(argConnAllowUids...)
	logger.InfoS(logf("Uids allowed to connect: %v", argConnAllowUids))

	statusAllow := ucred.NewUidAuthorizer(argStatusAllowUids...)
	logger.InfoS(
		logf("Uids allowed to call status: %v", argStatusAllowUids),
	)

	quotaAllow := ucred.NewUidAuthorizer(argQuotaAllowUids...)
	logger.InfoS(
		logf("Uids allowed to call quota: %v", argQuotaAllowUids),
	)

	rpcsrv := &suCallServer{
		StatusServer: statusd.NewServer(statusAllow, logger),
		QuotaServer: quotad.NewServer(
			quotaAllow, logger,
			*argQuotaFilesystem, *argQuotaDryRun,
		),
	}
	rpcsrv.StatusServer.Version = version
	const sockMode = 0777
	sucd, err := grpcd.NewServer(
		*argSocket, sockMode,
		connAllow, logger,
		rpcsrv,
	)
	if err != nil {
		msg := fmt.Sprintf("failed to create server: %s", err)
		log.Fatal(msg)
	}

	sigs := make(chan os.Signal, 1)
	isShutdown := false
	gracePeriod := 20 * time.Second
	done := make(chan bool, 1)
	signal.Notify(sigs, syscall.SIGTERM)

	go func() {
		err = sucd.Serve()
		if isShutdown {
			done <- true
			return
		}
		logger.FatalS(logf("failed to serve: %s", err))
	}()

	<-sigs
	isShutdown = true
	sucd.GracefulStop()
	timeout := time.NewTimer(gracePeriod)
	logger.InfoS(
		logf("SIGTERM, started %s graceful shutdown.", gracePeriod),
	)

	select {
	case <-timeout.C:
		sucd.Stop()
		logger.WarnS(logf("Timeout, forced shutdown."))
	case <-done:
	}

	logger.InfoS(logf("Exited."))
}
