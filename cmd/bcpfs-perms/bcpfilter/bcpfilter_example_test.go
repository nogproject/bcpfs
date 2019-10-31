package bcpfilter_test

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpfilter"
)

func ExampleDecidersFilter() {
	var deciders []bcpfilter.Decider
	// One "^..." and one "...$" to demonstrate automatic achoring.
	acceptCharly, _ := bcpfilter.NewRegexpDecider(bcpcfg.FilterRule{
		Services: []string{"^.*"},
		OrgUnits: []string{"ag-charly$"},
		Action:   "accept",
	})
	deciders = append(deciders, acceptCharly)
	deciders = append(deciders, bcpfilter.NewSameFacilityDecider())
	filter := bcpfilter.DecidersFilter{Rules: deciders}

	ok, reason := filter.Accept(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "ag-charly", IsFacility: false},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "foo-facility", IsFacility: true},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "bar-facility", IsFacility: true},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "ag-bar", IsFacility: false},
	)
	fmt.Println(ok, reason)

	// Output:
	// true service=~/^.*$/ and orgUnit=~/^ag-charly$/
	// true facilty foo-facility owns service micro
	// false facilty bar-facility does not own service micro
	// false no rule accepted
}

func ExampleSameFacilityDecider() {
	decider := bcpfilter.NewSameFacilityDecider()

	act, reason := decider.Decide(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "foo-facility", IsFacility: true},
	)
	fmt.Println(act, reason)

	act, reason = decider.Decide(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "bar-facility", IsFacility: true},
	)
	fmt.Println(act, reason)

	act, reason = decider.Decide(
		bcp.Service{Name: "micro", Facility: "foo"},
		bcp.OrgUnit{Name: "ag-bar", IsFacility: false},
	)
	fmt.Println(act, reason)

	// Output:
	// ACCEPT facilty foo-facility owns service micro
	// REJECT facilty bar-facility does not own service micro
	// PASS ag-bar is not a facility
}

func ExampleRegexpDecider() {
	var deciders []bcpfilter.Decider
	var act bcpfilter.Action
	var reason string

	fmt.Println("Decider: accept rule")

	rA := bcpcfg.FilterRule{}
	rA.Services = []string{"fake.*"}
	rA.OrgUnits = []string{".*fake.*"}
	rA.Action = "accept"
	regA, _ := bcpfilter.NewRegexpDecider(rA)
	deciders = append(deciders, regA)

	act, reason = regA.Decide(
		bcp.Service{Name: "fake-micro"},
		bcp.OrgUnit{Name: "ag-fake"},
	)
	fmt.Println(act, reason)

	act, reason = regA.Decide(
		bcp.Service{Name: "fake-micro"},
		bcp.OrgUnit{Name: "ag-foo"},
	)
	fmt.Println(act, reason)

	fmt.Println("Decider: reject rule")

	rR := bcpcfg.FilterRule{}
	rR.Services = []string{"fake.*"}
	rR.OrgUnits = []string{"ag-.*"}
	rR.Action = "reject"
	regR, _ := bcpfilter.NewRegexpDecider(rR)
	deciders = append(deciders, regR)

	act, reason = regR.Decide(
		bcp.Service{Name: "fake-micro"},
		bcp.OrgUnit{Name: "fake-facility"},
	)
	fmt.Println(act, reason)

	act, reason = regR.Decide(
		bcp.Service{Name: "fake-micro"},
		bcp.OrgUnit{Name: "ag-foo"},
	)
	fmt.Println(act, reason)

	fmt.Println("Decider: accept rule, multiple services")

	rA_MultiService := bcpcfg.FilterRule{}
	rA_MultiService.Services = []string{"em-micro", "lm-micro1", "lm-micro2"}
	rA_MultiService.OrgUnits = []string{".*"}
	rA_MultiService.Action = "accept"
	regA_MultiService, _ := bcpfilter.NewRegexpDecider(rA_MultiService)
	deciders = append(deciders, regA_MultiService)

	act, reason = regA_MultiService.Decide(
		bcp.Service{Name: "lm-micro1"},
		bcp.OrgUnit{Name: "ag-alice"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiService.Decide(
		bcp.Service{Name: "lm-micro2"},
		bcp.OrgUnit{Name: "ag-bob"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiService.Decide(
		bcp.Service{Name: "em-micro"},
		bcp.OrgUnit{Name: "ag-alice"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiService.Decide(
		bcp.Service{Name: "em-micro"},
		bcp.OrgUnit{Name: "ag-bob"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiService.Decide(
		bcp.Service{Name: "ms-micro"},
		bcp.OrgUnit{Name: "ag-bob"},
	)
	fmt.Println(act, reason)

	fmt.Println("Decider: accept rule, multiple orgUnits")

	rA_MultiOrgUnits := bcpcfg.FilterRule{}
	rA_MultiOrgUnits.Services = []string{"ms-data"}
	rA_MultiOrgUnits.OrgUnits = []string{"ag-foo", "ag-bar"}
	rA_MultiOrgUnits.Action = "accept"
	regA_MultiOrgUnits, _ := bcpfilter.NewRegexpDecider(rA_MultiOrgUnits)
	deciders = append(deciders, regA_MultiOrgUnits)

	act, reason = regA_MultiOrgUnits.Decide(
		bcp.Service{Name: "ms-data"},
		bcp.OrgUnit{Name: "ag-foo"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiOrgUnits.Decide(
		bcp.Service{Name: "ms-data"},
		bcp.OrgUnit{Name: "ag-bar"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiOrgUnits.Decide(
		bcp.Service{Name: "ms-data"},
		bcp.OrgUnit{Name: "ag-alice"},
	)
	fmt.Println(act, reason)

	act, reason = regA_MultiOrgUnits.Decide(
		bcp.Service{Name: "ms-spec"},
		bcp.OrgUnit{Name: "ag-foo"},
	)
	fmt.Println(act, reason)

	fmt.Println("Filter:")

	filter := bcpfilter.DecidersFilter{Rules: deciders}
	ok, reason := filter.Accept(
		bcp.Service{Name: "fake-micro"},
		bcp.OrgUnit{Name: "ag-fake"},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "fake-micro"},
		bcp.OrgUnit{Name: "ag-foo"},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "em-micro"},
		bcp.OrgUnit{Name: "ag-king"},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "ms-data"},
		bcp.OrgUnit{Name: "ag-foo"},
	)
	fmt.Println(ok, reason)

	ok, reason = filter.Accept(
		bcp.Service{Name: "ms-data"},
		bcp.OrgUnit{Name: "ag-king"},
	)
	fmt.Println(ok, reason)

	// Output:
	// Decider: accept rule
	// ACCEPT service=~/^fake.*$/ and orgUnit=~/^.*fake.*$/
	// PASS orgUnit!~/^.*fake.*$/
	// Decider: reject rule
	// PASS orgUnit!~/^ag-.*$/
	// REJECT service=~/^fake.*$/ and orgUnit=~/^ag-.*$/
	// Decider: accept rule, multiple services
	// ACCEPT service=~/^(em-micro|lm-micro1|lm-micro2)$/ and orgUnit=~/^.*$/
	// ACCEPT service=~/^(em-micro|lm-micro1|lm-micro2)$/ and orgUnit=~/^.*$/
	// ACCEPT service=~/^(em-micro|lm-micro1|lm-micro2)$/ and orgUnit=~/^.*$/
	// ACCEPT service=~/^(em-micro|lm-micro1|lm-micro2)$/ and orgUnit=~/^.*$/
	// PASS service!~/^(em-micro|lm-micro1|lm-micro2)$/
	// Decider: accept rule, multiple orgUnits
	// ACCEPT service=~/^ms-data$/ and orgUnit=~/^(ag-foo|ag-bar)$/
	// ACCEPT service=~/^ms-data$/ and orgUnit=~/^(ag-foo|ag-bar)$/
	// PASS orgUnit!~/^(ag-foo|ag-bar)$/
	// PASS service!~/^ms-data$/
	// Filter:
	// true service=~/^fake.*$/ and orgUnit=~/^.*fake.*$/
	// false service=~/^fake.*$/ and orgUnit=~/^ag-.*$/
	// true service=~/^(em-micro|lm-micro1|lm-micro2)$/ and orgUnit=~/^.*$/
	// true service=~/^ms-data$/ and orgUnit=~/^(ag-foo|ag-bar)$/
	// false no rule accepted
}
