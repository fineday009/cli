// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package deviceinfo

import (
	"fmt"
	"runtime"
	"strings"
	"sync"

	"github.com/larksuite/cli/internal/deviceinfo/osinfo"
)

const (
	userAgentSurfaceTerminal    = "terminal"
	userAgentSurfaceNonTerminal = "non-terminal"
)

var cachedTerminalUserAgent = sync.OnceValue(func() string {
	return buildEnhancedUserAgent(currentPlatform(), runtime.GOARCH, userAgentSurfaceTerminal)
})

var cachedNonTerminalUserAgent = sync.OnceValue(func() string {
	return buildEnhancedUserAgent(currentPlatform(), runtime.GOARCH, userAgentSurfaceNonTerminal)
})

// BuildUserAgent returns the device suffix appended to lark-cli's product
// User-Agent. Its stable shape is: (<platform>; <arch>; <surface>).
func BuildUserAgent(isTTY bool) string {
	if isTTY {
		return cachedTerminalUserAgent()
	}
	return cachedNonTerminalUserAgent()
}

func buildEnhancedUserAgent(platform, arch, surface string) string {
	return fmt.Sprintf("(%s; %s; %s)", platform, arch, surface)
}

func currentPlatform() string {
	info := osinfo.Get()
	for _, value := range [...]string{info.DisplayName, info.Name, runtime.GOOS} {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return "unknown"
}
