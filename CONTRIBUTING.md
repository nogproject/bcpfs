# Contributing to Bcpfs

## Submitting Changes

Bcpfs uses the Nog contributing workflow.

Submit changes with `Signed-off-by: Full Name <email>` to confirm that you have
the right to contribute as open source as described in the Developer
Certificate of Origin <http://developercertificate.org>.

Contact Steffen Prohaska, <prohaska@zib.de> for details.

See Go style below.

## Docker dev workflow

Aliases:

```bash
source ./tools/env.sh
```

`.git` must be the git dir in order for Git commands to work in the dev
container.  See `tools/env.sh` for a sequence of commands to replace a git file
with the corresponding git dir.

Makefile usage:

```bash
make help
```

Init vendor and build shared binaries:

```bash
make
```

Build binaries:

```bash
make binaries
```

Clean up old containers:

```bash
make gc
```

Clean up Docker objects that can be quickly re-created:

```bash
make down
```

Full cleanup, including stateful volumes:

```bash
make down-state
```

State is maintained in volumes.  You need to delete the `go` volume after
modifying the `godev` image, so that the volume is re-created with the update
image content.

Godoc:

```bash
make up-godoc
open http://localhost:6060/pkg/github.com/nogproject/?m=all
```

Re-build and set up from scratch:

```
docker-compose build
make clean
make
```

## How to release?

Managing versions: see `versions.yml` and `gitk -- versions.yml CHANGELOG.md`.

Test that the new version is backward compatible before releasing a Deb:

```bash
fileserver=...

make binaries
scp product/bin/bcpfs-perms ${fileserver}:/tmp
ssh ${fileserver} /tmp/bcpfs-perms describe config
ssh ${fileserver} /tmp/bcpfs-perms describe groups
ssh ${fileserver} /tmp/bcpfs-perms describe org
ssh ${fileserver} sudo /tmp/bcpfs-perms check
ssh ${fileserver} rm /tmp/bcpfs-perms
```

Make and test bcpfs-perms.deb with:

```bash
rm -rf product/deb && make deb

docker run -it --rm -v "$(pwd)/product/deb:/deb" bcpfsfake:latest bash
apt-get install /deb/bcpfs-perms_*_amd64.deb
dpkg -l bcpfs-perms
dpkg -L bcpfs-perms
bcpfs-perms --version

ln -s /usr/share/doc/bcpfs/generic-example-bcpfs.hcl /etc/bcpfs.hcl
mkdir -p /orgfs/data
bcpfs-perms apply
find /orgfs -ls

bcpfs-perms apply --sharing
find /orgfs -ls
```

Distribute the deb files or install them directly.

To install `bcpfs-perms` directly:

```bash
fileserver=...
version=$(grep ^bcpfs-perms: versions.yml | cut -d : -f 2 | tr -d ' ') && echo "version: ${version}"

scp "product/deb/bcpfs-perms_${version}_amd64.deb" "${fileserver}:/tmp" &&
ssh "${fileserver}" sudo apt-get install "/tmp/bcpfs-perms_${version}_amd64.deb" &&
ssh "${fileserver}" rm "/tmp/bcpfs-perms_${version}_amd64.deb"
```

To install `bcpfs-chown` directly:

```bash
fileserver=...
version=$(grep ^bcpfs-chown: versions.yml | cut -d : -f 2 | tr -d ' ') && echo "version: ${version}"

scp "product/deb/bcpfs-chown_${version}_amd64.deb" "${fileserver}:/tmp" &&
ssh "${fileserver}" sudo apt-get install "/tmp/bcpfs-chown_${version}_amd64.deb" &&
ssh "${fileserver}" rm "/tmp/bcpfs-chown_${version}_amd64.deb"
```

## bcpfs-perms

Dev `bcpfs-perms` command:

```bash
make
dfake bcpfs-perms version
dfake bcpfs-perms describe config
 # ...
```

## bcpsuc

DEPRECATED 2019-10: bcpsuc has been deprecated.  We will likely retire and
remove it until 2020.

Try with:

```bash
dfake bash -c '
  mkdir /var/run/bcpsucd &&
  bcpsucd --conn-allow-uids=0,1,2 --status-allow-uids=0,1 &
  while ! [ -e /var/run/bcpsucd/socket ]; do true; done &&
  ls -l /var/run/bcpsucd/socket &&
  echo "# root:" && /go/bin/bcpctl status &&
  echo "# daemon:" && su -ps /bin/sh daemon -c "/go/bin/bcpctl status" &&
  echo "# bin:" && ! su -ps /bin/sh bin -c "/go/bin/bcpctl status" &&
  echo "# nobody:" && ! su -ps /bin/sh nobody -c "/go/bin/bcpctl status" &&
  kill -s TERM $(pgrep bcpsucd) &&
  wait &&
  echo done
'
```

```bash
 # Echos setquota input.
dfake bash -c '
  mkdir /var/run/bcpsucd &&
  bcpsucd \
    --conn-allow-uids=0,1,2 \
    --status-allow-uids=0,1 \
    --quota-allow-uids=0 &
    --quota-dry-run &
  while ! [ -e /var/run/bcpsucd/socket ]; do true; done &&
  bcpctl setquota --user --batch /nonexistent <<<"alice 1 2 3 4" &&
  kill -s TERM $(pgrep bcpsucd) &&
  wait &&
  echo done
'

 # Fails on real setquota
dfake bash -c '
  mkdir /var/run/bcpsucd &&
  bcpsucd \
    --conn-allow-uids=0,1,2 \
    --status-allow-uids=0,1 \
    --quota-allow-uids=0 &
  while ! [ -e /var/run/bcpsucd/socket ]; do true; done &&
  bcpctl setquota --user --batch /nonexistent <<<"alice 1 2 3 4" &&
  kill -s TERM $(pgrep bcpsucd) &&
  wait &&
  echo done
'
```

## Go style

Follow "Effective Go", and keep this section short.

### Vendor

We rely on Dep and do not track `vendor/` in Git.  A build, thus, may require
network access.  Switching between commits may require an explicit `make
vendor`.  But our Git history contains no vendor noise.

Relying on Dep seems to be a good trade-off.  If we want to track `vendor/`
more rigorously in the future, we would consider tracking it in a Git submodule
to avoid the vendor noise in the main repo.

### Protobuf

We rely on Protoc to compile `.proto` files to `.pb.go` files and do not track
`.pb.go` files in Git.  It avoids noise in the Git history; a single `make`
just works; and there are no dependent packages that would rely on compiled
`.pb.go` files.

### Go package layout

Focus on decoupling when considering a package layout.   `cmd/`, `internal/`,
and `pkg/` are possible locations.  The Bill Kennedy way seems overall more
useful than the Ben Johnson way.  See references.

In particular, do not introduce a central package with domain types.  Accept
interfaces and return structs.  Some type duplication is acceptable if it
improves decoupling.  Err on the side of over packaging, since packages are
often difficult to split later.  But be pragmatic: it can be reasonable to
import sibling packages.

Packages in `pkg/` should be more on the kit side.  They should be generic and
candidates for importing from the outside.  But the outside should not import
a package unless the package states that it is ready to be imported.

Packages in `internal/` are more on the application side.  For example, server
implementation could be placed in `internal/`, since it should not be used from
the outside; while protos and client packages could be placed in `pkg/`,
because they are more likely to be used from the outside.

Packages that are imported from the outside, such as Zap or GRPC, should be
wrapped, either on the general `pkg/` or `internal/` level or below topic
packages, depending on how specific they are.  Examples `pkg/zap`,
`internal/suc/grpcd`.

Packages that are closely related to outside packages should be place in the
`pkg/` tree under a topic.  Example: `pkg/grpc/ucred/`.

Protobuf Buffer Go packages should be indicated by `pb`.  But the Protocol
Buffer package itself should not use `pb`.  Example: `pkg/suc/sucpb/suc.proto`:

```
syntax = "proto3";
package suc;
option go_package = "sucpb";
```

Packages that are related to servers should be indicated by `d`.  Example:
`internal/suc/statusd/`.

References:

* Carlisia Pinto, Go and a Package Focused Design,
  <https://blog.gopheracademy.com/advent-2016/go-and-package-focused-design/>.
* Bill Kennedy, Package Oriented Design,
  <https://www.goinggo.net/2017/02/package-oriented-design.html>.
* Ben Johnson, Standard Package Layout,
  <https://medium.com/@benbjohnson/standard-package-layout-7cdbc8391fc1>.
