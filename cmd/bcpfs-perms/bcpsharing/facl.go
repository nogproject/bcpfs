package bcpsharing

import (
	"errors"
	"regexp"
	"strings"
)

// `FileAcls` are ACLs that use filesystem paths and groups.
type FileAcls []FileAcl

type FileAcl struct {
	Path string
	Acl  Facl
}

type Facl []FaclAce

type FaclAce struct {
	Tag  string
	Mode AceMode
}

// `AsFacl()` maps logical group names to filesystem group names.
func (acl Acl) AsFacl(fs *Bcpfs) Facl {
	facl := make([]FaclAce, 0, len(acl))
	for _, ace := range acl {
		facl = append(facl, FaclAce{
			Tag:  "group:" + fs.FsGroupOrgUnit(ace.Group),
			Mode: ace.Mode,
		})
	}
	return facl
}

// `SelectNamedGroupEntries()` returns named group normal and default entries.
func (facl Facl) SelectNamedGroupEntries() Facl {
	sel := make([]FaclAce, 0, len(facl))
	for _, ace := range facl {
		if ace.IsNamedGroup() {
			sel = append(sel, ace)
		}
	}
	return sel
}

// SelectNamedGroupNormalEntries()` returns named group normal entries,
// rejecting default entries.
func (facl Facl) SelectNamedGroupNormalEntries() Facl {
	sel := make([]FaclAce, 0, len(facl))
	for _, ace := range facl {
		if ace.IsNamedGroupNormal() {
			sel = append(sel, ace)
		}
	}
	return sel
}

func (ace FaclAce) IsNamedGroup() bool {
	return ace.IsNamedGroupNormal() || ace.IsNamedGroupDefault()
}

func (ace FaclAce) IsNamedGroupNormal() bool {
	return strings.HasPrefix(ace.Tag, "group:") &&
		ace.Tag != "group:"
}

func (ace FaclAce) IsNamedGroupDefault() bool {
	return strings.HasPrefix(ace.Tag, "default:group:") &&
		ace.Tag != "default:group:"
}

func (ace FaclAce) String() string {
	return ace.Tag + ":" + string(ace.Mode)
}

func (ace FaclAce) GroupName() string {
	fields := strings.Split(ace.Tag, ":")
	return fields[len(fields)-1]
}

func (ace FaclAce) WithoutX() FaclAce {
	return FaclAce{
		Tag:  ace.Tag,
		Mode: ace.Mode.WithoutX(),
	}
}

var rgxSplitGetfaclText = regexp.MustCompile(`\n\s*\n`)

// `ParseGetfaclText()` parses getfacl output that may contain information
// about multiple files.
func ParseGetfaclText(txt string) (FileAcls, error) {
	acls := make([]FileAcl, 0)

	txt = strings.TrimSpace(txt)
	for _, par := range rgxSplitGetfaclText.Split(txt, -1) {
		acl, err := parseGetfaclParagraph(par)
		if err != nil {
			return nil, err
		}
		acls = append(acls, acl)
	}

	return acls, nil
}

func parseGetfaclParagraph(par string) (FileAcl, error) {
	var facl FileAcl

	lines := strings.Split(par, "\n")
	if len(lines) < 1 {
		return facl, errors.New("getfacl text too short")
	}

	const head = "# file: "
	if !strings.HasPrefix(lines[0], head) {
		return facl, errors.New("missing ACL file header")
	}
	facl.Path = lines[0][len(head):]

	aces := make([]FaclAce, 0, len(lines)-4)
	for _, l := range lines {
		if strings.HasPrefix(l, "#") {
			continue
		}
		ace, err := parseGetfaclAceLine(l)
		if err != nil {
			return facl, err
		}
		aces = append(aces, ace)
	}
	facl.Acl = aces

	return facl, nil
}

var rgxGetfaclAce = regexp.MustCompile(`` +
	`^` +
	`(?:default:)?` +
	`(?:user|group|mask|other:)` +
	`\S*:` +
	`[r-][w-][x-]` +
	`$`,
)

func parseGetfaclAceLine(line string) (FaclAce, error) {
	var ace FaclAce
	if !rgxGetfaclAce.MatchString(line) {
		return ace, errors.New("malformed getfacl ACE line")
	}
	ace.Tag = line[:len(line)-4]
	ace.Mode = AceMode(line[len(line)-3:])
	return ace, nil
}
