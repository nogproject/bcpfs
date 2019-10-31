// Package `bcp` provides data structures that represent an organization with
// service facilities and research units.  The structure of the organization is
// determined from the Unix groups and a configuration that describes group
// prefixes and the association between services and facilities.
//
// Use `New()` to parse the Unix groups and return an `Organization` instance.
package bcp

import (
	"fmt"
	"strings"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"
)

// `Names` contains lists of names as parsed from the Unix groups.  It is used
// internally and might be of little use externally.
type Names struct {
	OrgUnits   []string
	Services   []string
	Facilities []string
}

// `Organization` represents an organization with service facilities and
// organizational units that use the services.
type Organization struct {
	OrgUnits   []OrgUnit
	Facilities []Facility
	Services   []Service
}

// `Service` represents a facility service.  Services, such as microscopes, are
// subsumed under services.  Every service must be owned by a single facility
// and describe its access policy.  `Facility` contains the facility name
// without Unix groups suffix.  Example: `Facility=em`.  `Access` describes
// whether the facility provides its services restricted per device or service
// (`perService`) and OrgUnit or to all OrgUnits (`allOrgUnits`).  In case of
// `allOrgUnits`, the access is controlled by `SuperGroup`, which contains the
// members of all OrgUnits.
type Service struct {
	Name            string
	Facility        string
	Access          AccessPolicy
	SuperGroup      grp.Group `yaml:"supergroup,omitempty"`
	ServiceGroup    grp.Group
	ServiceOpsGroup grp.Group
}

// `OrgUnit` represents an organizational unit, such as a lab or a
// collaboration project.
//
// `Name` contains the full Unix group name.  If the organzational unit is a
// facility, `IsFacility` is `true` and `Facility` contains the facility name
// without the Unix group suffix that indicates facilities.  Example:
// `Name=em-facility`, `IsFacility=true`, `Facility=em`.
//
// `Subdirs` is a list of additional directories in the org unit tree with
// access policies.  See NOE-11.  Example: `{name:projects policy:group}`
//
// `ExtraDirs` is kept for backward compatibility.  It contains the same
// directories as `Subdirs`, but only their names, without policies.  See
// NOE-11.  Example: `projects`.
type OrgUnit struct {
	Name         string
	Subdirs      []DirWithPolicy
	ExtraDirs    []string
	IsFacility   bool
	Facility     string
	OrgUnitGroup grp.Group
}

// `DirWithPolicy` represents a filesystem directory with an access policy.
type DirWithPolicy struct {
	Name   string
	Policy DirPolicy
}

// `DirPolicy` enumerates directory access policies.  Its underlying type is
// `string`, instead of `int` as typical for enums, so that it will be
// marshaled to YAML strings without further ceremony.
type DirPolicy string

const (
	GroupPolicy   = "group"
	OwnerPolicy   = "owner"
	ManagerPolicy = "manager"
)

// `MustDirPolicy()` returns a `DirPolicy`, or panics if the string is invalid.
func MustDirPolicy(s string) DirPolicy {
	switch s {
	case OwnerPolicy:
		return OwnerPolicy
	case GroupPolicy:
		return GroupPolicy
	case ManagerPolicy:
		return ManagerPolicy
	default:
		panic(fmt.Sprintf("invalid DirPolicy from `%s`", s))
	}
}

// `Factility` represents a facility.
type Facility struct {
	Name string
}

// `FacilityAccess` represents a facility including its access policy to service
// directories.
type FacilityAccess struct {
	Name   string
	Access string
}

type AccessPolicy int

const (
	AccessUnspecified AccessPolicy = iota
	AccessPerService
	AccessAllOrgUnits
)

func (a AccessPolicy) IsPerService() bool {
	if a == AccessPerService {
		return true
	}
	return false
}

func (a AccessPolicy) IsAllOrgUnits() bool {
	if a == AccessAllOrgUnits {
		return true
	}
	return false
}

func AccessPolicyFromString(name string) (AccessPolicy, error) {
	if name == "perService" {
		return AccessPerService, nil
	}
	if name == "allOrgUnits" {
		return AccessAllOrgUnits, nil
	}
	if name == "" {
		return AccessPerService, nil
	}
	return AccessUnspecified, fmt.Errorf("invalid Access name `%s`", name)
}

func (a AccessPolicy) MarshalText() (text []byte, err error) {
	switch a {
	case AccessUnspecified:
		return []byte("unspecified"), nil
	case AccessPerService:
		return []byte("perService"), nil
	case AccessAllOrgUnits:
		return []byte("allOrgUnits"), nil
	default:
		return nil, fmt.Errorf(
			"Can't marshal `Access`: invalid value `%d`", a,
		)
	}
}

func New(groups []grp.Group, cfg *bcpcfg.Root) (
	*Organization, []string, error,
) {
	names, err := parseGroupNames(groups, cfg)
	if err != nil {
		return nil, nil, err
	}

	gm := NewGroupMap(groups, cfg)
	var org Organization
	if org.OrgUnits, err = parseOrgUnits(names, cfg, gm); err != nil {
		return nil, nil, err
	}

	var unconfServices []string
	org.Services, unconfServices, err = parseServices(names, cfg, gm)
	if err != nil {
		return nil, nil, err
	}

	if org.Facilities, err = parseFacilities(names, cfg); err != nil {
		return nil, nil, err
	}
	return &org, unconfServices, nil
}

func parseGroupNames(gs []grp.Group, cfg *bcpcfg.Root) (*Names, error) {
	var names Names
	names.OrgUnits = parseOrgUnitNames(gs, cfg)
	names.Services = parseServiceNames(gs, cfg)
	names.Facilities = parseFacilityNames(gs, cfg)
	return &names, nil
}

func parseServices(
	names *Names, cfg *bcpcfg.Root, gm *GroupMap,
) ([]Service, []string, error) {

	if cfg.SuperGroup == "" {
		msg := "No `SuperGroup` specified."
		logger.Info(msg)
	}

	haveF := make(map[string]bool)
	for _, f := range names.Facilities {
		haveF[f] = true
	}
	for _, f := range cfg.Facilities {
		name := f.Name
		if !haveF[name] {
			return nil, nil, fmt.Errorf(
				"missing group for facility `%s`", name,
			)
		}
	}

	var unconfServiceGroups []string
	facilityBySrv := make(map[string]FacilityAccess)
	for _, f := range cfg.Facilities {
		for _, s := range f.Services {
			facilityBySrv[s] = FacilityAccess{
				Name:   f.Name,
				Access: f.Access,
			}
		}
		if f.Access == "" {
			msg := fmt.Sprintf(
				"Facility `%s`: No access policy specified. ",
				f.Name,
			)
			msg = msg + "Assuming default policy `perService`."
			logger.Info(msg)
		}
	}

	srvs := make([]Service, 0)
	for _, s := range names.Services {
		f, ok := facilityBySrv[s]
		if !ok {
			msg := fmt.Sprintf(
				"Missing facility for service `%s`.", s,
			)
			unconfServiceGroups = append(unconfServiceGroups, msg)
			continue
		}

		access, err := AccessPolicyFromString(f.Access)
		if err != nil {
			return nil, nil, fmt.Errorf(
				"%s in service `%s`", err, s,
			)
		}

		srv := Service{
			Name:     s,
			Facility: f.Name,
			Access:   access,
		}

		if srv.Access.IsAllOrgUnits() && cfg.SuperGroup == "" {
			msg := "Can't apply `allOrgUnits` without `SuperGroup`"
			return nil, nil, fmt.Errorf(
				"Service `%s`: %s", srv.Name, msg,
			)
		}

		if g, ok := gm.FindServiceGroup(srv); !ok {
			return nil, nil, fmt.Errorf(
				"missing group for service `%s`", s,
			)
		} else {
			srv.ServiceGroup = g
		}

		// We temporarily allow an unspecified `SuperGroup` for
		// backward compatibility, which will lead to passing the
		// SuperGroup with Gid=0 to `ensureServiceSh` in `fsapply` in
		// case of `perService` policy.  This then runs `setfacl -X-`
		// with Gid=0 for root, what we consider as safe enough for
		// now.  We will make specifying the `SuperGroup` obligatory
		// after a transition period.  The `SuperGroup` will then
		// passed with its Gid, or the process stops if it does not
		// exist.
		if cfg.SuperGroup != "" {
			if g, ok := gm.GetByName(cfg.SuperGroup); !ok {
				return nil, nil, fmt.Errorf(
					"missing group for superGroup `%s`",
					cfg.SuperGroup,
				)
			} else {
				srv.SuperGroup = g
			}
		}

		if g, ok := gm.FindServiceOpsGroup(srv); !ok {
			return nil, nil, fmt.Errorf(
				"missing ops group for service `%s`", s,
			)
		} else {
			srv.ServiceOpsGroup = g
		}

		srvs = append(srvs, srv)
	}
	return srvs, unconfServiceGroups, nil
}

func parseOrgUnits(
	names *Names, cfg *bcpcfg.Root, gm *GroupMap,
) ([]OrgUnit, error) {
	// Gather lists of directories by ou.  `ExtraDirs` from `cfg` are
	// mapped to `Subdirs` with `policy:group` for backward compatibility.
	// See NOE-11.  `Subdirs` are also added to `ExtraDirs`, so that the
	// returned `Subdirs` and `ExtraDirs` both contain a complete list.
	subdirsByOu := make(map[string][]DirWithPolicy)
	extraDirsByOu := make(map[string][]string)
	for _, cou := range cfg.OrgUnits {
		used := make(map[string]bool)
		var checkErr error
		checkName := func(name string) {
			if checkErr != nil {
				return
			}
			if _, ok := used[name]; ok {
				checkErr = fmt.Errorf(
					"duplicate ou `%s` dir `%s`",
					cou.Name, name,
				)
			}
			used[name] = true
		}

		var sds []DirWithPolicy
		var xds []string
		for _, d := range cou.Subdirs {
			checkName(d.Name)
			sds = append(sds, DirWithPolicy{
				Name:   d.Name,
				Policy: MustDirPolicy(d.Policy),
			})
			xds = append(xds, d.Name)
		}

		for _, xd := range cou.ExtraDirs {
			checkName(xd)
			sds = append(sds, DirWithPolicy{
				Name:   xd,
				Policy: GroupPolicy,
			})
			xds = append(xds, xd)
		}

		if checkErr != nil {
			return nil, checkErr
		}

		subdirsByOu[cou.Name] = sds
		extraDirsByOu[cou.Name] = xds
	}

	fsuf := fmt.Sprintf("-%s", cfg.FacilitySuffix)

	ous := make([]OrgUnit, 0)
	for _, o := range names.OrgUnits {
		ou := OrgUnit{
			Name:       o,
			Subdirs:    subdirsByOu[o],
			ExtraDirs:  extraDirsByOu[o],
			IsFacility: strings.HasSuffix(o, fsuf),
		}
		if ou.IsFacility {
			ou.Facility = strings.TrimSuffix(o, fsuf)
		}
		if g, ok := gm.FindOrgUnitGroup(ou); !ok {
			return nil, fmt.Errorf(
				"missing group for org unit `%s`", o,
			)
		} else {
			ou.OrgUnitGroup = g
		}
		ous = append(ous, ou)
	}
	return ous, nil
}

func parseFacilities(names *Names, cfg *bcpcfg.Root) ([]Facility, error) {
	fs := make([]Facility, 0)
	for _, f := range names.Facilities {
		fs = append(fs, Facility{Name: f})
	}
	return fs, nil
}

// `org_<x>` -> `<x>`
func parseOrgUnitNames(gs []grp.Group, cfg *bcpcfg.Root) []string {
	pfx := fmt.Sprintf("%s_", cfg.OrgUnitPrefix)
	ous := make([]string, 0)
	for _, g := range gs {
		if strings.HasPrefix(g.Name, pfx) {
			ous = append(ous, g.Name[len(pfx):])
		}
	}
	return ous
}

// `srv_<x>` -> `<x>` unless `-ops` suffix.
func parseServiceNames(gs []grp.Group, cfg *bcpcfg.Root) []string {
	pfx := fmt.Sprintf("%s_", cfg.ServicePrefix)
	suf := fmt.Sprintf("-%s", cfg.OpsSuffix)
	ss := make([]string, 0)
	for _, g := range gs {
		if !strings.HasPrefix(g.Name, pfx) {
			continue
		}
		if strings.HasSuffix(g.Name, suf) {
			continue
		}
		ss = append(ss, g.Name[len(pfx):])
	}
	return ss
}

// `org_<x>-facility` -> `<x>`
func parseFacilityNames(gs []grp.Group, cfg *bcpcfg.Root) []string {
	pfx := fmt.Sprintf("%s_", cfg.OrgUnitPrefix)
	suf := fmt.Sprintf("-%s", cfg.FacilitySuffix)
	fs := make([]string, 0)
	for _, g := range gs {
		n := g.Name
		if !strings.HasPrefix(n, pfx) {
			continue
		}
		n = n[len(pfx):]
		if !strings.HasSuffix(n, suf) {
			continue
		}
		fs = append(fs, n[:len(n)-len(suf)])
	}
	return fs
}
