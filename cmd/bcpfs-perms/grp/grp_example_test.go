package grp_test

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"
)

func Example() {
	all, err := grp.Groups()
	_ = err
	root := grp.SelectGroups(all, []string{"root"}, []string{})
	fmt.Println(len(root))
	fmt.Println(root[0].Name, root[0].Gid)

	// Output:
	// 1
	// root 0
}

func ExampleSelectGroups() {
	all := []grp.Group{
		grp.Group{
			Name: "root",
			Gid:  0,
		},
		grp.Group{
			Name: "ag_org",
			Gid:  1,
		},
		grp.Group{
			Name: "org_ms-facility",
			Gid:  2,
		},
		grp.Group{
			Name: "srv_mic1",
			Gid:  3,
		},
		grp.Group{
			Name: "org_ag-foo",
			Gid:  4,
		},
		grp.Group{
			Name: "bar",
			Gid:  5,
		},
	}

	gs := grp.SelectGroups(all,
		[]string{"org", "srv"},
		[]string{"ag_org"},
	)

	for _, g := range gs {
		fmt.Printf("%+v\n", g)
	}

	//Output:
	// {Name:ag_org Gid:1}
	// {Name:org_ms-facility Gid:2}
	// {Name:srv_mic1 Gid:3}
	// {Name:org_ag-foo Gid:4}
}

func ExampleDedupGroups() {
	gs := []grp.Group{
		grp.Group{"foo", 1},
		grp.Group{"foo", 1},
	}
	gs, err := grp.DedupGroups(gs)

	fmt.Println("error:", err)
	for _, g := range gs {
		fmt.Printf("%+v\n", g)
	}

	//Output:
	// error: <nil>
	// {Name:foo Gid:1}
}

func ExampleDedupGroupsDiffGid() {
	gs := []grp.Group{
		grp.Group{"foo", 1},
		grp.Group{"foo", 2},
	}
	gs, err := grp.DedupGroups(gs)

	fmt.Println("error:", err)
	for _, g := range gs {
		fmt.Printf("%+v\n", g)
	}

	//Output:
	// error: conflicting groups 1(foo) and 2(foo)
}

func ExampleDedupGroupsDiffName() {
	gs := []grp.Group{
		grp.Group{"foo", 1},
		grp.Group{"bar", 1},
	}
	gs, err := grp.DedupGroups(gs)

	fmt.Println("error:", err)
	for _, g := range gs {
		fmt.Printf("%+v\n", g)
	}

	//Output:
	// error: conflicting groups 1(foo) and 1(bar)
}
