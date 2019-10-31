// Package `bcpsharing` implements logical NOE-9 BCPFS sharing.  The main
// function is `bcpsharing.Compile()`, which compiles a configuration to a
// sharing specification, which can then be applied with package
// `bcpsharingapply`.
package bcpsharing

import (
	"errors"
	"fmt"
	slashpath "path"
	"regexp"
	"sort"
	"strings"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
)

// `Bcpfs` is a helper type to map logical paths and group names to real
// filesystem paths and group names.
type Bcpfs struct {
	Rootdir            string
	ServiceDir         string
	OrgUnitDir         string
	ServiceGroupPrefix string
	OrgUnitGroupPrefix string
	facilityByService  map[string]string
	facilityDirs       map[string]struct{}
}

// `Sharing` contains the logical sharing specification.  It is returned by
// `Compile()`.
type Sharing struct {
	Bcpfs *Bcpfs

	// `Shares` contains active exports, that is exports that were also
	// imported.
	Shares Exports

	// `RealShares` contains ACLs for realpaths relative to the root dir.
	RealShares RealExports

	// `Traversal` contains `--x` ACLs to traverse toplevel dirs, so that
	// symlinks from the shared trees can be resolved.  `Traversal` uses
	// realpaths relative to the root dir.
	Traversal RealExports

	// `ShareTrees` contains the specification of `<org>/<ou>/shared`
	// trees.  It uses realpaths relative to the root dir.
	ShareTrees ShareTrees
}

// `NamingPolicy` is a compiled version of the config list
// `sharing.namingPolicy`.
type NamingPolicy []NamingRule

type NamingAction string

const (
	NamingActionAllow = "allow"
	NamingActionDeny  = "deny"
	NamingActionPass  = "pass"
)

type NamingRule struct {
	Action NamingAction
	Match  *regexp.Regexp
}

// `ImportFilter` is a compiled version of the config list `sharing.imports`.
type ImportFilter []ImportRule

type ImportAction string

const (
	ImportActionAccept = "accept"
	ImportActionReject = "reject"
	ImportActionPass   = "pass"
)

type ImportRule struct {
	Group  string
	Match  *regexp.Regexp
	Action ImportAction
}

// `Exports` is a compiled version of the config list `sharing.exports`.  It
// uses logical paths.
type Exports []ExportEntry

// `RealExports` specifies ACLs on realpaths.
type RealExports []ExportEntry

type ExportEntry struct {
	Path           string
	Acl            Acl
	ManagingGroups Groups
}

type Acl []Ace

// `AceMode` is a ACL entry mode in chmod `rwx` format.
type AceMode string

type Ace struct {
	Group string
	Mode  AceMode
}

type GroupType int

const (
	GroupTypeUnspecified = iota
	GroupTypeOu
	GroupTypeOps
)

// `Group` is an organizational unit group or an ops group.
type Group struct {
	GroupType GroupType
	Name      string
}

type Groups []Group

// `ShareTrees` contains the specification of `<org>/<ou>/shared` trees.
type ShareTrees []ShareTree

type ShareTree struct {
	OrgUnit string
	Files   []ShareFile
}

// `ShareFile` is a file in a shared tree, where `Path` is a realpath relative
// to the root dir and:
//
//  - Target == "" indicates a directory and
//  - Target != "" indicates a symlink.
//
type ShareFile struct {
	Path   string
	Target string
}

func Compile(cfg *bcpcfg.Root) (*Sharing, error) {
	fs := NewBcpfs(cfg)

	imps, err := compileImports(cfg.Sharing.Imports)
	if err != nil {
		return nil, err
	}

	exps, err := compileExports(fs, cfg.Sharing.Exports)
	if err != nil {
		return nil, err
	}

	pol, err := compileNamingPolicy(cfg.Sharing.NamingPolicies)
	if err != nil {
		return nil, err
	}

	if err := checkExportPaths(exps, pol); err != nil {
		return nil, err
	}
	if err := checkExportAclScopes(fs, exps); err != nil {
		return nil, err
	}
	if err := checkNoNesting(exps); err != nil {
		return nil, err
	}
	if err := checkNoRealpathNesting(fs, exps); err != nil {
		return nil, err
	}

	shares := selectImportedExports(exps, imps)
	realShares := compileRealShares(fs, shares)
	traversal := compileTraversal(fs, shares, realShares)
	shareTrees := compileShareTrees(fs, shares)

	return &Sharing{
		Bcpfs:      fs,
		Shares:     shares,
		RealShares: realShares,
		Traversal:  traversal,
		ShareTrees: shareTrees,
	}, nil
}

type NodeType int

const (
	UnspecifiedNode NodeType = iota
	InnerNode
	LeafNode
)

func checkNoNesting(exps Exports) error {
	tree := make(map[string]NodeType)

	checkPath := func(p string) error {
		switch tree[p] {
		case LeafNode:
			return fmt.Errorf("duplicate export `%s`", p)
		case InnerNode:
			return fmt.Errorf("nested export prefix `%s`", p)
		}
		tree[p] = LeafNode

		parts := strings.Split(p, "/")
		for i := len(parts) - 1; i > 0; i-- {
			prefix := slashpath.Join(parts[0:i]...)
			switch tree[prefix] {
			case LeafNode:
				return fmt.Errorf(
					"nested export suffix `%s`", p,
				)
			case InnerNode:
				return nil
			}
			tree[prefix] = InnerNode
		}

		return nil
	}

	for _, exp := range exps {
		if err := checkPath(exp.Path); err != nil {
			return err
		}
	}

	return nil
}

func checkNoRealpathNesting(fs *Bcpfs, exps Exports) error {
	tree := make(map[string]NodeType)

	checkPath := func(p string) error {
		switch tree[p] {
		case LeafNode:
			return nil // Duplicate realpath is allowed.
		case InnerNode:
			return fmt.Errorf("nested realpath prefix `%s`", p)
		}
		tree[p] = LeafNode

		parts := strings.Split(p, "/")
		for i := len(parts) - 1; i > 0; i-- {
			prefix := slashpath.Join(parts[0:i]...)
			switch tree[prefix] {
			case LeafNode:
				return fmt.Errorf(
					"nested realpath suffix `%s`", p,
				)
			case InnerNode:
				return nil
			}
			tree[prefix] = InnerNode
		}

		return nil
	}

	for _, exp := range exps {
		if err := checkPath(fs.Realpath(exp.Path)); err != nil {
			return err
		}
	}

	return nil
}

func NewBcpfs(cfg *bcpcfg.Root) *Bcpfs {
	fs := &Bcpfs{
		Rootdir:            cfg.Rootdir,
		ServiceDir:         cfg.ServiceDir,
		OrgUnitDir:         cfg.OrgUnitDir,
		ServiceGroupPrefix: cfg.ServicePrefix,
		OrgUnitGroupPrefix: cfg.OrgUnitPrefix,
		facilityByService:  make(map[string]string),
		facilityDirs:       make(map[string]struct{}),
	}

	for _, fac := range cfg.Facilities {
		for _, srv := range fac.Services {
			fs.facilityByService[srv] = fac.Name
		}
		fs.facilityDirs[fac.Name+"-facility"] = struct{}{}
	}

	return fs
}

// `FsGroupOrgUnit(ou)` returns the filesystem group for org unit `ou`.
func (fs *Bcpfs) FsGroupOrgUnit(ou string) string {
	return fmt.Sprintf("%s_%s", fs.OrgUnitGroupPrefix, ou)
}

// `FsGroups(groups)` returns filesystem group names for `groups`.
func (fs *Bcpfs) FsGroups(gs Groups) []string {
	fgs := make([]string, 0, len(gs))
	for _, g := range gs {
		switch g.GroupType {
		case GroupTypeOu:
			fgs = append(fgs, fmt.Sprintf(
				"%s_%s",
				fs.OrgUnitGroupPrefix, g.Name,
			))
		case GroupTypeOps:
			fgs = append(fgs, fmt.Sprintf(
				"%s_%s-ops",
				fs.ServiceGroupPrefix, g.Name,
			))
		default:
			panic("invalid group type")
		}
	}
	return fgs
}

var ErrNotServicePath = errors.New("not a service path")

func (fs *Bcpfs) IsServicePath(p string) bool {
	_, err := fs.facilityOfServicePath(p)
	return err == nil
}

func (fs *Bcpfs) FacilityGroupOfServicePath(p string) (string, error) {
	fac, err := fs.facilityOfServicePath(p)
	if err != nil {
		return "", err
	}
	return fac + "-facility", nil
}

func (fs *Bcpfs) facilityOfServicePath(p string) (string, error) {
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		return "", ErrNotServicePath
	}
	maybeSrv := parts[1]
	fac, ok := fs.facilityByService[maybeSrv]
	if !ok {
		return "", ErrNotServicePath
	}
	return fac, nil
}

func (fs *Bcpfs) IsServiceRealpath(p string) bool {
	base := strings.Split(p, "/")[0]
	return base == fs.ServiceDir
}

func (fs *Bcpfs) IsFacilityPath(p string) bool {
	parts := strings.Split(p, "/")
	if len(parts) < 1 {
		return false
	}
	maybeFac := parts[0]
	_, ok := fs.facilityDirs[maybeFac]
	return ok
}

func (fs *Bcpfs) ManagingGroupsOfPath(p string) (Groups, error) {
	parts := strings.Split(p, "/")

	// Only `ou` manages non-service paths.
	if !fs.IsServicePath(p) {
		ou := parts[0]
		return []Group{
			{GroupType: GroupTypeOu, Name: ou},
		}, nil
	}

	// For service paths, distinguish facility and ordinary ou.
	var ou string
	if fs.IsFacilityPath(p) {
		// `<fac>/<srv>/<ou>`; <fac> and <ou> may be equal.
		if len(parts) < 3 {
			return nil, errors.New("service path too short")
		}
		ou = parts[2]
	} else {
		// `<ou>/<srv>`.
		if len(parts) < 2 {
			return nil, errors.New("service path too short")
		}
		ou = parts[0]
	}
	srv := parts[1]
	fac := fs.facilityByService[srv]

	return []Group{
		{GroupType: GroupTypeOu, Name: ou},
		{GroupType: GroupTypeOps, Name: fac},
	}, nil
}

// `Realpath()` returns a realpath relative to the root dir.
func (fs *Bcpfs) Realpath(p string) string {
	if !fs.IsServicePath(p) {
		return slashpath.Join(
			fs.OrgUnitDir, p,
		)
	}

	// Path `p` must be a service path.
	parts := strings.Split(p, "/")
	if len(parts) < 2 {
		// Path too short.  Return an empty string to indicate the
		// problem instead of full error handling, because paths must
		// not be too short for a valid configuration.
		return ""
	}

	if fs.IsFacilityPath(p) {
		return slashpath.Join(append(
			[]string{fs.ServiceDir}, parts[1:]...,
		)...)
	}

	ou := parts[0]
	srv := parts[1]
	rest := parts[2:]
	return slashpath.Join(append(
		[]string{fs.ServiceDir, srv, ou}, rest...,
	)...)
}

func compileImports(cfg []bcpcfg.SharingImport) (ImportFilter, error) {
	imps := make([]ImportRule, 0, len(cfg))

	for i, c := range cfg {
		match, err := regexp.Compile("^" + c.Match + "$")
		if err != nil {
			err := fmt.Errorf(
				"failed to compile match of item %d: %v",
				i, err,
			)
			return nil, err
		}

		imps = append(imps, ImportRule{
			Group:  c.Group,
			Match:  match,
			Action: ImportAction(c.Action),
		})
	}

	return imps, nil
}

func compileExports(fs *Bcpfs, cfg []bcpcfg.SharingExport) (Exports, error) {
	exps := make([]ExportEntry, 0, len(cfg))

	for i, c := range cfg {
		acl, err := parseAcl(c.Acl)
		if err != nil {
			err := fmt.Errorf(
				"failed to parse ACL of item %d: %v",
				i, err,
			)
			return nil, err
		}

		mgroups, err := fs.ManagingGroupsOfPath(c.Path)
		if err != nil {
			err := fmt.Errorf(
				"failed to get group for path of item %d: %v",
				i, err,
			)
			return nil, err
		}

		exps = append(exps, ExportEntry{
			Path:           strings.Trim(c.Path, "/"),
			Acl:            acl,
			ManagingGroups: mgroups,
		})
	}

	return exps, nil
}

func compileNamingPolicy(
	cfg []bcpcfg.SharingNamingPolicy,
) (NamingPolicy, error) {
	pol := make([]NamingRule, 0, len(cfg))
	for i, c := range cfg {
		match, err := regexp.Compile("^" + c.Match + "$")
		if err != nil {
			err := fmt.Errorf(
				"failed to compile match of item %d: %v",
				i, err,
			)
			return nil, err
		}

		pol = append(pol, NamingRule{
			Match:  match,
			Action: NamingAction(c.Action),
		})

	}
	return pol, nil
}

// `checkExportPaths()` verifies that the exported paths are allowed by the
// naming policy.
func checkExportPaths(exps Exports, pol NamingPolicy) error {
	for _, exp := range exps {
		if err := checkExportPath(exp, pol); err != nil {
			return err
		}
	}
	return nil
}

func checkExportPath(exp ExportEntry, pol NamingPolicy) error {
	for _, rule := range pol {
		switch rule.Apply(exp) {
		case NamingActionAllow:
			return nil
		case NamingActionDeny:
			return fmt.Errorf(
				"naming policy: rule denied export path `%s`",
				exp.Path,
			)
		case NamingActionPass:
			// continue loop.
		default:
			panic("logic error")
		}
	}
	return fmt.Errorf(
		"naming policy: default deny export path `%s`", exp.Path,
	)
}

func (rule NamingRule) Apply(exp ExportEntry) NamingAction {
	if rule.Match.MatchString(exp.Path) {
		return rule.Action
	}
	return NamingActionPass
}

// `checkExportAclScopes()` verifies that the exports do not contain ACLs that
// are managed by `bcpfs-perms`, which manages ou and facility ACLs.  Export
// ACLs must grant permissions only to other groups, so that they do not
// interfere with `bcpfs-perms`.
func checkExportAclScopes(fs *Bcpfs, exps Exports) error {
	for _, exp := range exps {
		if err := checkExportEntryAclScopes(fs, exp); err != nil {
			return err
		}
	}
	return nil
}

func checkExportEntryAclScopes(fs *Bcpfs, exp ExportEntry) error {
	groups := make(map[string]struct{})
	for _, ace := range exp.Acl {
		groups[ace.Group] = struct{}{}
	}

	ou := strings.Split(exp.Path, "/")[0]
	if _, ok := groups[ou]; ok {
		return fmt.Errorf("self export `%s`", exp.Path)
	}

	// err == nil -> It is a service path.
	if fac, err := fs.FacilityGroupOfServicePath(exp.Path); err == nil {
		if _, ok := groups[fac]; ok {
			return fmt.Errorf(
				"export to owning facility `%s`", exp.Path,
			)
		}
	}

	return nil
}

var rgxAce = regexp.MustCompile(`^group:[a-z0-9-]+:[r-][w-][x-]$`)

func parseAcl(cfg []string) (Acl, error) {
	acl := make([]Ace, 0, len(cfg))

	for i, c := range cfg {
		if !rgxAce.MatchString(c) {
			err := fmt.Errorf("malformed ACL item %d", i)
			return nil, err
		}
		toks := strings.Split(c, ":")

		acl = append(acl, Ace{
			Group: toks[1],
			Mode:  AceMode(toks[2]),
		})
	}

	return acl, nil
}

// `selectImportedExports()` returns exports that are also imported.
func selectImportedExports(exps Exports, imps ImportFilter) Exports {
	sels := make([]ExportEntry, 0, len(exps))
	for _, exp := range exps {
		// An empty ACL is a special case that indicates that the path
		// should be unexported even if it is not selected by an import
		// filter.
		if len(exp.Acl) == 0 {
			sels = append(sels, ExportEntry{
				Path:           exp.Path,
				ManagingGroups: exp.ManagingGroups,
			})
			continue
		}

		acl := make([]Ace, 0, len(exp.Acl))
		for _, ace := range exp.Acl {
			switch imps.FilterPathAce(exp.Path, ace) {
			case ImportActionAccept:
				acl = append(acl, ace)
			case ImportActionReject:
				// continue loop
			default:
				panic("logic error")
			}
		}
		if len(acl) > 0 {
			sels = append(sels, ExportEntry{
				Path:           exp.Path,
				Acl:            acl,
				ManagingGroups: exp.ManagingGroups,
			})
		}
	}
	return sels
}

func (imps ImportFilter) FilterPathAce(path string, ace Ace) ImportAction {
	for _, imp := range imps {
		switch imp.FilterPathAce(path, ace) {
		case ImportActionAccept:
			return ImportActionAccept
		case ImportActionReject:
			return ImportActionReject
		case ImportActionPass:
			// continue loop
		default:
			panic("logic error")
		}
	}
	return ImportActionReject
}

func (imp ImportRule) FilterPathAce(path string, ace Ace) ImportAction {
	if ace.Group != imp.Group {
		return ImportActionPass
	}
	if !imp.Match.MatchString(path) {
		return ImportActionPass
	}
	return imp.Action
}

// `compileRealShares()` maps `shares` to realpaths.  Multiple logical paths
// may map to the same realpath.  If so, the realpath ACL is the union of the
// logical path ACLs.
func compileRealShares(fs *Bcpfs, shares Exports) RealExports {
	reals := make([]ExportEntry, 0, len(shares))
	realsByRealpath := make(map[string]int)
	for _, shr := range shares {
		rp := fs.Realpath(shr.Path)
		if idx, ok := realsByRealpath[rp]; ok {
			// Update exiting.
			reals[idx].Acl = reals[idx].Acl.Union(shr.Acl)
		} else {
			// Append new.
			realsByRealpath[rp] = len(reals)
			reals = append(reals, ExportEntry{
				Path:           rp,
				Acl:            shr.Acl,
				ManagingGroups: shr.ManagingGroups,
			})
		}
	}
	return reals
}

// `compileTraversal()` computes `--x` ACLs to allow traversing toplevel
// directories in order to resolve symlinks from the shared trees.
func compileTraversal(
	fs *Bcpfs,
	shares Exports,
	reals RealExports,
) RealExports {
	travs := make([]ExportEntry, 0, len(reals)*5)
	travsByRealpath := make(map[string]int)

	// Allow directory traversal along realpaths.
	for _, r := range reals {
		acl := NewTraversalAclWithGroups(r.Acl.Groups())
		parts := strings.Split(r.Path, "/")

		// Do not add `--x` to `<srvdir>/<srv>` but only to subdirs, so
		// that srv group membership is required to access realpath.
		begin := 2
		if fs.IsServiceRealpath(r.Path) {
			begin = 3
		}
		for i := begin; i < len(parts); i++ {
			path := slashpath.Join(parts[:i]...)
			if idx, ok := travsByRealpath[path]; ok {
				// Update existing.
				travs[idx].Acl = travs[idx].Acl.Union(acl)
			} else {
				// Append new.
				travsByRealpath[path] = len(travs)
				travs = append(travs, ExportEntry{
					Path:           path,
					Acl:            acl,
					ManagingGroups: r.ManagingGroups,
				})
			}
		}
	}

	// Allow traversal of ou toplevel directories to reach symlinks.
	for _, shr := range shares {
		acl := NewTraversalAclWithGroups(shr.Acl.Groups())
		ou := strings.Split(shr.Path, "/")[0]
		path := slashpath.Join(fs.OrgUnitDir, ou)
		if idx, ok := travsByRealpath[path]; ok {
			// Update existing.
			travs[idx].Acl = travs[idx].Acl.Union(acl)
		} else {
			// Append new.
			travsByRealpath[path] = len(travs)
			travs = append(travs, ExportEntry{
				Path: path,
				Acl:  acl,
			})
		}
	}

	return travs
}

func (f ShareFile) IsDir() bool {
	return f.Target == ""
}

func (f ShareFile) IsSymlink() bool {
	return f.Target != ""
}

func compileShareTrees(fs *Bcpfs, shares Exports) ShareTrees {
	// `treesM` maps ou => path => target; empty target indicates dir.
	treesM := make(map[string]map[string]string)

	getTree := func(ou string) map[string]string {
		tr, ok := treesM[ou]
		if !ok {
			tr = make(map[string]string)
			treesM[ou] = tr
		}
		return tr
	}

	realpath := func(ou string, parts []string) string {
		return slashpath.Join(append(
			[]string{fs.OrgUnitDir, ou, "shared"}, parts...,
		)...)
	}

	addDir := func(ou string, parts []string) {
		tr := getTree(ou)
		tr[realpath(ou, parts)] = ""
	}

	addLink := func(ou string, parts []string) {
		upLevels := len(parts) + 1
		targetParts := make([]string, upLevels, upLevels+len(parts))
		for i := 0; i < upLevels; i++ {
			targetParts[i] = ".."
		}
		targetParts = append(targetParts, parts...)
		target := slashpath.Join(targetParts...)

		tr := getTree(ou)
		tr[realpath(ou, parts)] = target
	}

	addShare := func(ou, path string) {
		parts := strings.Split(path, "/")
		for i := 1; i < len(parts); i++ {
			addDir(ou, parts[:i])
		}
		addLink(ou, parts)
	}

	// Shares to self.
	for _, shr := range shares {
		ou := strings.Split(shr.Path, "/")[0]
		addShare(ou, shr.Path)
	}

	// Shares from others.
	for _, shr := range shares {
		for _, ou := range shr.Acl.Groups() {
			addShare(ou, shr.Path)
		}
	}

	trees := make([]ShareTree, 0, len(treesM))
	for ou, tr := range treesM {
		files := make([]ShareFile, 0, len(tr))
		for path, target := range tr {
			files = append(files, ShareFile{
				Path:   path,
				Target: target,
			})
		}
		sort.Slice(files, func(i, j int) bool {
			return files[i].Path < files[j].Path
		})

		trees = append(trees, ShareTree{
			OrgUnit: ou,
			Files:   files,
		})
	}
	sort.Slice(trees, func(i, j int) bool {
		return trees[i].OrgUnit < trees[j].OrgUnit
	})
	return trees
}

func NewTraversalAclWithGroups(gs []string) Acl {
	acl := make([]Ace, 0, len(gs))
	for _, g := range gs {
		acl = append(acl, Ace{
			Group: g,
			Mode:  AceMode("--x"),
		})
	}
	return acl
}

func (acl Acl) Groups() []string {
	gs := make([]string, 0, len(acl))
	for _, ace := range acl {
		gs = append(gs, ace.Group)
	}
	return gs
}

func (a Acl) Union(b Acl) Acl {
	u := make([]Ace, len(a), len(a)+len(b))
	uByGroup := make(map[string]int)

	// Copy a to union.
	copy(u, a)
	for i, ace := range u {
		uByGroup[ace.Group] = i
	}

	// Merge b.
	for _, ace := range b {
		if idx, ok := uByGroup[ace.Group]; ok {
			// Update existing Ace for group.
			u[idx].Mode = u[idx].UnionMode(ace)
		} else {
			// Append new Ace.
			uByGroup[ace.Group] = len(u)
			u = append(u, ace)
		}
	}

	return u
}

func (a Ace) UnionMode(b Ace) AceMode {
	mode := []byte{'r', 'w', 'x'}
	for i, code := range mode {
		if a.Mode[i] == code || b.Mode[i] == code {
			// keep
		} else {
			mode[i] = '-'
		}
	}
	return AceMode(string(mode))
}

func (exps RealExports) Paths() []string {
	ps := make([]string, 0, len(exps))
	for _, exp := range exps {
		ps = append(ps, exp.Path)
	}
	return ps
}

func (m AceMode) WithoutX() AceMode {
	return m[0:2] + "-"
}
