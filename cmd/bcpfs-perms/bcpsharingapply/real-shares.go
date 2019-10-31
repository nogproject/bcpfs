package bcpsharingapply

import (
	"fmt"
	"sort"
	"strings"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpsharing"
)

// `EnsureRealShares()` applies the filesystem ACLs for `realShares`.
func EnsureRealShares(
	lg Logger,
	fs *bcpsharing.Bcpfs,
	realShares bcpsharing.RealExports,
) error {
	root := fs.Rootdir
	facls, err := getfaclDirPaths(root, realShares.Paths())
	if err != nil {
		return err
	}

	// Return early if none of the real shares exist on the filesystem.
	if len(facls) == 0 {
		return nil
	}

	faclsByPath := make(map[string]bcpsharing.Facl)
	for _, facl := range facls {
		faclsByPath[facl.Path] = facl.Acl
	}

	for _, rs := range realShares {
		path := rs.Path
		actual, ok := faclsByPath[path]
		if !ok {
			continue
		}

		desired := rs.Acl.AsFacl(fs)
		additionalGroups := fs.FsGroups(rs.ManagingGroups)
		if err := ensureFacl(
			lg,
			root, path,
			actual, desired,
			additionalGroups,
		); err != nil {
			return err
		}
	}

	return nil
}

func ensureFacl(
	lg Logger,
	root, path string,
	actual, desired bcpsharing.Facl,
	additionalGroups []string,
) error {
	if faclNeedModify(actual, desired) {
		modifyDirs := make([]string, 0, 2*len(desired))
		modifyFiles := make([]string, 0, len(desired))
		for _, ace := range desired {
			str := ace.String()
			modifyDirs = append(modifyDirs,
				str,
				"default:"+str,
			)
			modifyFiles = append(modifyFiles,
				ace.WithoutX().String(),
			)
		}

		if err := setfaclDirSubdirModify(
			root, path,
			modifyDirs, modifyFiles,
		); err != nil {
			return err
		}

		lg.Info(fmt.Sprintf(
			"Updated sharing ACL %s/%s %s",
			root, path,
			strings.Join(modifyDirs, ","),
		))

	}

	removeGroups := faclGroupsToRemove(actual, desired, additionalGroups)
	if len(removeGroups) > 0 {
		setfaclRm := make([]string, 0, 2*len(removeGroups))
		for _, g := range removeGroups {
			setfaclRm = append(setfaclRm,
				"group:"+g,
				"default:group:"+g,
			)
		}

		if err := setfaclDirSubdirRemove(
			root, path,
			setfaclRm,
		); err != nil {
			return err
		}

		lg.Info(fmt.Sprintf(
			"Removed sharing ACL %s/%s %s",
			root, path,
			strings.Join(setfaclRm, ","),
		))
	}

	return nil
}

// `faclNeedModify(actual, desired)` returns true if the actual ACL contains
// named group entries that differ from the desired named group entries.
// `actual` is a full directory ACL as read from the filesystem.  `desired` is
// an ACL that contains only ordinary named group entries.
func faclNeedModify(
	actual, desired bcpsharing.Facl,
) bool {
	have := make(map[string]struct{})
	for _, ace := range actual.SelectNamedGroupEntries() {
		have[ace.String()] = struct{}{}
	}

	for _, ace := range desired {
		str := ace.String()

		// Is the normal group ACE missing?
		_, ok := have[str]
		if !ok {
			return true
		}

		// Is the default group ACE missing?
		_, ok = have["default:"+str]
		if !ok {
			return true
		}
	}

	return false
}

// `faclGroupsToRemove()` returns `actual` groups minus `desired` groups minus
// `additionalGroups`.
func faclGroupsToRemove(
	actual, desired bcpsharing.Facl,
	additionalGroups []string,
) []string {
	gSet := make(map[string]struct{})
	for _, ace := range actual.SelectNamedGroupEntries() {
		gSet[ace.GroupName()] = struct{}{}
	}
	for _, ace := range desired.SelectNamedGroupEntries() {
		delete(gSet, ace.GroupName())
	}
	for _, g := range additionalGroups {
		delete(gSet, g)
	}

	gs := make([]string, 0, len(gSet))
	for g, _ := range gSet {
		gs = append(gs, g)
	}
	sort.Strings(gs)

	return gs
}
