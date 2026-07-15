//go:build darwin

// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import (
	"os/exec"
	"strings"

	"golang.org/x/sys/unix"
)

// read collects the macOS product version, build, and display name.
func read() Info {
	version := normalizeVersionTriplet(
		sysctlOrSWVers("kern.osproductversion", "-productVersion"),
	)
	build := sysctlOrSWVers("kern.osversion", "-buildVersion")

	displayName := "macOS"
	if version != "" {
		displayName += " " + version
	}
	return Info{
		Name:        "macOS",
		Version:     version,
		Build:       build,
		DisplayName: displayName,
	}
}

// sysctlOrSWVers reads a system value with sysctl and falls back to sw_vers.
func sysctlOrSWVers(sysctlName, swVersFlag string) string {
	if value, err := unix.Sysctl(sysctlName); err == nil && value != "" {
		return value
	}

	output, err := exec.Command("/usr/bin/sw_vers", swVersFlag).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}
