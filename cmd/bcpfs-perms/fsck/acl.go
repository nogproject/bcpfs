package fsck

import (
	"fmt"
	"os/exec"
	"sort"
	"strings"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/pkg/execx"
)

// `ACL` provides access to the information necessary to verify current
// filesystem ACLs.
//
// `NamedGids()` returns the numeric gids of the named group ACL entries, that
// is the ou and the facility if it is a service directory.  `NamedGids()` is
// used to ignore unrelated named group entries when comparing current
// filesystem ACLs.
//
// `FACLString()` returns a string representation that can be compared to the
// output of the command line tool `getfacl`.
type ACL interface {
	NamedGids() []int
	FACLString() string
}

// `CheckACLs()` checks whether the POSIX ACLs match for the non-symlink
// `entries`.
func CheckACLs(entries []Entry) (ok bool, err error) {
	ok = true
	for _, p := range entries {
		if p.IsSymlink {
			continue
		}

		facl, err := getfacl(p.Path)
		if err != nil {
			ok = false
			msg := fmt.Sprintf(
				"failed to getfacl `%s`: %v", p.Path, err,
			)
			logger.Error(msg)
			continue
		}

		// Remove unrelated named group entries before comparison.
		facl = rejectOtherNamedGroupEntries(facl, p.ACL.NamedGids())

		expected := fmt.Sprintf(
			"# file: %s\n%s", p.Path, p.ACL.FACLString(),
		)

		if facl != expected {
			ok = false
			msg := fmt.Sprintf(
				"wrong ACL; ...\n"+
					"    expected `%s`; ...\n"+
					"    got      `%s`.",
				strings.Replace(expected, "\n", ", ", -1),
				strings.Replace(facl, "\n", ", ", -1),
			)
			logger.Error(msg)
		}
	}
	return ok, nil
}

// `rejectOtherNamedGroupEntries(facl, gids)` returns a filter version of
// `facl` with named group entries other than for `gids` removed.
func rejectOtherNamedGroupEntries(facl string, gids []int) string {
	var keep []string
	for _, line := range strings.Split(facl, "\n") {
		if isOtherNamedGroupEntry(line, gids) {
			continue
		}
		keep = append(keep, line)
	}
	return strings.Join(keep, "\n")
}

func isOtherNamedGroupEntry(l string, gids []int) bool {
	if !isNamedGroupEntry(l) {
		return false
	}
	for _, gid := range gids {
		tag := fmt.Sprintf("group:%d:", gid)
		if strings.Contains(l, tag) {
			return false
		}
	}
	return true
}

func isNamedGroupEntry(l string) bool {
	if strings.Contains(l, "# ") {
		return false
	}
	if strings.Contains(l, "group::") {
		return false
	}
	return strings.Contains(l, "group:")
}

func getfacl(path string) (string, error) {
	out, err := exec.Command(
		getfaclTool.Path,
		"-p", // absolute names.
		"-E", // no effective rights.
		"-n", // numeric ids.
		path,
	).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

var getfaclTool = execx.MustLookTool(execx.ToolSpec{
	Program:   "getfacl",
	CheckArgs: []string{"--version"},
	CheckText: "getfacl 2",
})

// `SimpleACL` is an `ACL` for simple, traditional Unix permissions.
// Permissions are represented as strings like `User=rwx`.
type SimpleACL struct {
	Uid   int
	Gid   int
	User  string
	Group string
	Other string
}

func (a SimpleACL) NamedGids() []int {
	return []int{}
}

func (a SimpleACL) FACLString() string {
	return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
user::%s
group::%s
other::%s
`,
		a.Uid, a.Gid, a.User, a.Group, a.Other,
	))
}

// `ServiceACL` for `/orgfs/srv/<srv>` directories.  See NOE-10.
type ServiceACL struct {
	Uid           int
	ServiceGid    int
	ServiceOpsGid int
	SuperGid      int
	Access        bcp.AccessPolicy
}

func (a ServiceACL) NamedGids() []int {
	return []int{a.ServiceGid, a.ServiceOpsGid, a.SuperGid}
}

func (a ServiceACL) FACLString() string {
	gids := []int{a.ServiceGid, a.ServiceOpsGid}
	sort.Ints(gids)

	if a.Access.IsPerService() {
		return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
user::rwx
group::---
group:%d:r-x
group:%d:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:%d:r-x
default:group:%d:r-x
default:mask::r-x
default:other::---
`,
			a.Uid, a.ServiceGid, // header
			gids[0], gids[1], // group:...
			gids[0], gids[1], // default:group:...
		))
	}
	if a.Access.IsAllOrgUnits() {
		return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
user::rwx
group::---
group:%d:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:%d:r-x
default:mask::r-x
default:other::---
`,
			a.Uid, a.ServiceGid, // header
			a.SuperGid, // group:...
			a.SuperGid, // default:group:...
		))
	}
	panic("Invalid AccessPolicy")
}

// `ServiceOrgUnitACL` for `/orgfs/srv/*/<ou>` directories.  See NOE-10.
type ServiceOrgUnitACL struct {
	Uid           int
	OrgUnitGid    int
	ServiceOpsGid int
	// Include `ServiceGid` and `SuperGid` to be able to check that there is no
	// named group ACL entry for it, which confirms that the `mkdir` path
	// removed the default srv ACL entry from the parent dir.
	ServiceGid int
	SuperGid   int
}

func (a ServiceOrgUnitACL) NamedGids() []int {
	return []int{a.OrgUnitGid, a.ServiceGid, a.ServiceOpsGid, a.SuperGid}
}

func (a ServiceOrgUnitACL) FACLString() string {
	gids := []int{a.OrgUnitGid, a.ServiceOpsGid}
	sort.Ints(gids)
	return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
# flags: -s-
user::rwx
group::---
group:%d:rwx
group:%d:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:%d:rwx
default:group:%d:rwx
default:mask::rwx
default:other::---
`,
		a.Uid, a.OrgUnitGid, // header
		gids[0], gids[1], // group:...
		gids[0], gids[1], // default:group:...
	))
}

// `OrgUnitACL` for `/orgfs/org/<ou>` directories.  See NOE-10.
type OrgUnitACL struct {
	Uid int
	Gid int
}

func (a OrgUnitACL) NamedGids() []int {
	return []int{a.Gid}
}

func (a OrgUnitACL) FACLString() string {
	return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
# flags: -s-
user::rwx
group::---
group:%d:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:%d:r-x
default:mask::r-x
default:other::---
`,
		a.Uid, a.Gid, // header
		a.Gid, // group:...
		a.Gid, // default:group:...
	))
}

// `SubdirGroupACL` for `/orgfs/org/<ou>/<subdir>` directories.  See NOE-11.
type SubdirGroupACL struct {
	Uid int
	Gid int
}

func (a SubdirGroupACL) NamedGids() []int {
	return []int{a.Gid}
}

func (a SubdirGroupACL) FACLString() string {
	return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
# flags: -s-
user::rwx
group::---
group:%d:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:%d:rwx
default:mask::rwx
default:other::---
`,
		a.Uid, a.Gid, // header
		a.Gid, // group:...
		a.Gid, // default:group:...
	))
}

// `SubdirOwnerACL` for `/orgfs/org/<ou>/<subdir>` directories.  See NOE-11.
type SubdirOwnerACL struct {
	Uid int
	Gid int
}

func (a SubdirOwnerACL) NamedGids() []int {
	return []int{a.Gid}
}

func (a SubdirOwnerACL) FACLString() string {
	return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
# flags: -s-
user::rwx
group::---
group:%d:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:%d:r-x
default:mask::r-x
default:other::---
`,
		a.Uid, a.Gid, // header
		a.Gid, // group:...
		a.Gid, // default:group:...
	))
}

// `SubdirManagerACL` for `/orgfs/org/<ou>/<subdir>` directories.  See NOE-11.
type SubdirManagerACL struct {
	Uid int
	Gid int
}

func (a SubdirManagerACL) NamedGids() []int {
	return []int{a.Gid}
}

func (a SubdirManagerACL) FACLString() string {
	return strings.TrimSpace(fmt.Sprintf(`
# owner: %d
# group: %d
# flags: -s-
user::rwx
group::---
group:%d:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:%d:r-x
default:mask::r-x
default:other::---
`,
		a.Uid, a.Gid, // header
		a.Gid, // group:...
		a.Gid, // default:group:...
	))
}
