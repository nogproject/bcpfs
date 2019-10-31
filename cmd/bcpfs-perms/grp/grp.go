// vim: sw=8

// Package `grp` provides access to Unix groups.
package grp

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/nogproject/bcpfs/pkg/execx"
)

var getent = execx.MustLookTool(execx.ToolSpec{
	Program:   "getent",
	CheckArgs: []string{"--version"},
	CheckText: "getent",
})

type Group struct {
	Name string
	Gid  int
}

// `Groups()` returns a list of Unix groups as reported by `getent`.  The list
// may contain duplicates, even conflicting ones.
func Groups() ([]Group, error) {
	dat, err := exec.Command(getent.Path, "group").Output()
	if err != nil {
		return nil, fmt.Errorf("Failed to execute `getent`: %v", err)
	}
	txt := strings.TrimSpace(string(dat))

	gs := make([]Group, 0)
	for _, line := range strings.Split(txt, "\n") {
		fs := strings.Split(line, ":")
		if len(fs) != 4 {
			return nil, fmt.Errorf(
				"Invalid getent output `%s`", line,
			)
		}
		gid, err := strconv.Atoi(fs[2])
		if err != nil {
			return nil, fmt.Errorf("Invalid gid `%s`", fs[2])
		}
		gs = append(gs, Group{Name: fs[0], Gid: gid})
	}

	return gs, nil
}

// `selectGroups()` selects `groups` whose names begin with any of the
// `prefixes` or match one of the names in `equals`.
func SelectGroups(groups []Group, prefixes []string, equals []string) []Group {
	isEqual := func(s string) bool {
		for _, e := range equals {
			if e == s {
				return true
			}
		}
		return false
	}
	hasPrefix := func(n string) bool {
		for _, p := range prefixes {
			if strings.HasPrefix(n, p) {
				return true
			}
		}
		return false
	}

	res := make([]Group, 0)
	for _, g := range groups {
		if isEqual(g.Name) || hasPrefix(g.Name) {
			res = append(res, g)
		}
	}
	return res
}

// `DedupGroups()` returns a list without duplicate groups.  It returns an
// error if the input contains conflicting duplicates.
func DedupGroups(groups []Group) ([]Group, error) {
	byName := make(map[string]Group)
	byGid := make(map[int]Group)

	errConflict := func(a, b Group) error {
		return fmt.Errorf(
			"conflicting groups %d(%s) and %d(%s)",
			a.Gid, a.Name,
			b.Gid, b.Name,
		)
	}

	isDuplicate := func(g Group) (bool, error) {
		if seen, ok := byName[g.Name]; ok {
			if g != seen {
				return true, errConflict(seen, g)
			}
			return true, nil
		}
		if seen, ok := byGid[g.Gid]; ok {
			if g != seen {
				return true, errConflict(seen, g)
			}
			return true, nil
		}
		return false, nil
	}

	res := make([]Group, 0)
	for _, g := range groups {
		isDup, err := isDuplicate(g)
		if err != nil {
			return nil, err
		}
		if isDup {
			continue
		}
		byName[g.Name] = g
		byGid[g.Gid] = g
		res = append(res, g)
	}

	return res, nil
}
