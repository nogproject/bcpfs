// vim: sw=8

// Package `bcpcfg` contains a reader for a config file that describes the
// filesystem setup for a service facility tree and a research unit tree.
//
// The file format uses HCL <https://github.com/hashicorp/hcl>.
//
// `cmd/bcpfs-perms/generic-example-bcpfs.hcl` contains an example file with
// the available settings documented.
//
// Use `Load()` to load a config file.  Then use package `grp` to parse the
// available Unix groups and package `bcp` to combine the groups and the config
// and obtain a struct that represents the organization.
package bcpcfg

// For background info on parsing HCL, see:
//
// The Vault config code
// <https://github.com/hashicorp/vault/blob/master/command/server/config.go>,
// which gives a general idea how to parse HCL lists.
//
// James Nugent's Using HCL, which contains more details:
// <http://jen20.com/2015/09/07/using-hcl-part-1.html>,
// <http://jen20.com/2015/09/08/using-hcl-part-2.html>.

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/hcl"
	"github.com/hashicorp/hcl/hcl/ast"
)

// `Root` contains the filesystem config.  See example
// `generic-example-bcpfs.hcl` for a description of the available settings.
//
// The `hcl:"<key>"` fields are handled by the HCL decoder.  The ignored
// `hcl:"-"` fields are explicitly decoded in `Parse()`.
type Root struct {
	Rootdir        string       `hcl:"rootdir"`
	ServiceDir     string       `hcl:"serviceDir"`
	OrgUnitDir     string       `hcl:"orgUnitDir"`
	SuperGroup     string       `hcl:"superGroup"`
	OrgUnitPrefix  string       `hcl:"orgUnitPrefix"`
	ServicePrefix  string       `hcl:"servicePrefix"`
	OpsSuffix      string       `hcl:"opsSuffix"`
	FacilitySuffix string       `hcl:"facilitySuffix"`
	Facilities     []Facility   `hcl:"-"`
	OrgUnits       []OrgUnit    `hcl:"-"`
	Filter         []FilterRule `hcl:"-"`
	Symlinks       []Symlink    `hcl:"-"`
	Sharing        *Sharing     `hcl:"-" yaml:",omitempty"`
}

type Facility struct {
	Name     string   `hcl:"name"`
	Services []string `hcl:"services"`
	Access   string   `hcl:"access"`
}

type OrgUnit struct {
	Name    string          `hcl:"name"`
	Subdirs []DirWithPolicy `hcl:"subdirs"`
	// `ExtraDirs` is kept for compatibility; prefer `Subdirs`; see NOE-11.
	ExtraDirs []string `hcl:"extraDirs"`
}

type DirWithPolicy struct {
	Name   string `hcl:"name"`
	Policy string `hcl:"policy"`
}

type Sharing struct {
	NamingPolicies []SharingNamingPolicy `hcl:"-" yaml:"namingPolicies"`
	Exports        []SharingExport       `hcl:"-" yaml:"exports"`
	Imports        []SharingImport       `hcl:"-" yaml:"imports"`
}

type SharingNamingPolicy struct {
	Action string `hcl:"action" yaml:"action"`
	Match  string `hcl:"match" yaml:"match"`
}

type SharingExport struct {
	Path string   `hcl:"path" yaml:"path"`
	Acl  []string `hcl:"acl" yaml:"acl"`
}

type SharingImport struct {
	Action string `hcl:"action" yaml:"action"`
	Group  string `hcl:"group" yaml:"group"`
	Match  string `hcl:"match" yaml:"match"`
}

func isValidDirPolicy(p string) bool {
	switch p {
	case "owner", "group", "manager":
		return true
	default:
		return false
	}
}

type FilterRuleCfg struct {
	Service  string   `hcl:"service"`
	Services []string `hcl:"services"`
	OrgUnit  string   `hcl:"orgUnit"`
	OrgUnits []string `hcl:"orgUnits"`
	Action   string   `hcl:"action"`
}

type FilterRule struct {
	Services []string
	OrgUnits []string
	Action   string
}

type Symlink struct {
	Path   string `hcl:"path"`
	Target string `hcl:"target"`
}

// `Load()` loads a config from `path`.
func Load(path string) (*Root, error) {
	d, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(string(d))
}

// `Parse()` parses the config from a string.  External users usually should
// use `Load()`.
func Parse(d string) (*Root, error) {
	root, err := hcl.Parse(d)
	if err != nil {
		return nil, err
	}

	var cfg Root
	if err := hcl.DecodeObject(&cfg, root); err != nil {
		return nil, err
	}

	list, ok := root.Node.(*ast.ObjectList)
	if !ok {
		return nil, errors.New("Missing config root object")
	}

	if fs := list.Filter("facility"); len(fs.Items) > 0 {
		if err := parseFacilityConfigs(&cfg, fs); err != nil {
			return nil, fmt.Errorf(
				"Failed to parse 'facilities': %s", err,
			)
		}
	}

	if ous := list.Filter("orgUnit"); len(ous.Items) > 0 {
		if err := parseOrgUnitConfigs(&cfg, ous); err != nil {
			return nil, fmt.Errorf(
				"Failed to parse 'orgUnits': %s", err,
			)
		}
	}

	if fs := list.Filter("filter"); len(fs.Items) > 0 {
		if err := parseFilter(&cfg, fs); err != nil {
			return nil, fmt.Errorf(
				"Failed to parse 'filter': %s", err,
			)
		}
	}

	if links := list.Filter("symlink"); len(links.Items) > 0 {
		if err := parseSymlinks(&cfg, links); err != nil {
			return nil, fmt.Errorf(
				"Failed to parse 'symlink': %s", err,
			)
		}
	}

	if s := list.Filter("sharing"); len(s.Items) == 1 {
		var sharing Sharing
		if err := parseSharing(&sharing, s.Items[0].Val); err != nil {
			return nil, fmt.Errorf(
				"Failed to parse 'sharing': %s", err,
			)
		}
		cfg.Sharing = &sharing
	} else if len(s.Items) > 1 {
		return nil, errors.New("More than one 'sharing' block.")
	}

	if cfg.Rootdir == "" {
		return nil, errors.New("Missing `rootdir`")
	}
	if !filepath.IsAbs(cfg.Rootdir) {
		return nil, errors.New("`rootdir` must be absolute")
	}

	return &cfg, nil
}

func parseFacilityConfigs(cfg *Root, list *ast.ObjectList) error {
	fs := make([]Facility, len(list.Items))
	for i, e := range list.Items {
		if err := hcl.DecodeObject(&fs[i], e.Val); err != nil {
			return fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}

		if fs[i].Access != "perService" &&
			fs[i].Access != "allOrgUnits" &&
			fs[i].Access != "" {
			return fmt.Errorf(
				"invalid Access `%s` in facility `%s`.",
				fs[i].Access, fs[i].Name,
			)
		}
	}
	cfg.Facilities = fs
	return nil
}

func parseOrgUnitConfigs(cfg *Root, list *ast.ObjectList) error {
	ous := make([]OrgUnit, len(list.Items))
	for i, e := range list.Items {
		var ou OrgUnit
		if err := hcl.DecodeObject(&ou, e.Val); err != nil {
			return fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}

		err := validateSubdirs(ou.Subdirs)
		if err != nil {
			return fmt.Errorf(
				"invalid dirs in item %d: %s", i, err,
			)
		}

		ous[i] = ou
	}
	cfg.OrgUnits = ous
	return nil
}

func validateSubdirs(dirs []DirWithPolicy) error {
	for i, d := range dirs {
		if !isValidDirPolicy(d.Policy) {
			return fmt.Errorf("invalid policy in item %d", i)
		}
	}
	return nil
}

func parseFilter(cfg *Root, list *ast.ObjectList) error {
	var filterRules []FilterRule
	for i, e := range list.Items {
		var ruleCfg FilterRuleCfg
		if err := hcl.DecodeObject(&ruleCfg, e.Val); err != nil {
			return fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}

		if rule, err := ValidateFilterRule(ruleCfg); err != nil {
			return fmt.Errorf("item %d: %s", i, err)
		} else {
			filterRules = append(filterRules, rule)
		}
	}
	cfg.Filter = filterRules
	return nil
}

func ValidateFilterRule(r FilterRuleCfg) (rule FilterRule, err error) {

	if (r.Action != "accept") && (r.Action != "reject") {
		return rule, fmt.Errorf("Invalid action!")
	}
	rule.Action = r.Action

	if (r.Service != "") && (len(r.Services) > 0) {
		return rule, fmt.Errorf("Use either `service` or `services`!")
	}

	if r.Service == "" {
		if len(r.Services) == 0 {
			return rule, fmt.Errorf("No service defined!")
		}
		rule.Services = r.Services
	} else {
		rule.Services = append(rule.Services, r.Service)
	}

	if r.OrgUnit != "" && len(r.OrgUnits) > 0 {
		return rule, fmt.Errorf("Use either `orgUnit` or `orgUnits`!")
	}

	if r.OrgUnit == "" {
		if len(r.OrgUnits) == 0 {
			return rule, fmt.Errorf("No orgUnit defined!")
		}
		rule.OrgUnits = r.OrgUnits
	} else {
		rule.OrgUnits = append(rule.OrgUnits, r.OrgUnit)
	}

	return rule, nil
}

func parseSymlinks(cfg *Root, list *ast.ObjectList) error {
	var links []Symlink
	for i, e := range list.Items {
		var link Symlink
		if err := hcl.DecodeObject(&link, e.Val); err != nil {
			return fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}
		if link.Path == "" {
			return fmt.Errorf("empty `path` in item %d", i)
		}
		if link.Target == "" {
			return fmt.Errorf("empty `target` in item %d", i)
		}

		links = append(links, link)
	}
	cfg.Symlinks = links
	return nil
}

func parseSharing(cfg *Sharing, node ast.Node) error {
	obj, ok := node.(*ast.ObjectType)
	if !ok {
		return errors.New("Invalid 'sharing' block.")
	}
	fields := obj.List

	if fs := fields.Filter("namingPolicy"); len(fs.Items) > 0 {
		pol, err := parseSharingNamingPolicies(fs)
		if err != nil {
			return fmt.Errorf(
				"Failed to parse 'sharing.namingPolicy': %s",
				err,
			)
		}
		cfg.NamingPolicies = pol
	}

	if fs := fields.Filter("export"); len(fs.Items) > 0 {
		exps, err := parseSharingExports(fs)
		if err != nil {
			return fmt.Errorf(
				"Failed to parse 'sharing.export': %s", err,
			)
		}
		cfg.Exports = exps
	}

	if fs := fields.Filter("import"); len(fs.Items) > 0 {
		imps, err := parseSharingImports(fs)
		if err != nil {
			return fmt.Errorf(
				"Failed to parse 'sharing.imports': %s", err,
			)
		}
		cfg.Imports = imps
	}

	return nil
}

func parseSharingNamingPolicies(list *ast.ObjectList) (
	[]SharingNamingPolicy, error,
) {
	pols := make([]SharingNamingPolicy, 0, len(list.Items))
	for i, e := range list.Items {
		var pol SharingNamingPolicy
		if err := hcl.DecodeObject(&pol, e.Val); err != nil {
			return nil, fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}

		if !isValidNamingPolicyAction(pol.Action) {
			return nil, fmt.Errorf(
				"failed to parse item %d: "+
					"invalid naming policy action `%s`",
				i, pol.Action,
			)
		}

		pols = append(pols, pol)
	}
	return pols, nil
}

func isValidNamingPolicyAction(a string) bool {
	return a == "allow" || a == "deny"
}

var rgxSharingAce = regexp.MustCompile(`^group:[a-z0-9-]+:[r-][w-][x-]$`)

func parseSharingExports(list *ast.ObjectList) ([]SharingExport, error) {
	exps := make([]SharingExport, 0, len(list.Items))
	for i, e := range list.Items {
		var exp SharingExport
		if err := hcl.DecodeObject(&exp, e.Val); err != nil {
			return nil, fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}

		for j, ace := range exp.Acl {
			if !rgxSharingAce.MatchString(ace) {
				return nil, fmt.Errorf(
					"failed to parse item %d: "+
						"malformed ACL entry %d",
					i, j,
				)
			}
		}

		exps = append(exps, exp)
	}
	return exps, nil
}

func parseSharingImports(list *ast.ObjectList) ([]SharingImport, error) {
	imps := make([]SharingImport, 0, len(list.Items))
	for i, e := range list.Items {
		var imp SharingImport
		if err := hcl.DecodeObject(&imp, e.Val); err != nil {
			return nil, fmt.Errorf(
				"failed to parse item %d: %s", i, err,
			)
		}

		if !isValidImportAction(imp.Action) {
			return nil, fmt.Errorf(
				"failed to parse item %d: "+
					"invalid Import action `%s`",
				i, imp.Action,
			)
		}

		imps = append(imps, imp)
	}
	return imps, nil
}

func isValidImportAction(a string) bool {
	return a == "accept" || a == "reject"
}
