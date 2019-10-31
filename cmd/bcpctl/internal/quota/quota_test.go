package quota_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/nogproject/bcpfs/cmd/bcpctl/internal/quota"
)

func TestParsesValidQuota(t *testing.T) {
	cases := []struct {
		name   string
		text   string
		quotas []quota.Quota
	}{
		{
			"basic", `
alice 1 2 3 4

# comment
bob 5 6 7 8
`, []quota.Quota{
				{"alice", 1, 2, 3, 4},
				{"bob", 5, 6, 7, 8},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qs, err := quota.Parse(strings.NewReader(tc.text))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(qs, tc.quotas) {
				t.Error("quotas differ")
			}
		})
	}
}

func TestRejectInvalidQuota(t *testing.T) {
	cases := []struct {
		name string
		text string
		err  string
	}{
		{"1-field", "a\n", "line 1: wrong number of fields"},
		{"2-fields", "\na 2\n", "line 2: wrong number of fields"},
		{"3-fields", "a 2 3\n", "wrong number of fields"},
		{"4-fields", "a 2 3 4\n", "wrong number of fields"},
		{"6-fields", "a 2 3 4 5 6\n", "wrong number of fields"},
		{"invalid-f2", "a -2 3 4 5 ", "invalid field 2"},
		{"invalid-f3", "a 2 -3 4 5 ", "invalid field 3"},
		{"invalid-f4", "a 2 3 -4 5 ", "invalid field 4"},
		{"invalid-f5", "a 2 3 4 -5 ", "invalid field 5"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := quota.Parse(strings.NewReader(tc.text))
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(fmt.Sprint(err), tc.err) {
				t.Fatalf(
					"wrong error message; want %s; got %s",
					tc.err, err,
				)
			}
		})
	}
}
