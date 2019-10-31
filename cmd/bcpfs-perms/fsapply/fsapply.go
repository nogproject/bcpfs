/*
Package `fsapply` provides `EnsurePermissions()` to create directories with the
expected permissions.
*/
package fsapply

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

type Options struct {
	Recursive bool
}

// `EnsurePermissions()` iterates over the toplevel directories, creating
// missing directories and applying the expected permissions.  Directories are
// only created if they pass the `filter`.
//
// It delegates to `ServiceTree` and `OrgUnitDir` for the respective subtrees.
func EnsurePermissions(
	cfg *bcpcfg.Root,
	org *bcp.Organization,
	filter bfilter.OrgServiceFilter,
	opts *Options,
) error {
	root, err := filepath.Abs(cfg.Rootdir)
	if err != nil {
		return err
	}
	serviceRoot := filepath.Join(root, cfg.ServiceDir)
	orgUnitRoot := filepath.Join(root, cfg.OrgUnitDir)

	if err := ensureRootDir(serviceRoot); err != nil {
		return fmt.Errorf("dir `%s`: %v", serviceRoot, err)
	}

	sTree := ServiceTree{
		root:      serviceRoot,
		services:  org.Services,
		orgUnits:  org.OrgUnits,
		filter:    filter,
		recursive: opts.Recursive,
	}
	sTree.EnsureServiceDirs()
	sTree.EnsureServiceOrgUnitDirs()
	if err := sTree.err; err != nil {
		return fmt.Errorf("service dirs: %v", err)
	}

	if err := ensureRootDir(orgUnitRoot); err != nil {
		return fmt.Errorf("dir `%s`: %v", orgUnitRoot, err)
	}
	ouTree := OrgUnitTree{
		root:       orgUnitRoot,
		serviceDir: cfg.ServiceDir,
		services:   org.Services,
		orgUnits:   org.OrgUnits,
		filter:     filter,
		recursive:  opts.Recursive,
	}
	ouTree.EnsureOrgUnitDirs()
	ouTree.EnsureOrgUnitServiceLinks()
	ouTree.EnsureOrgUnitSubdirs()
	if err := ouTree.err; err != nil {
		return fmt.Errorf("org unit dirs: %v", err)
	}

	for _, link := range cfg.Symlinks {
		if err := ensureSymlink(
			link.Target, filepath.Join(root, link.Path),
		); err != nil {
			return fmt.Errorf("symlink: %v", err)
		}
	}

	return nil
}

func ensureRootDir(path string) (err error) {
	if dirIsMissing(path) {
		defer func() {
			if err != nil {
				return
			}
			msg := fmt.Sprintf("Created `%s`.", path)
			logger.Info(msg)
		}()
	}
	return runBash(ensureToplevelSh, struct{ Path string }{path})
}

// `dirIsMissing()` returns true if the path is missing.  It ignores errors; it
// should be used for reporting.
func dirIsMissing(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return false
	}
	if os.IsNotExist(err) {
		return true
	}
	return false
}

func ensureSymlink(dest, path string) error {
	if isDestSymlink(dest, path) {
		return nil
	}
	if err := os.Symlink(dest, path); err != nil {
		return fmt.Errorf(
			"failed to create explicit symlink `%s`: %v",
			path, err,
		)
	}
	msg := fmt.Sprintf("Created explicit symlink `%s`.", path)
	logger.Info(msg)
	return nil
}
