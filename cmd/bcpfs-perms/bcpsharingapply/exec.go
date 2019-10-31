package bcpsharingapply

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"strings"
	"text/template"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpsharing"
	"github.com/nogproject/bcpfs/pkg/execx"
)

var (
	bash = execx.MustLookTool(execx.ToolSpec{
		Program:   "bash",
		CheckArgs: []string{"--version"},
		CheckText: "GNU bash",
	})
)

// `getfaclDirPaths()` runs `cd <dir>; getfacl <paths>...`, ignoring
// missing paths.
func getfaclDirPaths(
	dir string,
	paths []string,
) (bcpsharing.FileAcls, error) {
	exists := make([]string, 0, len(paths))
	for _, p := range paths {
		if isDir(path.Join(dir, p)) {
			exists = append(exists, p)
		}
	}
	if len(exists) == 0 {
		return nil, nil // TODO
	}

	out, err := runBashXargsOutput(
		getFaclSh,
		struct{ Dir string }{dir},
		exists,
	)
	if err != nil {
		return nil, err
	}

	return bcpsharing.ParseGetfaclText(out)
}

func isDir(path string) bool {
	st, err := os.Stat(path)
	if err != nil {
		return false
	}
	return st.IsDir()
}

var getFaclSh = template.Must(template.New("getFaclSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

cd '{{ .Dir }}'

xargs -0 --no-run-if-empty \
getfacl --
`))

// `setfaclDirSubdirModify()` modifies selected directory and regular file ACL
// entries recursively below `<dir>/<path>`.
func setfaclDirSubdirModify(
	dir, path string,
	modifyDirs, modifyFiles []string,
) error {
	return runBash(
		setfaclDirSubdirModifySh,
		struct {
			Dir         string
			Path        string
			ModifyDirs  string
			ModifyFiles string
		}{
			Dir:         dir,
			Path:        path,
			ModifyDirs:  strings.Join(modifyDirs, ","),
			ModifyFiles: strings.Join(modifyFiles, ","),
		},
	)
}

var setfaclDirSubdirModifySh = template.Must(
	template.New("setfaclDirSubdirModifySh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

cd '{{ .Dir }}'

find '{{ .Path }}' -type d -print0 \
| xargs -0 --no-run-if-empty \
setfacl -nm {{ .ModifyDirs }} --

find '{{ .Path }}' -type f -print0 \
| xargs -0 --no-run-if-empty \
setfacl -nm {{ .ModifyFiles }} --
`))

// `setfaclDirSubdirModify()` removes selected directory and regular file ACL
// entries recursively below `<dir>/<path>`.
func setfaclDirSubdirRemove(
	dir, path string,
	remove []string,
) error {
	return runBash(
		setfaclDirSubdirRemoveSh,
		struct {
			Dir    string
			Path   string
			Remove string
		}{
			Dir:    dir,
			Path:   path,
			Remove: strings.Join(remove, ","),
		},
	)
}

var setfaclDirSubdirRemoveSh = template.Must(
	template.New("setfaclDirSubdirRemoveSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

cd '{{ .Dir }}'

find '{{ .Path }}' -type d -print0 -or -type f -print0 \
| xargs -0 --no-run-if-empty \
setfacl -nx {{ .Remove }} --
`))

// `setfaclDirPathsTraversal()` adds traversal `--x` ACL entries for group
// `fsGroup` to `paths`, which are relative to `<dir>`.
func setfaclDirPathsTraversal(
	dir string,
	paths []string,
	fsGroup string,
) error {
	return runBashXargs(
		setfaclDirPathsTraversalSh,
		struct {
			Dir   string
			Group string
		}{
			Dir:   dir,
			Group: fsGroup,
		},
		paths,
	)
}

var setfaclDirPathsTraversalSh = template.Must(
	template.New("setfaclDirPathsTraversalSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

cd '{{ .Dir }}'

xargs -0 --no-run-if-empty \
setfacl -nm group:{{ .Group }}:--x --
`))

// `runBash()` runs the template `sh` with parameters from `data`.
func runBash(sh *template.Template, data interface{}) error {
	c := exec.Command(bash.Path, "-c", mustRender(sh, data))
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// `runBashXargs()` runs the template `sh` with parameters from `data` and
// NUL-separated `xargs` on stdin.
func runBashXargs(
	sh *template.Template,
	data interface{},
	xargs []string,
) error {
	c := exec.Command(bash.Path, "-c", mustRender(sh, data))
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	stdin, err := c.StdinPipe()
	if err != nil {
		return err
	}
	closeStdin := func() error {
		if stdin == nil {
			return nil
		}
		e := stdin.Close()
		stdin = nil
		return e
	}

	errC := make(chan error, 1)
	go func() {
		defer closeStdin()
		for _, a := range xargs {
			if _, err := io.WriteString(stdin, a); err != nil {
				errC <- err
				return
			}
			if _, err = stdin.Write([]byte{0}); err != nil {
				errC <- err
				return
			}
		}
		errC <- closeStdin()
	}()

	if err = c.Run(); err != nil {
		return err
	}
	return <-errC
}

// `runBashXargsOutput()` runs the template `sh` with parameters from `data`
// and NUL-separated `xargs` on stdin, returning stdout.
func runBashXargsOutput(
	sh *template.Template,
	data interface{},
	xargs []string,
) (string, error) {
	c := exec.Command(bash.Path, "-c", mustRender(sh, data))
	c.Stderr = os.Stderr

	stdin, err := c.StdinPipe()
	if err != nil {
		return "", err
	}
	closeStdin := func() error {
		if stdin == nil {
			return nil
		}
		e := stdin.Close()
		stdin = nil
		return e
	}

	errC := make(chan error, 1)
	go func() {
		defer closeStdin()
		for _, a := range xargs {
			if _, err := io.WriteString(stdin, a); err != nil {
				errC <- err
				return
			}
			if _, err = stdin.Write([]byte{0}); err != nil {
				errC <- err
				return
			}
		}
		errC <- closeStdin()
	}()

	out, err := c.Output()
	if err != nil {
		return "", err
	}
	err = <-errC
	if err != nil {
		return "", err
	}

	return string(out), nil
}

// `mustRender()` renders a template.  It panics if rendering fails; the caller
// must ensure a valid combination of template and `data`.
func mustRender(t *template.Template, data interface{}) string {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		msg := fmt.Sprintf("Failed to render template: %v", err)
		panic(msg)
	}
	return b.String()
}
