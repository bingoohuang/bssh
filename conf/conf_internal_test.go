package conf

import "testing"

func TestReplacePasswordOnly(t *testing.T) {
	tcs := []struct {
		name        string
		input       string
		pass        string
		replacement string
		want        string
		ok          bool
	}{
		{
			name:        "replace password in user:pass@host",
			input:       "root:root@172.16.235.133",
			pass:        "root",
			replacement: "{PBE}xxx",
			want:        "root:{PBE}xxx@172.16.235.133",
			ok:          true,
		},
		{
			name:        "replace password in user/pass@host",
			input:       "root/root@172.16.235.133",
			pass:        "root",
			replacement: "{PBE}xxx",
			want:        "root/{PBE}xxx@172.16.235.133",
			ok:          true,
		},
		{
			name:        "do not replace when password does not match",
			input:       "root:admin@172.16.235.133",
			pass:        "root",
			replacement: "{PBE}xxx",
			want:        "root:admin@172.16.235.133",
			ok:          false,
		},
		{
			name:        "do not replace when no credential separator",
			input:       "172.16.235.133",
			pass:        "root",
			replacement: "{PBE}xxx",
			want:        "172.16.235.133",
			ok:          false,
		},
	}

	for _, tc := range tcs {
		got, ok := replacePasswordOnly(tc.input, tc.pass, tc.replacement)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("%s: got (%q, %v), want (%q, %v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
