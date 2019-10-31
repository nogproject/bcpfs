package fsck

import (
	"fmt"
	"os"
)

// `CheckSymlinks()` verifies the symlink `entries`.
func CheckSymlinks(entries []Entry) (ok bool, err error) {
	ok = true
	for _, p := range entries {
		if !p.IsSymlink {
			continue
		}
		reason, err := checkSymlink(p.LinkDest, p.Path)
		if err != nil {
			return false, err
		}
		if reason == "" {
			continue
		}
		ok = false
		msg := fmt.Sprintf(
			"symlink `%s` failure; expected target `%s`: %s",
			p.Path, p.LinkDest, reason,
		)
		logger.Error(msg)
	}
	return ok, nil
}

// `CheckExplicitSymlinks()` verifies `symlinks`, where the keys are symlink
// paths and the values are symlink targets.
func CheckExplicitSymlinks(symlinks map[string]string) (ok bool, err error) {
	ok = true
	for path, target := range symlinks {
		reason, err := checkSymlink(target, path)
		if err != nil {
			return false, err
		}
		if reason == "" {
			continue
		}
		ok = false
		msg := fmt.Sprintf(
			"explicit symlink `%s` failure; expected target `%s`: %s",
			path, target, reason,
		)
		logger.Error(msg)
	}
	return ok, nil
}

func checkSymlink(dest string, path string) (reason string, err error) {
	st, err := os.Lstat(path)
	if err != nil {
		return err.Error(), nil
	}
	if st.Mode()&os.ModeSymlink == 0 {
		return "not a symlink", nil
	}
	actual, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if actual != dest {
		return fmt.Sprintf("got `%s`", actual), nil
	}
	return "", nil
}
