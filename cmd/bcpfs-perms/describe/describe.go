package describe

import (
	"fmt"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/grp"

	"gopkg.in/yaml.v2"
)

func MustDescribeConfig(cfg *bcpcfg.Root) string {
	d, err := yaml.Marshal(&cfg)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return string(d)
}

func MustDescribeGroups(gs []grp.Group) string {
	d, err := yaml.Marshal(&gs)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return string(d)
}

func MustDescribeOrg(org *bcp.Organization) string {
	d, err := yaml.Marshal(&org)
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal: %v", err))
	}
	return string(d)
}
