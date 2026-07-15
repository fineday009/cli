//go:build windows

// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import (
	"fmt"
	"strconv"

	"golang.org/x/sys/windows"
)

// read collects the Windows kernel version and derives its display fields.
func read() Info {
	major, minor, build := windows.RtlGetNtVersionNumbers()
	version := fmt.Sprintf("%d.%d.%d", major, minor, build)
	return Info{
		Name:        "Windows",
		Version:     version,
		Build:       strconv.FormatUint(uint64(build), 10),
		DisplayName: "Windows " + version,
	}
}
