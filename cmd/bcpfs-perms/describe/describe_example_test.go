package describe

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"
)

var gs = []grp.Group{
	grp.Group{
		Name: "ag_org",
		Gid:  1,
	},
	grp.Group{
		Name: "org_lm-facility",
		Gid:  2,
	},
	grp.Group{
		Name: "srv_lm-ops",
		Gid:  3,
	},
	grp.Group{
		Name: "srv_mic1",
		Gid:  4,
	},
	grp.Group{
		Name: "srv_mic2",
		Gid:  5,
	},
	grp.Group{
		Name: "org_ag-foo",
		Gid:  6,
	},
}

func ExampleDescribeGroups() {

	d := MustDescribeGroups(gs)
	fmt.Printf("%s", d)

	//Output:
	// - name: ag_org
	//   gid: 1
	// - name: org_lm-facility
	//   gid: 2
	// - name: srv_lm-ops
	//   gid: 3
	// - name: srv_mic1
	//   gid: 4
	// - name: srv_mic2
	//   gid: 5
	// - name: org_ag-foo
	//   gid: 6
}
