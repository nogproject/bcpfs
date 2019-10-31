package fsck

import (
	"fmt"
	"path/filepath"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

// `OrgUnitTreePaths` creates paths lists for the org tree.
type OrgUnitTreePaths struct {
	root       string
	serviceDir string
	services   []bcp.Service
	orgUnits   []bcp.OrgUnit
	filter     bfilter.OrgServiceFilter
}

// `OrgUnitDirsList()` lists expected paths `/orgfs/org/<ou>`.
func (ot *OrgUnitTreePaths) OrgUnitDirsList() (list []Entry) {
	for _, o := range ot.orgUnits {
		path := filepath.Join(ot.root, o.Name)
		ouG := o.OrgUnitGroup
		list = append(list, Entry{
			Path:      path,
			IsSymlink: false,
			ACL: OrgUnitACL{
				Uid: 0,
				Gid: ouG.Gid,
			},
		})
	}
	return
}

// `OrgUnitServiceLinksList()` lists expected symlinks
// `/orgfs/org/<ou>/<link>`.  Facility symlinks point to the toplevel
// directories for services that are operated by the facility.
func (ot *OrgUnitTreePaths) OrgUnitServiceLinksList() (list []Entry) {
	serviceIsOfFacility := func(ou bcp.OrgUnit, s bcp.Service) bool {
		return ou.IsFacility && s.Facility == ou.Facility
	}

	appendOUSLn := func(ou bcp.OrgUnit, s bcp.Service) {
		path := filepath.Join(ot.root, ou.Name, s.Name)
		dest := filepath.Join("../..", ot.serviceDir, s.Name)
		if !serviceIsOfFacility(ou, s) {
			dest = filepath.Join(dest, ou.Name)
		}
		list = append(list, Entry{
			Path:      path,
			IsSymlink: true,
			LinkDest:  dest,
		})
	}

	logSkip := func(s bcp.Service, ou bcp.OrgUnit, reason string) {
		msg := fmt.Sprintf(
			"Skipped `service=%s orgUnit=%s`: %s",
			s.Name, ou.Name, reason,
		)
		logger.Debug(msg)
	}

	for _, ou := range ot.orgUnits {
		for _, s := range ot.services {
			if ok, reason := ot.filter.Accept(s, ou); ok {
				appendOUSLn(ou, s)
			} else {
				logSkip(s, ou, reason)
			}
		}
	}
	return
}

// `OrgUnitSubdirsList()` lists expected subdirs `/orgfs/org/<ou>/<subdir>`.
func (ot *OrgUnitTreePaths) OrgUnitSubdirsList() (list []Entry) {
	appendSubdir := func(o bcp.OrgUnit, d bcp.DirWithPolicy) {
		path := filepath.Join(ot.root, o.Name, d.Name)
		ouG := o.OrgUnitGroup
		ent := Entry{
			Path:      path,
			IsSymlink: false,
		}
		switch d.Policy {
		case bcp.GroupPolicy:
			ent.ACL = SubdirGroupACL{Uid: 0, Gid: ouG.Gid}
		case bcp.OwnerPolicy:
			ent.ACL = SubdirOwnerACL{Uid: 0, Gid: ouG.Gid}
		case bcp.ManagerPolicy:
			ent.ACL = SubdirManagerACL{Uid: 0, Gid: ouG.Gid}
		default:
			panic("invalid subdir policy")
		}
		list = append(list, ent)
	}

	for _, o := range ot.orgUnits {
		for _, d := range o.Subdirs {
			appendSubdir(o, d)
		}
	}
	return
}
