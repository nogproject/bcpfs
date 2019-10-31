/*
Package `bcpfilter` implements mechanisms to reject combinations of services
and org units.  It can be used to avoid meaningless directories.

The interface `OrgServiceFilter` is used by other packages to test whether to
`Accept(service, orgUnit)`.  `DecidersFilter` implements the interface as an
array of `Decider` instances.  The deciders are usually initialized by
startup code based on configuration settings.

Decider Constructors

`NewSameFacilityDecider()` creates a decider that decides if the org unit is a
facility and passes otherwise.  It accepts combinations of (service, org unit)
if the facility owns the service, and rejects otherwise.

`NewRegexpDecider()` creates a decider that uses regular expressions for the
service name and the org unit name.
*/
package bcpfilter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
)

type Action int

const (
	ACCEPT Action = 1 + iota
	REJECT
	PASS
)

func ActionFromString(name string) Action {
	if strings.EqualFold(name, "accept") {
		return ACCEPT
	}
	if strings.EqualFold(name, "reject") {
		return REJECT
	}
	if strings.EqualFold(name, "pass") {
		return PASS
	}
	panic("invalid Action name")
}

func (a Action) String() string {
	switch a {
	case ACCEPT:
		return "ACCEPT"
	case REJECT:
		return "REJECT"
	case PASS:
		return "PASS"
	}
	panic("invalid Action value")
}

// An `OrgServiceFilter` can be asked whether directories for a combination of
// service and org unit should be created.  `Accept()` returns the answer and a
// reason.
type OrgServiceFilter interface {
	Accept(bcp.Service, bcp.OrgUnit) (ok bool, reason string)
}

// `DecidersFilter` is an `OrgServiceFilter`.  It tests a list of decider
// `Rules`.  If a rule matches, the filter accepts or rejects according to the
// return value of the decider. It rejects by default if no rule matches.
type DecidersFilter struct {
	Rules []Decider
}

// `Decider` is the interface of `DecidersFilter` rules.
type Decider interface {
	Decide(bcp.Service, bcp.OrgUnit) (action Action, reason string)
}

func (f *DecidersFilter) Accept(
	s bcp.Service, ou bcp.OrgUnit,
) (bool, string) {
	for _, r := range f.Rules {
		if action, reason := r.Decide(s, ou); action == ACCEPT {
			return true, reason
		}
		if action, reason := r.Decide(s, ou); action == REJECT {
			return false, reason
		}
	}
	return false, "no rule accepted"
}

// A `SameFacilityDecider` decides if the org unit is a facility and passes
// otherwise.  It accepts combinations of (service, org unit) if the facility
// owns the service, and rejects otherwise.
//
// Use `NewSameFacilityDecider()` to create an instance.
type SameFacilityDecider struct{}

func NewSameFacilityDecider() Decider {
	return &SameFacilityDecider{}
}

func (r *SameFacilityDecider) Decide(
	s bcp.Service, ou bcp.OrgUnit,
) (Action, string) {
	if !ou.IsFacility {
		return PASS, fmt.Sprintf("%s is not a facility", ou.Name)
	}
	if strings.HasPrefix(ou.Name, s.Facility) {
		return ACCEPT, fmt.Sprintf(
			"facilty %s owns service %s", ou.Name, s.Name,
		)
	}
	return REJECT, fmt.Sprintf(
		"facilty %s does not own service %s", ou.Name, s.Name,
	)

}

// `RegexpDecider` evaluates combinations of (service, org unit) based on a
// pair of regexes.  A combination is passed to the next rule if one or both
// regexes do not match.  If both regexes match, the combination is treated
// according to one of the two `action` modes 'accept' or 'reject'.
//
// Regexes are automatically anchored to the beginning "^" and end "$".
//
// Use `NewRegexpDecider()` to create an instance.
type RegexpDecider struct {
	action         Action
	servicePattern string
	orgUnitPattern string
	serviceRgx     *regexp.Regexp
	orgUnitRgx     *regexp.Regexp
}

func strings2Pattern(list []string) string {
	if len(list) == 1 {
		return anchoredPattern(list[0])
	}

	p := strings.Join(list, "|")
	p = "(" + p + ")"
	return anchoredPattern(p)
}

func anchoredPattern(p string) string {
	if len(p) == 0 {
		return "^$"
	}
	if p[0] != '^' {
		p = "^" + p
	}
	if p[len(p)-1] != '$' {
		p = p + "$"
	}
	return p
}

func NewRegexpDecider(r bcpcfg.FilterRule) (Decider, error) {
	res := &RegexpDecider{
		action:         ActionFromString(r.Action),
		servicePattern: strings2Pattern(r.Services),
		orgUnitPattern: strings2Pattern(r.OrgUnits),
	}

	var err error
	res.serviceRgx, err = regexp.Compile(res.servicePattern)
	if err != nil {
		return nil, err
	}
	res.orgUnitRgx, err = regexp.Compile(res.orgUnitPattern)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (r *RegexpDecider) Decide(
	s bcp.Service, ou bcp.OrgUnit,
) (Action, string) {
	if !r.serviceRgx.MatchString(s.Name) {
		return PASS, fmt.Sprintf("service!~/%s/", r.servicePattern)
	}
	if !r.orgUnitRgx.MatchString(ou.Name) {
		return PASS, fmt.Sprintf("orgUnit!~/%s/", r.orgUnitPattern)
	}
	return r.action, fmt.Sprintf(
		"service=~/%s/ and orgUnit=~/%s/",
		r.servicePattern, r.orgUnitPattern,
	)
}
