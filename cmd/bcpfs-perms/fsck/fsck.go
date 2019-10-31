/*
Package `fsck` provides `CheckPermissions()`, which verifies that only the
expected directories with the expected permissions are present.

The package deliberately shares little code with `fsapply` in order to
independently verify the result of `fsapply`.
*/
package fsck

import (
	"fmt"
	"path/filepath"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

// `Entry` represents a desired `Path` on disk.  It is a symlink if
// `IsSymlink=true`; `LinkDest` contains the symlink target.  For
// `IsSymlink=false`, `ACL` contains the expected POSIX ACL.
type Entry struct {
	Path      string
	IsSymlink bool
	ACL       ACL
	LinkDest  string
}

// `CheckPermissions()` verifies the toplevel filesystem structure.  It returns
// `reason=""` if all checks passed.  It logs failures and returns a `reason`
// if checks failed.  It returns an error if there was a fundamental problem
// and checks could not be fully executed.
func CheckPermissions(
	cfg *bcpcfg.Root, org *bcp.Organization,
	filter bfilter.OrgServiceFilter,
) (reason string, err error) {
	var failures []string

	root, err := filepath.Abs(cfg.Rootdir)
	if err != nil {
		return "", err
	}
	serviceRoot := filepath.Join(root, cfg.ServiceDir)
	orgUnitRoot := filepath.Join(root, cfg.OrgUnitDir)

	entries := []Entry{
		{
			Path:      root,
			IsSymlink: false,
			ACL: SimpleACL{
				Uid:   0,
				Gid:   0,
				User:  "rwx",
				Group: "r-x",
				Other: "r-x",
			},
		},
		{
			Path:      serviceRoot,
			IsSymlink: false,
			ACL: SimpleACL{
				Uid:   0,
				Gid:   0,
				User:  "rwx",
				Group: "r-x",
				Other: "r-x",
			},
		},
		{
			Path:      orgUnitRoot,
			IsSymlink: false,
			ACL: SimpleACL{
				Uid:   0,
				Gid:   0,
				User:  "rwx",
				Group: "r-x",
				Other: "r-x",
			},
		},
	}

	sTree := ServiceTreePaths{
		root:     serviceRoot,
		services: org.Services,
		orgUnits: org.OrgUnits,
		filter:   filter,
	}
	entries = append(entries, sTree.ServiceDirsList()...)
	entries = append(entries, sTree.ServiceOrgUnitDirsList()...)

	ouTree := OrgUnitTreePaths{
		root:       orgUnitRoot,
		serviceDir: cfg.ServiceDir,
		services:   org.Services,
		orgUnits:   org.OrgUnits,
		filter:     filter,
	}
	entries = append(entries, ouTree.OrgUnitDirsList()...)
	entries = append(entries, ouTree.OrgUnitServiceLinksList()...)
	entries = append(entries, ouTree.OrgUnitSubdirsList()...)

	explicitSymlinks := make(map[string]string)
	for _, link := range cfg.Symlinks {
		explicitSymlinks[filepath.Join(root, link.Path)] = link.Target
	}

	if ok, err := CheckNoUnexpected(
		serviceRoot, entries, explicitSymlinks,
	); err != nil {
		return "", err
	} else if !ok {
		failures = append(failures, "no-unexpected-srv")
	}

	if ok, err := CheckNoUnexpected(
		orgUnitRoot, entries, explicitSymlinks,
	); err != nil {
		return "", err
	} else if !ok {
		failures = append(failures, "no-unexpected-ou")
	}

	if ok, err := CheckSymlinks(entries); err != nil {
		return "", err
	} else if !ok {
		failures = append(failures, "symlinks")
	}

	if ok, err := CheckExplicitSymlinks(explicitSymlinks); err != nil {
		return "", err
	} else if !ok {
		failures = append(failures, "explicit-symlinks")
	}

	if ok, err := CheckACLs(entries); err != nil {
		return "", err
	} else if !ok {
		failures = append(failures, "acls")
	}

	if len(failures) > 0 {
		return fmt.Sprintf("checks failed: %s", failures), nil
	}

	return "", nil
}
