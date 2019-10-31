package fsapply

// See design document for permissions.

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"text/template"

	"github.com/nogproject/bcpfs/pkg/execx"
)

// Less obvious programs are verified during init, so that the scripts do not
// fail for trivial reasons later.
var (
	bash = execx.MustLookTool(execx.ToolSpec{
		Program:   "bash",
		CheckArgs: []string{"--version"},
		CheckText: "GNU bash",
	})
	setfacl = execx.MustLookTool(execx.ToolSpec{
		Program:   "setfacl",
		CheckArgs: []string{"--version"},
		CheckText: "setfacl 2",
	})
)

// The scripts create only a single directory level, so that a configuration
// that points to a missing rootdir will fail.
//
// Use `setfacl -M`, so that other named group entries, like `--x` traversal,
// are preserved.

var ensureToplevelSh = template.Must(template.New("ensureToplevelSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:root '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::r-x
other::r-x
EOF

`))

var ensureServiceSh = template.Must(template.New("ensureServiceSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:{{ .Gid }} '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .Gid }}:r-x
group:{{ .OpsGid }}:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:r-x
default:group:{{ .OpsGid }}:r-x
default:mask::r-x
default:other::---
EOF

setfacl -X- '{{ .Path }}' <<EOF
group:{{ .SuperGid }}
default:group:{{ .SuperGid }}
EOF

`))

var ensureServiceAllOrgUnitsSh = template.Must(
	template.New("ensureServiceAllOrgUnitsSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:{{ .Gid }} '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .SuperGid }}:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:{{ .SuperGid }}:r-x
default:mask::r-x
default:other::---
EOF

setfacl -X- '{{ .Path }}' <<EOF
group:{{ .Gid }}
group:{{ .OpsGid }}
default:group:{{ .Gid }}
default:group:{{ .OpsGid }}
EOF

`))

// `ensureSOUSh` adds ou `.Gid` and ops `.OpsGid` ACL entries and removes
// parent srv `.SrvGid` or `.SuperGid` ACL entries, which propagated during
// `mkdir`.
var ensureSOUSh = template.Must(template.New("ensureSOUSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:"{{ .Gid }}" '{{ .Path }}'
chmod g+s "{{ .Path }}"

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .Gid }}:rwx
group:{{ .OpsGid }}:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:rwx
default:group:{{ .OpsGid }}:rwx
default:mask::rwx
default:other::---
EOF

setfacl -X- '{{ .Path }}' <<EOF
group:{{ .SrvGid }}:
group:{{ .SuperGid }}:
default:group:{{ .SrvGid }}:
default:group:{{ .SuperGid }}:
EOF

`))

// See comment at `findXargsIncSh`.
var ensureSOURecursiveSh = template.Must(
	template.New("ensureSOURecursiveSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

dirAcl="$(mktemp -t 'dir.acl.XXXXXXXXX')"
fileAcl="$(mktemp -t 'file.acl.XXXXXXXXX')"
trap 'rm "${dirAcl}" "${fileAcl}"' EXIT

cat >"${dirAcl}" <<EOF
user::rwx
group::---
group:{{ .Gid }}:rwx
group:{{ .OpsGid }}:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:rwx
default:group:{{ .OpsGid }}:rwx
default:mask::rwx
default:other::---
EOF

cat >"${fileAcl}" <<EOF
user::rw-
group::---
group:{{ .Gid }}:rwx
group:{{ .OpsGid }}:rwx
mask::rw-
other::---
EOF

` + findXargsIncSh))

// `findXargsIncSh` is included by scripts to run `find | xargs` in order to
// set owning groups, SGID bits, and ACLs for files below a toplevel directory.
// The toplevel directory itself is left unmodified.
//
// The scripts must prepare files `dirAcl` and `fileAcl` for the subdirs before
// including `findXargsIncSh`.  `dirAcl` has normal and default entries:
//
//  - the toplevel default ACL becomes the normal ACL;
//  - the toplevel default ACL is propagated.
//
// `fileAcl` has only normal entries:
//
//  - based on the `dirAcl` normal entries;
//  - without x-bit for user, mask, and other entries; but keeping the x-bit
//    for group entries, so that the effective group permissions are only
//    restricted via mask.
//
// Owning group: Run `chgrp` only if necessary in order to avoid unnecessary
// ctime changes.  `chgrp` always updates the ctime even if the group is
// unmodified.
//
// SGID:  Run `chmod` only if necessary in order to avoid unnecessary ctime
// changes.  `chmod` always updates the ctime even if the permissions are
// unmodified.
const findXargsIncSh = `
# Owning group.
find '{{ .Path }}' -mindepth 1 -not -gid {{ .Gid }} -print0 \
| xargs -0 --no-run-if-empty \
chgrp --no-dereference {{ .Gid }} --

# SGID.
find '{{ .Path }}' -mindepth 1 -type d -not -perm -g+s -print0 \
| xargs -0 --no-run-if-empty \
chmod g+s --

# Modify dir ACLs.
find '{{ .Path }}' -mindepth 1 -type d -print0 \
| xargs -0 --no-run-if-empty setfacl --modify-file="${dirAcl}" --

# Modify file ACLs.
find '{{ .Path }}' -mindepth 1 -type f -print0 \
| xargs -0 --no-run-if-empty setfacl --modify-file="${fileAcl}" --
`

var ensureOrgUnitSh = template.Must(template.New("ensureOrgUnitSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:{{ .Gid }} '{{ .Path }}'
chmod g+s '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .Gid }}:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:r-x
default:mask::r-x
default:other::---
EOF

`))

// See NOE-11 policy `group`.
var ensureOrgUnitGroupSubdirSh = template.Must(
	template.New("ensureOrgUnitGroupSubdirSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:{{ .Gid }} '{{ .Path }}'
chmod g+s '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .Gid }}:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:rwx
default:mask::rwx
default:other::---
EOF

`))

// See comment at `findXargsIncSh`.
var ensureOrgUnitGroupSubdirRecursiveSh = template.Must(
	template.New("ensureOrgUnitGroupSubdirRecursiveSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

dirAcl="$(mktemp -t 'dir.acl.XXXXXXXXX')"
fileAcl="$(mktemp -t 'file.acl.XXXXXXXXX')"
trap 'rm "${dirAcl}" "${fileAcl}"' EXIT

cat >"${dirAcl}" <<EOF
user::rwx
group::---
group:{{ .Gid }}:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:rwx
default:mask::rwx
default:other::---
EOF

cat >"${fileAcl}" <<EOF
user::rw-
group::---
group:{{ .Gid }}:rwx
mask::rw-
other::---
EOF

` + findXargsIncSh))

// See NOE-11 policy `owner`.
var ensureOrgUnitOwnerSubdirSh = template.Must(
	template.New("ensureOrgUnitOwnerSubdirSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:{{ .Gid }} '{{ .Path }}'
chmod g+s '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .Gid }}:rwx
mask::rwx
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:r-x
default:mask::r-x
default:other::---
EOF

`))

// See comment at `findXargsIncSh`.
var ensureOrgUnitOwnerSubdirRecursiveSh = template.Must(
	template.New("ensureOrgUnitOwnerSubdirRecursiveSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

dirAcl="$(mktemp -t 'dir.acl.XXXXXXXXX')"
fileAcl="$(mktemp -t 'file.acl.XXXXXXXXX')"
trap 'rm "${dirAcl}" "${fileAcl}"' EXIT

cat >"${dirAcl}" <<EOF
user::rwx
group::---
group:{{ .Gid }}:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:r-x
default:mask::r-x
default:other::---
EOF

cat >"${fileAcl}" <<EOF
user::rw-
group::---
group:{{ .Gid }}:r-x
mask::r--
other::---
EOF

` + findXargsIncSh))

// See NOE-11 policy `manager`.
var ensureOrgUnitManagerSubdirSh = template.Must(
	template.New("ensureOrgUnitOwnerSubdirSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

if ! [ -d '{{ .Path }}' ]; then
    mkdir '{{ .Path }}'
fi
chown root:{{ .Gid }} '{{ .Path }}'
chmod g+s '{{ .Path }}'

setfacl -M- '{{ .Path }}' <<EOF
user::rwx
group::---
group:{{ .Gid }}:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:r-x
default:mask::r-x
default:other::---
EOF

`))

// See comment at `findXargsIncSh`.
var ensureOrgUnitManagerSubdirRecursiveSh = template.Must(
	template.New("ensureOrgUnitOwnerSubdirRecursiveSh").Parse(`
set -o errexit -o nounset -o pipefail -o noglob

dirAcl="$(mktemp -t 'dir.acl.XXXXXXXXX')"
fileAcl="$(mktemp -t 'file.acl.XXXXXXXXX')"
trap 'rm "${dirAcl}" "${fileAcl}"' EXIT

cat >"${dirAcl}" <<EOF
user::rwx
group::---
group:{{ .Gid }}:r-x
mask::r-x
other::---
default:user::rwx
default:group::---
default:group:{{ .Gid }}:r-x
default:mask::r-x
default:other::---
EOF

cat >"${fileAcl}" <<EOF
user::rw-
group::---
group:{{ .Gid }}:r-x
mask::r--
other::---
EOF

` + findXargsIncSh))

// `runBash()` runs the template `sh` with the placeholders filled in from
// `data`.
func runBash(sh *template.Template, data interface{}) error {
	c := exec.Command(bash.Path, "-c", mustRender(sh, data))
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}

// `mustRender()` renders a template.  It panics if rendering fails; the caller
// must ensure a valid combination of template and `data`.
func mustRender(t *template.Template, data interface{}) string {
	var b bytes.Buffer
	if err := t.Execute(&b, data); err != nil {
		msg := fmt.Sprintf("Failed to render template: %v", err)
		logger.Panic(msg)
	}
	return b.String()
}
