package bcp_test

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"
)

func ExampleGroupMap() {
	cfg := &bcpcfg.Root{
		ServicePrefix: "srv",
		OrgUnitPrefix: "org",
		OpsSuffix:     "ops",
	}
	groups := []grp.Group{
		{Gid: 1, Name: "org_alice"},
		{Gid: 2, Name: "org_em-facility"},
		{Gid: 3, Name: "srv_foo"},
		{Gid: 4, Name: "srv_em-ops"},
	}
	gm := bcp.NewGroupMap(groups, cfg)

	g, ok := gm.GetByName("org_alice")
	fmt.Println(g, ok)

	_, ok = gm.GetByName("org_bob")
	fmt.Println(ok)

	g, ok = gm.FindOrgUnitGroup(bcp.OrgUnit{Name: "alice"})
	fmt.Println(g, ok)

	_, ok = gm.FindOrgUnitGroup(bcp.OrgUnit{Name: "bob"})
	fmt.Println(ok)

	g, ok = gm.FindServiceGroup(bcp.Service{Name: "foo"})
	fmt.Println(g, ok)

	_, ok = gm.FindServiceGroup(bcp.Service{Name: "bar"})
	fmt.Println(ok)

	g, ok = gm.FindServiceOpsGroup(
		bcp.Service{Name: "foo", Facility: "em"},
	)
	fmt.Println(g, ok)

	_, ok = gm.FindServiceOpsGroup(
		bcp.Service{Name: "foo", Facility: "lm"},
	)
	fmt.Println(ok)

	// Output:
	// {org_alice 1} true
	// false
	// {org_alice 1} true
	// false
	// {srv_foo 3} true
	// false
	// {srv_em-ops 4} true
	// false
}
