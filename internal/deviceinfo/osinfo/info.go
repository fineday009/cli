// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package osinfo collects operating-system identity and version information.
package osinfo

// Info describes the operating system independently from the hardware model.
type Info struct {
	// Name is the operating-system family, such as macOS, Windows, or Ubuntu.
	Name string
	// Version is a numeric product version on macOS/Windows and PRETTY_NAME on
	// Linux.
	Version string
	// Build is the platform build identifier when one is available.
	Build string
	// DisplayName is the human-readable release name. On Linux this maps to
	// PRETTY_NAME from os-release, matching Chromium's GetLinuxDistro source.
	DisplayName string
}

// Get returns the current operating-system information.
func Get() Info {
	return read()
}
