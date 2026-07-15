// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import (
	"strings"
	"testing"
)

func TestParseOSRelease(t *testing.T) {
	content := `
# comment
NAME="Ubuntu"
VERSION_ID="24.04"
PRETTY_NAME="Ubuntu 24.04.2 LTS"
BUILD_ID='noble'
ID=ubuntu
BROKEN_LINE
`

	got := parseOSRelease(content)
	want := map[string]string{
		"NAME":        "Ubuntu",
		"VERSION_ID":  "24.04",
		"PRETTY_NAME": "Ubuntu 24.04.2 LTS",
		"BUILD_ID":    "noble",
		"ID":          "ubuntu",
	}
	for key, value := range want {
		if got[key] != value {
			t.Errorf("parseOSRelease()[%q] = %q, want %q", key, got[key], value)
		}
	}
}

func TestInfoFromOSReleaseMatchesChromiumDistro(t *testing.T) {
	longName := strings.Repeat("x", chromiumLinuxDistroMaxBytes+10)
	tests := []struct {
		name    string
		content string
		want    string
		ok      bool
	}{
		{
			name:    "pretty name",
			content: "NAME=Ubuntu\nVERSION_ID=24.04\nPRETTY_NAME=\" Ubuntu 24.04 LTS \"",
			want:    "Ubuntu 24.04 LTS",
			ok:      true,
		},
		{
			name:    "missing pretty name",
			content: "NAME=Ubuntu\nVERSION_ID=24.04",
			ok:      false,
		},
		{
			name:    "truncate to Chromium limit",
			content: "PRETTY_NAME=\"" + longName + "\"",
			want:    longName[:chromiumLinuxDistroMaxBytes],
			ok:      true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			info, ok := infoFromOSRelease(test.content)
			if ok != test.ok {
				t.Fatalf("infoFromOSRelease() ok = %v, want %v", ok, test.ok)
			}
			if info.Version != test.want {
				t.Fatalf("Version = %q, want %q", info.Version, test.want)
			}
		})
	}
}

func TestNormalizeVersionTriplet(t *testing.T) {
	tests := map[string]string{
		"":          "",
		"15":        "15.0.0",
		"15.1":      "15.1.0",
		"15.1.2":    "15.1.2",
		"15.1.beta": "15.1.beta",
	}
	for input, want := range tests {
		if got := normalizeVersionTriplet(input); got != want {
			t.Errorf("normalizeVersionTriplet(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestVersion(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{
			name: "macOS uses product version",
			info: Info{Name: "macOS", Version: "26.5.2", DisplayName: "macOS 26.5.2"},
			want: "26.5.2",
		},
		{
			name: "Windows uses numeric version",
			info: Info{Name: "Windows", Version: "10.0.26100", DisplayName: "Windows 10.0.26100"},
			want: "10.0.26100",
		},
		{
			name: "Linux uses PRETTY_NAME",
			info: Info{
				Name:        "Ubuntu",
				Version:     "Ubuntu 24.04.2 LTS",
				DisplayName: "Ubuntu 24.04.2 LTS",
			},
			want: "Ubuntu 24.04.2 LTS",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := test.info.Version; got != test.want {
				t.Fatalf("Version = %q, want %q", got, test.want)
			}
		})
	}
}
