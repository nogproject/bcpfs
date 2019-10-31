# bcpfs -- Changelog
By Steffen Prohaska
<!--@@VERSIONINC@@-->

## Introduction

This log describes changes in repo bcpfs, which contains:

- The command bcpfs-perms.
- Supporting commands.

The log sections describe releases of the repo.  Within each section, notes are
grouped by topic.

Individual tools' versions may change independently.  See `versions.yml`.

## bcpfs-2.0.0, 2019-10-31

GIT RANGE: `9a8ef409d5..129d53c6da`

Program versions:

* bcpfs-perms-2.0.0, updated
* bcpfs-chown-1.0.0, unchanged
* bcpsucd-0.4.1, unchanged
* bcpctl-0.4.1, unchanged
* Internal tools `bcpfs-file-summary-from-log`, `bcpfs-propagate-toplevel-acls`
  without version, unchanged.

General changes:

* All potentially internal information has been removed.

bcpfs-perms-2.0.0, 2019-10-31:

* Config fields `orgUnitDir`, `serviceDir`, `orgUnitPrefix`, and
  `servicePrefix` are now mandatory.

## bcpfs-1.3.0, 2019-10-30

GIT RANGE: `b86e6bdeb6..297fff1c95`

Program versions:

* bcpfs-perms-1.5.0, updated
* bcpsucd-0.4.1, updated
* bcpctl-0.4.1, updated
* bcpfs-chown-1.0.0, updated
* Internal tools `bcpfs-file-summary-from-log`, `bcpfs-propagate-toplevel-acls`
  without version, unchanged.

bcpfs-chown-1.0.0, 2019-10-30:

* Polished `bcpfs-chown` output.

bcpfs-perms-1.5.0, 2019-10-30:

* `bcpfs-perms apply --sharing` logs changes.

bcpfs-chown-0.1.0, 2019-10-24

* Debian package.

bcpfs-perms-1.4.2, 2019-10-07:

* Go 1.13.1

bcpsucd-0.4.1, bcpctl-0.4.1, 2019-10-07:

* Go 1.13.1
* go-grpc 1.24.0.  `bcpsucd` and `bcpctl` must probably be upgraded at the same
  time.

bcpfs-perms-1.4.1, 2019-10-04:

* `bcpfs-perms apply --recursive` applies correct toplevel `people/` ACLs.
  Previously, toplevel `people/` directories were not group-writable.

bcpfs-perms-1.4.0, 2019-10-02:

* `bcpfs-perms apply` has a new option `--sharing` to manage sharing ACLs and
  shared trees.  The new option will replace the `bcpshare` kit.
* `bcpfs-perms apply` has a new option `--recursive`, which is a replacement
  for `bcpfs-propagate-toplevel-acls`.

bcpfs-perms-1.3.0, 2019-07-22:

* `bcpfs-perms` handles service Unix groups that are not mentioned in the
  configuration gracefully, so that new Unix groups can be added without
  affecting cron jobs that still use the old configuration.

## bcpfs-1.2.0, 2019-07-05

Program versions:

* bcpfs-perms-1.2.3, updated
* bcpsucd-0.4.0, updated
* bcpctl-0.4.0, updated
* Internal tools `bcpfs-chown`, `bcpfs-file-summary-from-log`,
  `bcpfs-propagate-toplevel-acls` without version.

bcpfs-perms-1.2.3, 2019-07-05:

* Go 1.12.6.

bcpsucd-0.4.0, bcpctl-0.4.0, 2019-07-05:

* Go 1.12.6.
* go-grpc 1.22.0.  `bcpsucd` and `bcpctl` must probably be upgraded at the same
  time.

## bcpfs-1.1.0, 2019-05-02

Program versions:

* bcpfs-perms-1.2.2, updated
* bcpsucd-0.3.0, updated
* bcpctl-0.3.0, updated
* Internal tools `bcpfs-chown`, `bcpfs-file-summary-from-log`,
  `bcpfs-propagate-toplevel-acls` without version.

bcpsucd-0.3.0, bcpctl-0.3.0, 2018-09-20:

* Go 1.11.0.
* go-grpc 1.15.0.  `bcpsucd` and `bcpctl` must be upgraded at the same time.
  Specifically, `bcpctl-0.3.0` does not work with `bcpsucd-0.2.0`.

bcpfs-perms-1.2.2, 2018-09-20:

* Go 1.11.0.

bcpfs-perms-1.2.1, 2018-09-06:

* `bcpfs-perms` now handles duplicate groups.  It removes duplicates with
  identical name and GID.  It fails if relevant groups are ambiguous.

bcpfs-perms-1.2.0, 2018-03-15:

* Facilities can now use two different service access policies.  The policy is
  configured in `bcpfs.hcl` `facility.access`.  `perService` is the policy that
  was used before.  It requires membership in the service group.  It is the
  default if no policy is specified.  `allOrgUnits` is the new policy.
  Membership in the organization group that has been specified in the toplevel
  `bcpfs.hcl` field `superGroup` is sufficient to access the service
  directories.  See NOE-17 for details.  The new configuration fields
  `superGroup` and `facility.access` will become mandatory after a short
  transition period.

bcpfs-perms-1.1.0, 2018-02-13:

* Filters can now be configured to match multiple `services` and `orgUnits`.
  The old `service` and `orgUnit` configuration keys still work.
* `bcpfs-perms describe config` now uses lists `services` and `orgunits` to
  report the filter config.

## bcpfs-1.0.0, 2018-01-08

Program versions:

* bcpfs-perms-1.0.0, updated
* bcpsucd-0.2.0, unchanged
* bcpctl-0.2.0, unchanged

bcpfs-perms-1.0.0, 2018-01-08:

* Bump to 1.0.0 to indicate that we use `bcpfs-perms` in production.
* No changes.

bcpfs-perms-0.6.2, 2017-09-28:

* Fixed security issue with `<srv>/<ou>` ACLs.  Since bcpfs-perms-0.3.0, the
  parent `<srv>` default ACL entry for the service group incorrectly propagated
  to the `<ou>` sub-directory when creating a new directory, which allowed all
  microscopy users to access files of all other organizational units.  Existing
  directories were not affected.

bcpfs-perms-0.6.1, 2017-09-15:

* Stable describe output, ordered by group name.

## bcpfs-0.5.0, 2017-09-05

Program versions:

* bcpfs-perms-0.6.0, updated
* bcpsucd-0.2.0, unchanged
* bcpctl-0.2.0, unchanged

bcpfs-perms-0.6.0, 2017-09-05:

* NOE-14 explicit symlinks.

bcpfs-perms-0.5.0:

* Go 1.9.
* Disabled logging stack traces.
* Fixed logging file positions.

dev:

* Go 1.9.
* Switched to Go Dep for vendor management.

## bcpfs-0.4.0, 2017-08-05

Program versions:

* bcpfs-perms-0.4.0, updated
* bcpsucd-0.2.0, new
* bcpctl-0.2.0, new

bcpsucd-0.2.0, bcpctl-0.2.0:

* `bcpctl setquota` that mimics `setquota(8)` batch mode.  The root server is
  restricted to a single filesystem.

bcpsucd-0.1.1, bcpctl-0.1.0:

* New privilege separation root server and client command as described in
  NOE-12.  The initial version only provides a status operation that reports
  details about the connected client.

bcpfs-perms-0.4.0:

* The logging format has changed after upgrading to Zap 1.5.
* The version now contains more details in the build tag.

Supporting commands:

* `bcpfs-propagate-toplevel-acls` no longer sets full ACLs.  Its responsibility
  has been restricted to permissions that are handled by `bcpfs-perms`.  NOE-9
  sharing permissions are no longer affected.

dev:

* Fixed Glide cache volume mount.
* Prepared switch to vendor with Dep command.  The git dir is now required to
  be below the worktree, so that Git works in the dev container.  You need to
  move the git dir if you cloned the worktree as a submodule.
* The build tag now encodes the commit date and abbreviated hash of the head
  commit and the build Unix timestamp.  A dirty worktree is indicated.
* Go package layout recommendations have been added.

## bcpfs-0.3.0, 2017-08-02

bcpfs-perms-0.3.0:

* Go 1.8.3.
* bcpfs-perms now fails if rootdir does not exist as a protection against
  configuration errors.  rootdir must be manually created.
* The shell helpers' stdout and stderr are now correctly forwarded.
* More realistic bcpfs.hcl examples with `data` subdir.
* The filtering approach has been changed to whitelisting of service-org-unit
  combinations with automatically anchored regular expressions.
* Unexpected empty srv/ou directories are now automatically removed.
* Unexpected ou/srv symlinks are now automatically removed.
* New ACL design that works better with Samba; see NOE-10 for details.
* New config options `orgUnit.subdirs` with per-subdir permission policies
  `group`, `owner`, `manager`.  See NOE-11.  The old config option
  `orgUnit.extraDirs` is still supported for backward compatibility.
* `bcpfs-perms` now preserves named user ACL entries and named group ACL
  entries for which it is not responsible during `apply`.  It ignores them
  during `check`.  This will allow a separate implementation to manage `--x`
  directory traversal permissions.  See NOE-9.

New supporting commands:

* `bcpfs-propagate-toplevel-acls`: Fix ACLs during NOE-10 transition.
* `bcpfs-orphan-check`: List unnamed accounts with data.
* `bcpfs-orphan-reassign`: Reassign orphan data to named owners.

## bcpfs-0.2.0, 2017-02-20

bcpfs-perms-0.2.0:

* Symlinks from `x-facility` directories now point to toplevel device and
  service directories, so that operators can easily access the full trees.
* Go 1.8.0
* Removed example 'hello' project, so that `bcpfs.git` contains only production
  code.
* Switched to Glide for vendor management.
* Polished dev workflow.

## bcpfs-0.1.0, 2017-01-13

bcpfs-perms-0.1.0:

* New command line tool `bcpfs-perms` for managing the toplevel BCP directories
  and permissions.
