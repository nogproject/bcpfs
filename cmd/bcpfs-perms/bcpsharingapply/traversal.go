package bcpsharingapply

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpsharing"
)

// `EnsureTraversal()` adds traversal `--x` ACL entries to the filesystem.
func EnsureTraversal(
	lg Logger,
	fs *bcpsharing.Bcpfs,
	traversal bcpsharing.RealExports,
) error {
	root := fs.Rootdir
	facls, err := getfaclDirPaths(root, traversal.Paths())
	if err != nil {
		return err
	}

	faclsByPath := make(map[string]bcpsharing.Facl)
	for _, facl := range facls {
		faclsByPath[facl.Path] = facl.Acl
	}

	// Gather paths to be modified by `<group>:--x`, so that all paths of a
	// group can be modified with a single `xargs | setfacl`.
	pathsByGroup := make(map[string][]string)
	for _, tr := range traversal {
		path := tr.Path
		actual, ok := faclsByPath[path]
		if !ok {
			continue
		}

		actualAceStrs := make(map[string]struct{})
		for _, ace := range actual.SelectNamedGroupNormalEntries() {
			actualAceStrs[ace.String()] = struct{}{}
		}

		for _, ace := range tr.Acl {
			fsGroup := fs.FsGroupOrgUnit(ace.Group)
			aceStr := "group:" + fsGroup + ":--x"
			if _, ok := actualAceStrs[aceStr]; !ok {
				pathsByGroup[fsGroup] = append(
					pathsByGroup[fsGroup], path,
				)
			}
		}
	}

	for fsGroup, paths := range pathsByGroup {
		if err := setfaclDirPathsTraversal(
			root, paths, fsGroup,
		); err != nil {
			return err
		}

		for _, p := range paths {
			lg.Info(fmt.Sprintf(
				"Added sharing traversal ACL group %s/%s %s",
				root, p, fsGroup,
			))
		}
	}

	return nil
}
