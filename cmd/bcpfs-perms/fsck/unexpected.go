package fsck

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/nogproject/bcpfs/pkg/execx"
)

// `CheckNoUnexpected()` checks for unexpected paths.  It returns `ok=false` if
// there are unexpected paths and `ok=true` if there are none.  `err` is only
// used to report problems that prevented checking, like an error accessing the
// filesystem.
func CheckNoUnexpected(
	subroot string, entries []Entry, symlinks map[string]string,
) (ok bool, err error) {
	pathSet := map[string]bool{}
	for _, e := range entries {
		pathSet[e.Path] = true
	}

	paths, err := findPaths(subroot)
	if err != nil {
		return false, fmt.Errorf(
			"failed to list `%s`: %v", subroot, err,
		)
	}

	ok = true
	for _, p := range paths {
		if pathSet[p] {
			continue
		}
		if _, ok := symlinks[p]; ok {
			continue
		}
		ok = false
		msg := fmt.Sprintf("Unexpected path `%s`", p)
		logger.Error(msg)
	}

	return ok, nil
}

func findPaths(subroot string) ([]string, error) {
	out, err := exec.Command(
		find.Path, subroot, "-maxdepth", "2", "-print0",
	).Output()
	if err != nil {
		return nil, err
	}
	sep := "\000"
	return strings.Split(strings.TrimRight(string(out), sep), sep), nil
}

var find = execx.MustLookTool(execx.ToolSpec{
	Program:   "find",
	CheckArgs: []string{"--version"},
	CheckText: "GNU findutils",
})
