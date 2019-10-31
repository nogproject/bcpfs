rootdir = "/fsroot"

serviceDir = "srv"
orgUnitDir = "org"
superGroup = ""

orgUnitPrefix = "org"
servicePrefix = "srv"
opsSuffix = "ops"
facilitySuffix = "facility"

facility {
    name = "lm"
    services = [
        "mic1",
        "mic2",
    ]
    access = "perService"
}

orgUnit {
    name = "ag-foo"
    subdirs = [
        { name = "people", policy = "owner" },
        { name = "service", policy = "group" },
        { name = "shared", policy = "manager" },
    ]
    extraDirs = [
        "projects",
    ]
}

symlink {
    target = "org/lm-facility/service/guides"
    path = "srv/mic1/guides"
}

filter {
    services = [
        "mic1",
        "mic2",
    ]
    orgUnits = [
        "ag_foo",
    ]
    action = "accept"
}
