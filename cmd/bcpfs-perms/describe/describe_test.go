package describe

import (
	"bytes"
	"flag"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcp"
	"github.com/nogproject/bcpfs/cmd/bcpfs-perms/bcpcfg"
)

var update = flag.Bool("update", false, "update .golden files")

const testDataDir = "testdata"
const configFile = "config.hcl"

// The `describe` test run as usual with `make test`. If the golden files need
// an update, they can be overwritten by the current output with:
// `ddev go test ./cmd/bcpfs-perms/describe/ -update`

func TestDescribeConfig(t *testing.T) {
	path := filepath.Join(testDataDir, configFile)
	d, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
		return
	}
	cfg, _ := bcpcfg.Parse(string(d))

	got := MustDescribeConfig(cfg)
	compareToGoldenFile([]byte(got), testDataDir, t)
}

func TestDescribeOrgWithoutSuperGroup(t *testing.T) {
	path := filepath.Join(testDataDir, configFile)
	d, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
		return
	}
	cfg, _ := bcpcfg.Parse(string(d))
	org, _ := bcp.New(gs, cfg)

	got := MustDescribeOrg(org)
	compareToGoldenFile([]byte(got), testDataDir, t)
}

func TestDescribeOrgWithSuperGroup(t *testing.T) {
	path := filepath.Join(testDataDir, configFile)
	d, err := ioutil.ReadFile(path)
	if err != nil {
		t.Error(err)
		return
	}
	cfg, _ := bcpcfg.Parse(string(d))
	cfg.SuperGroup = "ag_org"
	org, _ := bcp.New(gs, cfg)

	got := MustDescribeOrg(org)
	compareToGoldenFile([]byte(got), testDataDir, t)
}

func compareToGoldenFile(got []byte, testDataDir string, t *testing.T) {
	golden := filepath.Join(testDataDir, t.Name()+".golden")
	if *update {
		ioutil.WriteFile(golden, got, 0666)
	}
	expected, err := ioutil.ReadFile(golden)
	if err != nil {
		t.Error(err)
		return
	}
	if !bytes.Equal(got, expected) {
		t.Errorf("%s:\n**got**\n%s\n**want**\n%s\n",
			t.Name(), got, expected)
	}
}
