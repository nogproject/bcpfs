// vim: sw=8

package bcpcfg_test

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
)

var config_hcl = `
rootdir = "/fsroot"
superGroup = "ag_org"

facility {
    name = "microscopy"
    services = [
        "m1"
    ]
    access = "perService"
}

orgUnit {
    name = "lab"
    subdirs = [
        { name = "people", policy = "owner" },
        { name = "service", policy = "group" },
        { name = "shared", policy = "manager" },
    ]
    extraDirs = [
        "projects",
    ]
}

filter {
    service = "m1",
    orgUnit = "lab1",
    action = "accept"
}

filter {
    services = [
        "m1",
        "m2",
    ]
    orgUnits = [
        "lab1",
        "lab2",
    ]
    action = "accept"
}

`

func ExampleParseCfg() {
	// Real code would use `Load(path)`.
	cfg, err := bcpcfg.Parse(config_hcl)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(cfg.Rootdir)
	fmt.Println(cfg.SuperGroup)
	fmt.Printf("%+v\n", cfg.Facilities)
	fmt.Printf("%+v\n", cfg.OrgUnits)
	fmt.Printf("%+v\n", cfg.Filter)

	// Output:
	// /fsroot
	// ag_org
	// [{Name:microscopy Services:[m1] Access:perService}]
	// [{Name:lab Subdirs:[{Name:people Policy:owner} {Name:service Policy:group} {Name:shared Policy:manager}] ExtraDirs:[projects]}]
	// [{Services:[m1] OrgUnits:[lab1] Action:accept} {Services:[m1 m2] OrgUnits:[lab1 lab2] Action:accept}]
}

func ExampleValidateFilterRule() {
	srvOrg := "mic"
	srvOrgs := []string{"lab1", "lab2"}
	action := "accept"
	srvOrgEmpty := ""
	srvOrgsEmpty := []string{}
	actionWrong := "acceptt"

	var r bcpcfg.FilterRule
	var err error

	_, err = bcpcfg.ValidateFilterRule(bcpcfg.FilterRuleCfg{
		Service:  srvOrg,
		Services: srvOrgsEmpty,
		OrgUnit:  srvOrg,
		OrgUnits: srvOrgsEmpty,
		Action:   actionWrong,
	})
	fmt.Println(err)

	_, err = bcpcfg.ValidateFilterRule(bcpcfg.FilterRuleCfg{
		Service:  srvOrgEmpty,
		Services: srvOrgsEmpty,
		OrgUnit:  srvOrg,
		OrgUnits: srvOrgsEmpty,
		Action:   action,
	})
	fmt.Println(err)

	_, err = bcpcfg.ValidateFilterRule(bcpcfg.FilterRuleCfg{
		Service:  srvOrg,
		Services: srvOrgs,
		OrgUnit:  srvOrg,
		OrgUnits: srvOrgsEmpty,
		Action:   action,
	})
	fmt.Println(err)

	_, err = bcpcfg.ValidateFilterRule(bcpcfg.FilterRuleCfg{
		Service:  srvOrg,
		Services: srvOrgsEmpty,
		OrgUnit:  srvOrgEmpty,
		OrgUnits: srvOrgsEmpty,
		Action:   action,
	})
	fmt.Println(err)

	_, err = bcpcfg.ValidateFilterRule(bcpcfg.FilterRuleCfg{
		Service:  srvOrg,
		Services: srvOrgsEmpty,
		OrgUnit:  srvOrg,
		OrgUnits: srvOrgs,
		Action:   action,
	})
	fmt.Println(err)

	r, _ = bcpcfg.ValidateFilterRule(bcpcfg.FilterRuleCfg{
		Service:  srvOrg,
		Services: srvOrgsEmpty,
		OrgUnit:  srvOrgEmpty,
		OrgUnits: srvOrgs,
		Action:   action,
	})
	fmt.Println(r)

	// Output:
	// Invalid action!
	// No service defined!
	// Use either `service` or `services`!
	// No orgUnit defined!
	// Use either `orgUnit` or `orgUnits`!
	// {[mic] [lab1 lab2] accept}
}
