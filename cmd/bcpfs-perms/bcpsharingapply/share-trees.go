package bcpsharingapply

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpsharing"
)

// `EnsureShareTrees()` applies the shared tree to the filesystem.
func EnsureShareTrees(
	lg Logger,
	fs *bcpsharing.Bcpfs,
	shareTrees bcpsharing.ShareTrees,
) error {
	for _, st := range shareTrees {
		if err := ensureShareTree(lg, fs, st); err != nil {
			return err
		}
	}
	return nil
}

func ensureShareTree(
	lg Logger,
	fs *bcpsharing.Bcpfs,
	tree bcpsharing.ShareTree,
) error {
	expected := make(map[string]string)
	for _, f := range tree.Files {
		expected[f.Path] = f.Target
	}
	existing := make(map[string]struct{})

	// Gather unexpected files in `rm` and expected existing files in
	// `existing`.
	var rm []string
	treeRoot := filepath.Join(
		fs.Rootdir, fs.OrgUnitDir, tree.OrgUnit, "shared",
	)
	rootLen := len(fs.Rootdir)
	walkFn := func(path string, inf os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if path == treeRoot {
			return nil
		}

		relpath := path[rootLen+1:]
		if target, ok := expected[relpath]; !ok {
			rm = append(rm, path)
		} else if target == "" {
			// expect dir
			if !inf.IsDir() {
				rm = append(rm, path)
			} else {
				existing[relpath] = struct{}{}
			}
		} else {
			// expect symlink
			if inf.Mode()&os.ModeSymlink != 0 {
				t, err := os.Readlink(path)
				if err != nil {
					return err
				}
				if t == target {
					existing[relpath] = struct{}{}
				} else {
					rm = append(rm, path)
				}
			} else {
				rm = append(rm, path)
			}
		}

		return nil
	}
	if err := filepath.Walk(treeRoot, walkFn); err != nil {
		return err
	}

	// Remove unexpected files in depth first order.
	sort.Sort(sort.Reverse(sort.StringSlice(rm)))
	for _, f := range rm {
		if err := os.Remove(f); err != nil {
			return err
		}
		lg.Info(fmt.Sprintf(
			"Removed unexpected sharing file %s", f,
		))
	}

	// Create missing files.
	for _, f := range tree.Files {
		if _, ok := existing[f.Path]; ok {
			continue
		}

		path := filepath.Join(fs.Rootdir, f.Path)
		if f.IsDir() {
			if err := os.Mkdir(path, 0777); err != nil {
				return err
			}
			lg.Info(fmt.Sprintf(
				"Created sharing directory %s", path,
			))
		} else if f.IsSymlink() {
			if err := os.Symlink(f.Target, path); err != nil {
				return err
			}
			lg.Info(fmt.Sprintf(
				"Created sharing symlink %s", path,
			))
		} else {
			panic("logic error")
		}
	}

	return nil
}
