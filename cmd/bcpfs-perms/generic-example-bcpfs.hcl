# `generic-example-bcpfs.hcl` contains explanations for all available settings
# fields and uses generic names for illustration.

# To see the current default settings, run `bcpfs-perms describe config` with a
# minimal config file that contains only a `rootdir` statement.


# `rootdir` is the path to the root directory of the managed filesystem.  The
# toplevel folders for services and organizational units are managed as direct
# subfolders of the `rootdir`.
rootdir = "/orgfs/data"

# `serviceDir` is the toplevel subdirectory that contains the facility service
# directories.
#
# Example: `serviceDir=srv` -> `/orgfs/data/srv`.
serviceDir = "srv"

# `orgUnitDir` is the toplevel subdirectory that contains the organizational
# unit directories.
#
# Example: `orgUnitDir=org` -> `/orgfs/data/org`.
orgUnitDir = "org"

# `superGroup` is the Unix group that contains all members of all orgUnits.  It
# will be used to realize `allOrgUnits` permission between services and
# orgUnits.
superGroup = "ag_org"

# `orgUnitPrefix` is the Unix group prefix for organizational units.
#
# Example: `orgUnitPrefix=org`, Unix groups `org_foo`, `org_bar`.
orgUnitPrefix = "org"

# `servicePrefix` is the Unix group prefix for services and devices.
#
# Example: `servicePrefix=srv`, Unix groups `srv_foo`, `srv_bar`.
servicePrefix = "srv"

# `opsSuffix` is the Unix group suffix for facility operations groups.
#
# Example: `opsSuffix=ops`, Unix group `srv_em-ops`.
opsSuffix = "ops"

# `facilitySuffix` is the Unix group suffix for facility organizational unit
# groups.
#
# Example: `facilitySuffix=facility`, Unix group `org_lm-facility`.
facilitySuffix = "facility"


# `facility` describes the facility `facility.name`, which manages
# `facility.services`.  Devices are listed as services.  The `facility`
# statement can be repeated.  Services are automatically parsed from the Unix
# groups and `servicePrefix`.  Every service must be assigned to exactly one
# facility.  Th access policy of a facility must be configured to either
# `perService` or `allOrgUnits`.
#
# Example directories for config below: `/orgfs/data/srv/rem-707/...`, ... .
facility {
    name = "em"
    services = [
        "tem-505",
        "rem-707",
        "em-analysis",
    ]
    access = "perService"
}

facility {
    name = "lm"
    services = [
        "spim-100",
        "spim-222",
    ]
    access = "perService"
}

facility {
    name = "ms"
    services = [
        "ms-data",
    ]
    access = "allOrgUnits"
}

facility {
    name = "fake"
    services = [
        "fake-analysis",
        "fake-tem",
    ]
    access = "perService"
}

# `orgUnit` describes special configuration settings for the organizational
# unit `orgUnit.name`.
#
# `orgUnit.dirs` is a list of additional directories in the organizational unit
# tree with access policies.  Access policies:
#
# - `owner`: Users can create subdirs that are owner read-write, the
#   organizational unit group can read.
# - `group`: Organizational unit group read-write.
# - `manager`: root manages the directory tree; the organizational unit group
#   can read.
#
# Example: `dirs=[{name:people policy:owner}]` ->
# `/orgfs/data/org/ag-alice/people` with owner read-write, group read.
#
# `orgUnit.extraDirs` is supported for backward compatibility.  Entries are
# automatically added to `dirs` with policy `group`.
#
# Organization units that are not listed have no additional dirs.
orgUnit {
    name = "ag-alice"
    subdirs = [
        { name = "people", policy = "owner" },
        { name = "service", policy = "manager" },
        { name = "shared", policy = "manager" },
    ]
    extraDirs = [
        "projects",
    ]
}

orgUnit {
    name = "ag-bob"
    subdirs = [
        { name = "shared", policy = "manager" },
    ]
}

orgUnit {
    name = "em-facility"
    subdirs = [
        { name = "shared", policy = "manager" },
    ]
}

# `filter` rules define the allowed combinations of service and org unit.  A
# filter rule defines one regex for `service` and one for `orgUnit`.  Regexes
# are automatically anchored to the beginning "^" and end "$" of names. If both
# regexes match, the `action` is applied.  The `action` can be `accept` of
# `reject`.  The order of filter rules matters.  If no rule matches, the
# default is to `reject`.
#
# Example: With the rules below, directories `/orgfs/data/srv/*/nog` and
# symlinks `/orgfs/data/org/nog/*` will be rejected.  Combinations that have
# 'fake' in both `service` and `orgUnit` will be accepted.  Combinations that
# have 'fake' only in one component will be rejected.  `orgUnits` that start
# with `ag-` will be accepted.  All other `orgUnits` will be rejected.
filter {
    service = ".*"
    orgUnit = "nog"
    action = "reject"
}

filter {
    service = "fake.*"
    orgUnit = ".*fake.*"
    action = "accept"
}

filter {
    service = "fake.*"
    orgUnit = ".*"
    action = "reject"
}

filter {
    service = ".*"
    orgUnit = ".*fake.*"
    action = "reject"
}

# em: full ag-* list for all services
filter {
    services = [
        "tem-505",
        "rem-707",
        "em-analysis",
    ]
    orgUnit = "ag-.*"
    action = "accept"
}

# ms: reduced ag-* list for service folder
filter {
    service = "ms-data"
    orgUnits = [
        "ag-alice",
    ]
    action = "accept"
}


# `symlink` entries define a list of explicit symlinks.
symlink {
    target = "../../fake-facility/service/guides"
    path = "srv/fake-tem/guides"
}

symlink {
    target = "../../fake-facility/service/guides"
    path = "srv/fake-analysis/guides"
}

# `sharing` specifies the `<ou>/shared` trees.  See NOE-9 for a general
# description.
sharing {
    namingPolicy { action = "allow", match = "em-facility/service/guides(/.*)?" }
    namingPolicy { action = "allow", match = "em-facility/tem-505(/.*)?" }
    namingPolicy { action = "allow", match = "ag-bob/tem-505(/.*)?" }

    export {
        path = "em-facility/service/guides"
        acl = [
            "group:ag-alice:r-x",
            "group:ag-bob:r-x",
        ]
    }

    export {
        path = "em-facility/tem-505/ag-bob/foo"
        acl = [
            "group:ag-alice:r-x",
        ]
    }
    export {
        path = "em-facility/tem-505/ag-bob/bar"
        acl = [
            "group:ag-alice:r-x",
        ]
    }

    export {
        path = "ag-bob/tem-505/foo"
        acl = [
            "group:ag-alice:r-x",
        ]
    }

    import { action = "accept", group = "ag-alice", match = "em-facility/.*" }
    import { action = "accept", group = "ag-alice", match = "ag-bob/.*" }
    import { action = "accept", group = "ag-bob", match = "em-facility/service/.*" }
}
