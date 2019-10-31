// vim: sw=8

// Command `bcpfs-perms` maintains a toplevel filesystem tree for an
// organization with service facilities and research units.  It creates
// directories and adjusts permissions and POSIX ACLs.
package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/docopt/docopt-go"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpsharing"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpsharingapply"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/describe"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/fsapply"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/fsck"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/v"
)

// The linker injects the version via the subpackage `v`, because direct
// injection into the main package did not work as expected.
var version = fmt.Sprintf("bcpfs-perms-%s+%s", v.Version, v.Build)

// `qqBackticks()` translates double single quote to backtick.
func qqBackticks(s string) string {
	return strings.Replace(s, "''", "`", -1)
}

var usage = qqBackticks(`Usage:
  bcpfs-perms [--config=<path>] describe config
  bcpfs-perms [--config=<path>] describe groups [--strict]
  bcpfs-perms [--config=<path>] describe org [--strict]
  bcpfs-perms [--config=<path>] apply [--debug] [--recursive] [--sharing]
  bcpfs-perms [--config=<path>] check [--debug]
  bcpfs-perms version

Options:
  --config=<path>  [default: /etc/bcpfs.hcl]
        Path to a mandatory HCL config file, which must at least specify the
        ''rootdir''.  See ''/usr/share/doc/bcpfs'' for an example config.
  --debug   Enable debug logging.
  --strict  Enable stricter checking for compatibility of configuration and
        Unix groups.
  --recursive  Apply permissions recursively.
  --sharing    Apply sharing permissions.

''bcpfs-perms'' manages the toplevel directories as described in the 2016
filesystem concept.

''bcpfs-perms describe config'' prints the active config, which is based on the
config file and defaults.

''bcpfs-perms describe groups'' prints the Unix groups, filtered for the active
config.

''bcpfs-perms describe org'' prints the org units, facilities, and services, as
determined from the config and the Unix groups.  ''bcpfs-perms describe org''
runs a basic sanity check to confirm that the configuration is compatible with
the the Unix groups.  If run with ''--strict'', ''bcpfs-perms describe org''
checks if there are service Unix groups that are not specified in the
configuration and exits with an error if there are any.

''bcpfs-perms apply'' creates the toplevel directories and applies permissions.
If used with ''--recursive'', permissions will be propagated to
sub-directories.  Sub-directories are updated silently.

''bcpfs-perms apply --sharing'' manages ''<ou>/shared'' trees, in addition to
the usual permissions, as configured in the ''sharing'' configuration block.
See NOE-9 for a general description.  Example configuration:

''''''
sharing {
    namingPolicy { action = "allow", match = "em-facility/tem-505(/.*)?" }
    namingPolicy { action = "allow", match = "ag-bob/tem-505(/.*)?" }

    export {
        path = "em-facility/tem-505/ag-bob/foo"
        acl = [
            "group:ag-alice:r-x",
        ]
    }

    export {
        path = "ag-bob/tem-505/foo"
        acl = [
            "group:ag-alice:r-x",
        ]
    }

    import { action = "accept", group = "ag-alice", match = "em-facility/.*" }
    import { action = "accept", group = "ag-alice", match = "ag-bob/.*" }
}
''''''

Supported naming policy actions: ''allow'' and ''deny''.
Supported import actions: ''accept'' and ''reject''.

''bcpfs-perms check'' verifies directories and permissions.  ''--debug''
enables reporting of details, such as paths that are skipped due to ''filter''
statements in the configuration.
`)

func main() {
	args := argparse()
	if args["--debug"].(bool) {
		InitDebugLogger()
	}
	switch {
	case args["version"].(bool):
		cmdVersion()
	case args["apply"].(bool):
		cmdApply(args)
	case args["check"].(bool):
		cmdCheck(args)
	case args["describe"].(bool) && args["config"].(bool):
		cmdDescribeConfig(args)
	case args["describe"].(bool) && args["groups"].(bool):
		cmdDescribeGroups(args)
	case args["describe"].(bool) && args["org"].(bool):
		cmdDescribeOrg(args)
	}
}

func argparse() map[string]interface{} {
	const autoHelp = true
	const noOptionFirst = false
	args, _ := docopt.Parse(
		usage, nil, autoHelp, version, noOptionFirst,
	)
	return args
}

func cmdVersion() {
	fmt.Println(version)
}

func cmdApply(args map[string]interface{}) {
	opts := &fsapply.Options{
		Recursive: args["--recursive"].(bool),
	}

	cfg := MustLoadConfig(args["--config"].(string))
	_, org, unconfServices := MustLoadGroups(cfg)
	if len(unconfServices) > 0 {
		for _, s := range unconfServices {
			logger.Error(s)
		}
		msg := "fsapply: There are unconfigured services."
		logger.Fatal(msg)
	}

	filter := MustCompileFilter(cfg)

	err := fsapply.EnsurePermissions(cfg, org, filter, opts)
	if err != nil {
		msg := fmt.Sprintf("Failed to apply permissions: %v", err)
		logger.Fatal(msg)
	}

	if args["--sharing"].(bool) {
		if cfg.Sharing == nil {
			msg := "Missing sharing config."
			logger.Fatal(msg)
		}

		sharing, err := bcpsharing.Compile(cfg)
		if err != nil {
			msg := fmt.Sprintf(
				"Failed to compile sharing: %v", err,
			)
			logger.Fatal(msg)
		}

		if err := bcpsharingapply.EnsureRealShares(
			logger, sharing.Bcpfs, sharing.RealShares,
		); err != nil {
			msg := fmt.Sprintf(
				"Failed to apply sharing: %v", err,
			)
			logger.Fatal(msg)
		}

		if err := bcpsharingapply.EnsureTraversal(
			logger, sharing.Bcpfs, sharing.Traversal,
		); err != nil {
			msg := fmt.Sprintf(
				"Failed to apply sharing traversal: %v", err,
			)
			logger.Fatal(msg)
		}

		if err := bcpsharingapply.EnsureShareTrees(
			logger, sharing.Bcpfs, sharing.ShareTrees,
		); err != nil {
			msg := fmt.Sprintf(
				"Failed to apply sharing trees: %v", err,
			)
			logger.Fatal(msg)
		}
	}
}

func cmdCheck(args map[string]interface{}) {
	cfg := MustLoadConfig(args["--config"].(string))
	_, org, unconfServices := MustLoadGroups(cfg)
	if len(unconfServices) > 0 {
		for _, s := range unconfServices {
			logger.Error(s)
		}
		msg := "fsck: There are unconfigured services."
		logger.Fatal(msg)
	}
	filter := MustCompileFilter(cfg)
	reason, err := fsck.CheckPermissions(cfg, org, filter)
	if err != nil {
		msg := fmt.Sprintf("fsck error: %v", err)
		logger.Fatal(msg)
	}
	if reason != "" {
		msg := fmt.Sprintf("fsck failed: %s", reason)
		logger.Fatal(msg)
	}
	logger.Info("fsck ok")
}

// `MustLoadConfig()` loads the config and inserts defaults.
func MustLoadConfig(path string) *bcpcfg.Root {
	cfg, err := bcpcfg.Load(path)
	if err != nil {
		msg := fmt.Sprintf("Failed to load config: %v", err)
		logger.Fatal(msg)
	}
	if cfg.OrgUnitPrefix == "" {
		logger.Fatal("Missing config `orgUnitPrefix`.")
	}
	if cfg.ServicePrefix == "" {
		logger.Fatal("Missing config `servicePrefix`.")
	}
	if cfg.OpsSuffix == "" {
		cfg.OpsSuffix = "ops"
	}
	if cfg.FacilitySuffix == "" {
		cfg.FacilitySuffix = "facility"
	}
	if cfg.ServiceDir == "" {
		logger.Fatal("Missing config `serviceDir`.")
	}
	if cfg.OrgUnitDir == "" {
		logger.Fatal("Missing config `orgUnitDir`.")
	}
	return cfg
}

// `MustLoadGroups()` loads the Unix groups and parses them to return an
// `Organization`.
func MustLoadGroups(cfg *bcpcfg.Root) (
	[]grp.Group, *bcp.Organization, []string,
) {
	gs, err := grp.Groups()
	if err != nil {
		msg := fmt.Sprintf("Failed to get groups: %v", err)
		logger.Fatal(msg)
	}

	prefixes := []string{
		fmt.Sprintf("%s_", cfg.OrgUnitPrefix),
		fmt.Sprintf("%s_", cfg.ServicePrefix),
	}
	equals := []string{
		cfg.SuperGroup,
	}
	gs = grp.SelectGroups(gs, prefixes, equals)
	gs, err = grp.DedupGroups(gs)
	if err != nil {
		msg := fmt.Sprintf("Failed to select groups: %v", err)
		logger.Fatal(msg)
	}

	sort.SliceStable(gs, func(i, j int) bool {
		return gs[i].Name < gs[j].Name
	})

	org, unconfServices, err := bcp.New(gs, cfg)
	if err != nil {
		msg := fmt.Sprintf("Failed to parse groups: %v", err)
		logger.Fatal(msg)
	}

	return gs, org, unconfServices
}

func MustCompileFilter(cfg *bcpcfg.Root) bfilter.OrgServiceFilter {
	var deciders []bfilter.Decider
	for _, decide := range cfg.Filter {
		if r, err := bfilter.NewRegexpDecider(decide); err != nil {
			msg := fmt.Sprintf(
				"Invalid reject=%+v: %v", decide, err,
			)
			logger.Fatal(msg)
		} else {
			deciders = append(deciders, r)
		}
	}
	deciders = append(deciders, bfilter.NewSameFacilityDecider())
	return &bfilter.DecidersFilter{Rules: deciders}
}

func cmdDescribeConfig(args map[string]interface{}) {
	cfg := MustLoadConfig(args["--config"].(string))
	fmt.Printf("%s", describe.MustDescribeConfig(cfg))
}

func cmdDescribeGroups(args map[string]interface{}) {
	cfg := MustLoadConfig(args["--config"].(string))
	gs, _, unconfServices := MustLoadGroups(cfg)
	if args["--strict"].(bool) && len(unconfServices) > 0 {
		for _, s := range unconfServices {
			logger.Error(s)
		}
		msg := "There are unconfigured services."
		logger.Fatal(msg)
	}
	fmt.Printf("%s", describe.MustDescribeGroups(gs))
}

func cmdDescribeOrg(args map[string]interface{}) {
	cfg := MustLoadConfig(args["--config"].(string))
	_, org, unconfServices := MustLoadGroups(cfg)
	if args["--strict"].(bool) && len(unconfServices) > 0 {
		for _, s := range unconfServices {
			logger.Error(s)
		}
		msg := "There are unconfigured services."
		logger.Fatal(msg)
	}
	fmt.Printf("%s", describe.MustDescribeOrg(org))
}
