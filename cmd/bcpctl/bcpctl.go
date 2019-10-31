// vim: sw=8

// The `bcpctl` command controls BCP configuration.
package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	docopt "github.com/docopt/docopt-go"

	"github.com/nogproject/bcpfs/cmd/bcpctl/internal/quota"
	"github.com/nogproject/bcpfs/cmd/bcpctl/v"
	"github.com/nogproject/bcpfs/pkg/suc/grpc"
	quotac "github.com/nogproject/bcpfs/pkg/suc/quota"
	"github.com/nogproject/bcpfs/pkg/suc/status"
	pb "github.com/nogproject/bcpfs/pkg/suc/sucpb"
)

var version = fmt.Sprintf("bcpctl-%s+%s", v.Version, v.Build)

// Double single quote is backtick substitute.
func qqBackticks(s string) string {
	return strings.Replace(s, "''", "`", -1)
}

var usage = qqBackticks(`Usage:
  bcpctl [--socket=<socket>] status
  bcpctl [--socket=<socket>] setquota --user --batch <filesystem>
  bcpctl [--socket=<socket>] setquota --group --batch <filesystem>
  bcpctl version

Options:
  --socket=<socket>  [default: /var/run/bcpsucd/socket]
	Path to the bcpsucd socket.

  --user    Set user quota.
  --group   Set group quota.

''bcpctl'' controls BCP configuration.  It contacts bcpsucd for operations that
require admin privileges.

''bcpctl status'' reports the bcpsucd status.

''bcpctl setquota'' reads quota settings from stdin and applies them to
''<filesystem>''.  See man ''setquota(8)'' for the input format.
`)

func main() {
	args := argparse()
	switch {
	case args["version"].(bool):
		cmdVersion()
	case args["status"].(bool):
		cmdStatus(args)
	case args["setquota"].(bool):
		cmdSetquota(args)
	default:
		panic("must not be reached")
	}
}

func argparse() map[string]interface{} {
	const autoHelp = true
	const noOptionFirst = false
	args, err := docopt.Parse(
		usage, nil, autoHelp, version, noOptionFirst,
	)
	if err != nil {
		panic(fmt.Sprintf("docopt failed: %s", err))
	}
	return args
}

func cmdVersion() {
	fmt.Println(version)
}

func cmdStatus(args map[string]interface{}) {
	socket := args["--socket"].(string)

	conn, err := grpc.Dial(socket)
	if err != nil {
		msg := fmt.Sprintf("failed to dial: %s", err)
		log.Fatal(msg)
	}
	defer func() {
		// Ignore error.  Process is about to exit anyway.
		_ = conn.Close()
	}()

	suc := status.NewClient(conn)
	txt, err := suc.Status()
	if err != nil {
		msg := fmt.Sprintf("status failed: %s", err)
		log.Fatal(msg)
	}
	fmt.Print(txt)
}

func cmdSetquota(args map[string]interface{}) {
	socket := args["--socket"].(string)

	quotas, err := quota.Parse(os.Stdin)
	if err != nil {
		msg := fmt.Sprintf("failed to parse stdin: %s", err)
		log.Fatal(msg)
	}

	conn, err := grpc.Dial(socket)
	if err != nil {
		msg := fmt.Sprintf("failed to dial: %s", err)
		log.Fatal(msg)
	}
	defer func() {
		// Ignore error.  Process is about to exit anyway.
		_ = conn.Close()
	}()

	req := &pb.SetQuotaRequest{
		Filesystem: args["<filesystem>"].(string),
	}
	if args["--user"].(bool) {
		req.Scope = pb.QuotaScope_USER_QUOTA
	}
	if args["--group"].(bool) {
		req.Scope = pb.QuotaScope_GROUP_QUOTA
	}
	for _, q := range quotas {
		pq := pb.QuotaLimit{
			Xid:            q.Xid,
			BlockSoftLimit: q.BlockSoftLimit,
			BlockHardLimit: q.BlockHardLimit,
			InodeSoftLimit: q.InodeSoftLimit,
			InodeHardLimit: q.InodeHardLimit,
		}
		req.Limits = append(req.Limits, &pq)
	}

	suc := quotac.NewClient(conn)
	err = suc.SetQuota(req)
	if err != nil {
		msg := fmt.Sprintf("server SetQuota(): %s", err)
		log.Fatal(msg)
	}
}
