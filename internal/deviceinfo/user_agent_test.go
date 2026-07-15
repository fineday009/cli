// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package deviceinfo

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuildEnhancedUserAgent(t *testing.T) {
	got := buildEnhancedUserAgent("macOS 26.0.4", "arm64", "terminal")
	want := "(macOS 26.0.4; arm64; terminal)"
	if got != want {
		t.Fatalf("enhanced user agent = %q, want %q", got, want)
	}
}

func TestBuildUserAgentUsesCurrentPlatformAndTTYSurface(t *testing.T) {
	tests := []struct {
		name    string
		isTTY   bool
		surface string
	}{
		{name: "terminal", isTTY: true, surface: userAgentSurfaceTerminal},
		{name: "non terminal", isTTY: false, surface: userAgentSurfaceNonTerminal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildUserAgent(tt.isTTY)
			if !strings.HasPrefix(got, "(") || !strings.HasSuffix(got, ")") {
				t.Fatalf("BuildUserAgent(%t) = %q, want parenthesized device suffix", tt.isTTY, got)
			}
			if !strings.Contains(got, "; "+runtime.GOARCH+"; ") {
				t.Fatalf("BuildUserAgent(%t) = %q, want architecture %q", tt.isTTY, got, runtime.GOARCH)
			}
			if !strings.HasSuffix(got, "; "+tt.surface+")") {
				t.Fatalf("BuildUserAgent(%t) = %q, want surface %q", tt.isTTY, got, tt.surface)
			}
			if strings.Contains(got, "Mozilla/") || strings.Contains(got, "Chrome/") {
				t.Fatalf("BuildUserAgent(%t) = %q, Chromium-style tokens must be omitted", tt.isTTY, got)
			}
		})
	}
}
