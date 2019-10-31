package fsapply

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	bfilter "github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

// `ServiceTree` manages the `/orgfs/srv` service subtree.
type ServiceTree struct {
	root      string
	services  []bcp.Service
	orgUnits  []bcp.OrgUnit
	filter    bfilter.OrgServiceFilter
	recursive bool
	err       error
}

// `EnsureServiceDirs()` manages the `/orgfs/srv/<service>/` dirs.
func (st *ServiceTree) EnsureServiceDirs() {
	if st.err != nil {
		return
	}
	for _, s := range st.services {
		path := filepath.Join(st.root, s.Name)
		srvG := s.ServiceGroup
		opsG := s.ServiceOpsGroup
		superG := s.SuperGroup
		access := s.Access
		err := ensureServiceDir(
			path, srvG.Gid, opsG.Gid, superG.Gid, access,
		)
		if err != nil {
			st.err = fmt.Errorf(
				"service dir `%s`: %v", s.Name, err,
			)
			return
		}
	}
}

// `EnsureServiceOrgUnitDirs()` manages the `/orgfs/srv/<service>/<ou>` dirs.
func (st *ServiceTree) EnsureServiceOrgUnitDirs() {
	for _, s := range st.services {
		st.ensureSrvSubdirs(s)
	}
}

func (st *ServiceTree) ensureSrvSubdirs(s bcp.Service) {
	if st.err != nil {
		return
	}

	logSkip := func(s bcp.Service, ou bcp.OrgUnit, reason string) {
		msg := fmt.Sprintf(
			"Skipped `service=%s orgUnit=%s`: %s",
			s.Name, ou.Name, reason,
		)
		logger.Debug(msg)
	}

	expected := make(map[string]bool)
	for _, ou := range st.orgUnits {
		if ok, reason := st.filter.Accept(s, ou); ok {
			expected[ou.Name] = true
			st.ensureSOU(s, ou)
		} else {
			logSkip(s, ou, reason)
		}
	}

	st.rmUnexpectedSubdirs(s, expected)
}

func (st *ServiceTree) ensureSOU(s bcp.Service, ou bcp.OrgUnit) {
	if st.err != nil {
		return
	}
	path := filepath.Join(st.root, s.Name, ou.Name)
	wasMissing := dirIsMissing(path)
	ouG := ou.OrgUnitGroup
	srvG := s.ServiceGroup
	opsG := s.ServiceOpsGroup
	superG := s.SuperGroup
	data := struct {
		Path     string
		Gid      int
		SrvGid   int
		OpsGid   int
		SuperGid int
	}{path, ouG.Gid, srvG.Gid, opsG.Gid, superG.Gid}
	if err := runBash(ensureSOUSh, data); err != nil {
		st.err = err
		return
	}
	if wasMissing {
		msg := fmt.Sprintf("Created `%s`.", path)
		logger.Info(msg)
	}
	if st.recursive {
		if err := runBash(ensureSOURecursiveSh, data); err != nil {
			st.err = err
			return
		}
	}
}

func (st *ServiceTree) rmUnexpectedSubdirs(
	s bcp.Service, expected map[string]bool,
) {
	if st.err != nil {
		return
	}

	srvDir := filepath.Join(st.root, s.Name)
	children, err := ioutil.ReadDir(srvDir)
	if err != nil {
		st.err = err
		return
	}

	for _, child := range children {
		// Only look at directories.
		if child.Mode()&os.ModeDir == 0 {
			continue
		}

		name := child.Name()
		if expected[name] {
			continue
		}

		// Non-empty directories as logged as info.  Other
		// errors stop processing.
		path := filepath.Join(srvDir, name)
		err := os.Remove(path)
		if err == nil {
			msg := fmt.Sprintf("Removed `%s`.", path)
			logger.Info(msg)
			continue
		}
		if err.(*os.PathError).Err != syscall.ENOTEMPTY {
			st.err = err
			return
		}
		msg := fmt.Sprintf(
			"Kept unexpected directory `%s`.", path,
		)
		logger.Info(msg)
	}
}

func ensureServiceDir(
	path string, gid int, opsGid int, superGid int, access bcp.AccessPolicy,
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
		Path     string
		Gid      int
		OpsGid   int
		SuperGid int
	}{path, gid, opsGid, superGid}

	if access.IsAllOrgUnits() {
		return runBash(ensureServiceAllOrgUnitsSh, data)
	}
	if access.IsPerService() {
		return runBash(ensureServiceSh, data)
	}
	panic("Invalid Access policy")
}
