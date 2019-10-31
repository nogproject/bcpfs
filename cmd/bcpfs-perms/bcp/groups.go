package bcp

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"
)

// `GroupMap` provides lookup of groups.
type GroupMap struct {
	byName        map[string]grp.Group
	servicePrefix string
	orgUnitPrefix string
	opsSuffix     string
}

func NewGroupMap(groups []grp.Group, cfg *bcpcfg.Root) *GroupMap {
	byName := make(map[string]grp.Group)
	for _, g := range groups {
		byName[g.Name] = g
	}
	return &GroupMap{
		byName:        byName,
		servicePrefix: cfg.ServicePrefix,
		orgUnitPrefix: cfg.OrgUnitPrefix,
		opsSuffix:     cfg.OpsSuffix,
	}
}

func (gm *GroupMap) GetByName(name string) (g grp.Group, ok bool) {
	g, ok = gm.byName[name]
	return
}

func (gm *GroupMap) FindOrgUnitGroup(ou OrgUnit) (g grp.Group, ok bool) {
	name := fmt.Sprintf("%s_%s", gm.orgUnitPrefix, ou.Name)
	return gm.GetByName(name)
}

func (gm *GroupMap) FindServiceGroup(s Service) (g grp.Group, ok bool) {
	name := fmt.Sprintf("%s_%s", gm.servicePrefix, s.Name)
	return gm.GetByName(name)
}

func (gm *GroupMap) FindServiceOpsGroup(s Service) (g grp.Group, ok bool) {
	name := fmt.Sprintf(
		"%s_%s-%s", gm.servicePrefix, s.Facility, gm.opsSuffix,
	)
	return gm.GetByName(name)
}
