//go:build linux

// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import "github.com/larksuite/cli/internal/vfs"

// read collects Linux identity information from the first valid os-release file.
func read() Info {
	for _, path := range [...]string{"/etc/os-release", "/usr/lib/os-release"} {
		content, err := vfs.ReadFile(path)
		if err != nil {
			continue
		}

		if info, ok := infoFromOSRelease(string(content)); ok {
			return info
		}
	}
	return Info{
		Name:        "Linux",
		Version:     "Unknown",
		DisplayName: "Linux",
	}
}
