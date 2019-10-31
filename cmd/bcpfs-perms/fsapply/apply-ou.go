package fsapply

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

// `OrgUnitTree` manages the `/orgfs/org` subtree.
type OrgUnitTree struct {
	root       string
	serviceDir string
	services   []bcp.Service
	orgUnits   []bcp.OrgUnit
	filter     bfilter.OrgServiceFilter
	recursive  bool
	err        error
}

// `EnsureOrgUnitDirs` manages the `/orgfs/org/<ou>/` dirs.
func (ot *OrgUnitTree) EnsureOrgUnitDirs() {
	if ot.err != nil {
		return
	}
	for _, o := range ot.orgUnits {
		path := filepath.Join(ot.root, o.Name)
		ouG := o.OrgUnitGroup
		err := ensureOrgUnitDir(path, ouG.Gid)
		if err != nil {
			ot.err = fmt.Errorf(
				"org unit dir `%s`: %v", o.Name, err,
			)
			return
		}
	}
}

func ensureOrgUnitDir(path string, gid int) (err error) {
	if dirIsMissing(path) {
		defer func() {
			if err != nil {
				return
			}
			msg := fmt.Sprintf("Created `%s`.", path)
			logger.Info(msg)
		}()
	}
	data := struct {
		Path string
		Gid  int
	}{path, gid}
	return runBash(ensureOrgUnitSh, data)
}

// `EnsureOrgUnitServiceLinks()` creates symlinks `/orgfs/org/<ou>/<service>`
// from org units to services.  It leaves existing links alone.
//
// Symlinks point to the toplevel directories of a service if it is operated by
// the facility.
func (ot *OrgUnitTree) EnsureOrgUnitServiceLinks() {
	for _, ou := range ot.orgUnits {
		ot.ensureOULinks(ou)
	}
}

func (ot *OrgUnitTree) ensureOULinks(ou bcp.OrgUnit) {
	logSkip := func(s bcp.Service, ou bcp.OrgUnit, reason string) {
		msg := fmt.Sprintf(
			"Skipped `service=%s orgUnit=%s`: %s",
			s.Name, ou.Name, reason,
		)
		logger.Debug(msg)
	}

	expected := make(map[string]bool)
	for _, s := range ot.services {
		if ok, reason := ot.filter.Accept(s, ou); ok {
			expected[s.Name] = true
			ot.ensureOUSLn(ou, s)
		} else {
			logSkip(s, ou, reason)
		}
	}

	ot.rmUnexpectedLinks(ou, expected)
}

func (ot *OrgUnitTree) ensureOUSLn(ou bcp.OrgUnit, s bcp.Service) {
	// Do not not use `ln -sf $dest $path`, since it is not atomic; see:
	//
	// ```
	// strace ln -sf a b 2>&1 | grep link
	// ```

	serviceIsOfFacility := func(ou bcp.OrgUnit, s bcp.Service) bool {
		return ou.IsFacility && s.Facility == ou.Facility
	}

	if ot.err != nil {
		return
	}
	path := filepath.Join(ot.root, ou.Name, s.Name)
	dest := filepath.Join("../..", ot.serviceDir, s.Name)
	if !serviceIsOfFacility(ou, s) {
		dest = filepath.Join(dest, ou.Name)
	}
	if isDestSymlink(dest, path) {
		return
	}
	if err := os.Symlink(dest, path); err != nil {
		ot.err = fmt.Errorf(
			"failed to create service symlink `%s`: %v",
			path, err,
		)
		return
	}
	msg := fmt.Sprintf("Created symlink `%s`.", path)
	logger.Info(msg)
}

func (ot *OrgUnitTree) rmUnexpectedLinks(
	ou bcp.OrgUnit, expected map[string]bool,
) {
	if ot.err != nil {
		return
	}

	ouDir := filepath.Join(ot.root, ou.Name)
	children, err := ioutil.ReadDir(ouDir)
	if err != nil {
		ot.err = err
		return
	}

	for _, child := range children {
		// Only look at symlinks.
		if child.Mode()&os.ModeSymlink == 0 {
			continue
		}

		name := child.Name()
		if expected[name] {
			continue
		}

		path := filepath.Join(ouDir, name)
		err := os.Remove(path)
		if err != nil {
			ot.err = err
			return
		}
		msg := fmt.Sprintf("Removed `%s`.", path)
		logger.Info(msg)
	}
}

// `isDestSymlink()` is `true` if `path` is a symlink to `dest`.  Errors are
// reported as `false`.
func isDestSymlink(dest string, path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return false
	}
	actual, err := os.Readlink(path)
	if err != nil {
		return false
	}
	return (actual == dest)
}

// `EnsureOrgUnitSubdirs` manages `/orgfs/org/<ou>/<dir>` directories.
func (ot *OrgUnitTree) EnsureOrgUnitSubdirs() {
	ensureSubdir := func(o bcp.OrgUnit, d bcp.DirWithPolicy) {
		if ot.err != nil {
			return
		}
		path := filepath.Join(ot.root, o.Name, d.Name)
		ouG := o.OrgUnitGroup
		err := ensureOrgUnitSubdir(path, ouG.Gid, d.Policy)
		if err != nil {
			ot.err = fmt.Errorf(
				"org unit dir `%s`: %v", o.Name, err,
			)
			return
		}
		if ot.recursive {
			err := ensureOrgUnitSubdirRecursive(
				path, ouG.Gid, d.Policy,
			)
			if err != nil {
				ot.err = fmt.Errorf(
					"org unit dir `%s`: %v", o.Name, err,
				)
				return
			}
		}
	}

	for _, o := range ot.orgUnits {
		for _, d := range o.Subdirs {
			ensureSubdir(o, d)
		}
	}
}

func ensureOrgUnitSubdir(
	path string, gid int, policy bcp.DirPolicy,
) (err error) {
	if dirIsMissing(path) {
		defer func() {
			if err != nil {
				return
			}
			msg := fmt.Sprintf("Created `%s`.", path)
			logger.Info(msg)
		}()
	}

	data := struct {
		Path string
		Gid  int
	}{path, gid}
	switch policy {
	case bcp.GroupPolicy:
		return runBash(ensureOrgUnitGroupSubdirSh, data)
	case bcp.OwnerPolicy:
		return runBash(ensureOrgUnitOwnerSubdirSh, data)
	case bcp.ManagerPolicy:
		return runBash(ensureOrgUnitManagerSubdirSh, data)
	default:
		panic(fmt.Sprintf("unsupported dir policy `%s`", policy))
	}
}

func ensureOrgUnitSubdirRecursive(
	path string, gid int, policy bcp.DirPolicy,
) (err error) {
	data := struct {
		Path string
		Gid  int
	}{path, gid}
	switch policy {
	case bcp.GroupPolicy:
		return runBash(ensureOrgUnitGroupSubdirRecursiveSh, data)
	case bcp.OwnerPolicy:
		return runBash(ensureOrgUnitOwnerSubdirRecursiveSh, data)
	case bcp.ManagerPolicy:
		return runBash(ensureOrgUnitManagerSubdirRecursiveSh, data)
	default:
		panic(fmt.Sprintf("unsupported dir policy `%s`", policy))
	}
}
