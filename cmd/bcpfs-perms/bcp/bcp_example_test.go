package bcp_test

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
)

func ExampleFacilityAccessPolicy() {

	a1 := "perService"
	a2 := "allOrgUnits"
	a3 := "something"
	a4 := ""

	access1, err1 := bcp.AccessPolicyFromString(a1)
	fmt.Println(access1, err1)
	fmt.Println(access1.IsPerService())
	fmt.Println(access1.IsAllOrgUnits())

	access2, err2 := bcp.AccessPolicyFromString(a2)
	fmt.Println(access2, err2)
	fmt.Println(access2.IsPerService())
	fmt.Println(access2.IsAllOrgUnits())

	access3, err3 := bcp.AccessPolicyFromString(a3)
	fmt.Println(access3, err3)
	fmt.Println(access3.IsPerService())
	fmt.Println(access3.IsAllOrgUnits())

	access4, err4 := bcp.AccessPolicyFromString(a4)
	fmt.Println(access4, err4)
	fmt.Println(access4.IsPerService())
	fmt.Println(access4.IsAllOrgUnits())

	// Output:
	// 1 <nil>
	// true
	// false
	// 2 <nil>
	// false
	// true
	// 0 invalid Access name `something`
	// false
	// false
	// 1 <nil>
	// true
	// false
}
