package fsck

import (
	"fmt"
	"path/filepath"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

// `ServiceTreePaths` creates paths lists for the service tree.
type ServiceTreePaths struct {
	root     string
	services []bcp.Service
	orgUnits []bcp.OrgUnit
	filter   bfilter.OrgServiceFilter
}

// `ServiceDirsList()` lists expected paths `/orgfs/srv/<srv>`.
func (st *ServiceTreePaths) ServiceDirsList() (list []Entry) {
	for _, s := range st.services {
		path := filepath.Join(st.root, s.Name)
		srvG := s.ServiceGroup
		superG := s.SuperGroup
		opsG := s.ServiceOpsGroup
		access := s.Access
		entry := Entry{
			Path:      path,
			IsSymlink: false,
			ACL: ServiceACL{
				Uid:           0,
				ServiceGid:    srvG.Gid,
				ServiceOpsGid: opsG.Gid,
				SuperGid:      superG.Gid,
				Access:        access,
			},
		}
		list = append(list, entry)
	}
	return
}

// `ServiceOrgUnitDirsList()` lists expected paths `/orgfs/srv/<srv>/<ou>`.
func (st *ServiceTreePaths) ServiceOrgUnitDirsList() (list []Entry) {
	appendSOU := func(s bcp.Service, ou bcp.OrgUnit) {
		path := filepath.Join(st.root, s.Name, ou.Name)

		ouG := ou.OrgUnitGroup
		srvG := s.ServiceGroup
		superG := s.SuperGroup
		opsG := s.ServiceOpsGroup

		list = append(list, Entry{
			Path:      path,
			IsSymlink: false,
			ACL: ServiceOrgUnitACL{
				Uid:           0,
				OrgUnitGid:    ouG.Gid,
				ServiceGid:    srvG.Gid,
				ServiceOpsGid: opsG.Gid,
				SuperGid:      superG.Gid,
			},
		})
	}

	logSkip := func(s bcp.Service, ou bcp.OrgUnit, reason string) {
		msg := fmt.Sprintf(
			"Skipped `service=%s orgUnit=%s`: %s",
			s.Name, ou.Name, reason,
		)
		logger.Debug(msg)
	}

	for _, s := range st.services {
		for _, ou := range st.orgUnits {
			if ok, reason := st.filter.Accept(s, ou); ok {
				appendSOU(s, ou)
			} else {
				logSkip(s, ou, reason)
			}
		}
	}

	return
}
