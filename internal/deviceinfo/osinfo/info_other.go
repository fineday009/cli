//go:build !darwin && !windows && !linux

// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import "runtime"

// read returns the Go operating-system identifier on unsupported platforms.
func read() Info {
	return Info{Name: runtime.GOOS, DisplayName: runtime.GOOS}
}
