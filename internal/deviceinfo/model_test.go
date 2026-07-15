// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package deviceinfo

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"unicode"
)

func TestCollectorCachesNonEmptyModel(t *testing.T) {
	calls := 0
	c := collector{read: func() string {
		calls++
		return "  MacBookPro18,3\n"
	}}

	if got := c.get(); got != "MacBookPro18,3" {
		t.Fatalf("first get = %q, want %q", got, "MacBookPro18,3")
	}
	if got := c.get(); got != "MacBookPro18,3" {
		t.Fatalf("second get = %q, want cached model", got)
	}
	if calls != 1 {
		t.Fatalf("read called %d times, want 1", calls)
	}
}

func TestCollectorCachesEmptyModel(t *testing.T) {
	calls := 0
	c := collector{read: func() string {
		calls++
		return ""
	}}

	if got := c.get(); got != "" {
		t.Fatalf("first get = %q, want empty", got)
	}
	if got := c.get(); got != "" {
		t.Fatalf("second get = %q, want cached empty result", got)
	}
	if calls != 1 {
		t.Fatalf("read called %d times, want 1", calls)
	}
}

func TestCollectorReadsOnceAcrossConcurrentCalls(t *testing.T) {
	var calls atomic.Int32
	c := collector{read: func() string {
		calls.Add(1)
		return "ThinkPad X1 Carbon"
	}}

	const goroutines = 32
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			if got := c.get(); got != "ThinkPad X1 Carbon" {
				t.Errorf("get = %q, want %q", got, "ThinkPad X1 Carbon")
			}
		}()
	}
	wg.Wait()

	if got := calls.Load(); got != 1 {
		t.Fatalf("read called %d times, want 1", got)
	}
}

func TestGetFallsBackToUnknown(t *testing.T) {
	calls := 0
	original := defaultCollector
	defaultCollector = &collector{read: func() string {
		calls++
		return ""
	}}
	t.Cleanup(func() { defaultCollector = original })

	if got := Get(); got != Unknown {
		t.Fatalf("Get() = %q, want %q", got, Unknown)
	}
	if got := Get(); got != Unknown {
		t.Fatalf("second Get() = %q, want %q", got, Unknown)
	}
	if calls != 1 {
		t.Fatalf("read called %d times, want 1", calls)
	}
}

func TestNormalizeModel(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  string
	}{
		{name: "trims surrounding whitespace", model: "  MacBookPro18,3\n", want: "MacBookPro18,3"},
		{name: "trims device tree terminator", model: "Raspberry Pi 5\x00", want: "Raspberry Pi 5"},
		{name: "allows printable Unicode", model: "联想 ThinkPad X1", want: "联想 ThinkPad X1"},
		{name: "rejects empty", model: " \t\r\n"},
		{name: "rejects invalid UTF-8", model: string([]byte{'M', 0xff, '1'})},
		{name: "removes CRLF", model: "model\r\nname", want: "modelname"},
		{name: "normalizes tab", model: "model\tname", want: "model name"},
		{name: "removes NUL", model: "model\x00name", want: "modelname"},
		{name: "removes control character", model: "model\x1fname", want: "modelname"},
		{name: "removes DEL", model: "model\x7fname", want: "modelname"},
		{name: "normalizes Unicode line separator", model: "model\u2028name", want: "model name"},
		{name: "collapses whitespace", model: "  model\t \u00a0 name  ", want: "model name"},
		{name: "accepts maximum byte length", model: strings.Repeat("a", deviceModelMaxBytes), want: strings.Repeat("a", deviceModelMaxBytes)},
		{name: "rejects overlong value", model: strings.Repeat("a", deviceModelMaxBytes+1)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeModel(tt.model); got != tt.want {
				t.Fatalf("normalizeModel(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestNormalizeModelRemovesHTTPControlBytes(t *testing.T) {
	for value := 0; value <= 0x7f; value++ {
		if value >= 0x20 && value < 0x7f {
			continue
		}
		t.Run(fmt.Sprintf("0x%02x", value), func(t *testing.T) {
			model := "model" + string(rune(value)) + "name"
			want := "modelname"
			if value != '\r' && value != '\n' && value != '\x00' && unicode.IsSpace(rune(value)) {
				want = "model name"
			}
			if got := normalizeModel(model); got != want {
				t.Fatalf("normalizeModel(%q) = %q, want %q", model, got, want)
			}
		})
	}
}

func TestGetFallsBackToUnknownWhenSanitizedModelIsEmpty(t *testing.T) {
	original := defaultCollector
	defaultCollector = &collector{read: func() string { return "\x00\x1f\x7f" }}
	t.Cleanup(func() { defaultCollector = original })

	if got := Get(); got != Unknown {
		t.Fatalf("Get() = %q, want %q", got, Unknown)
	}
}

func TestGetOSType(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{name: "Windows", want: OSTypeWindows},
		{name: "Linux", want: OSTypeLinux},
		{name: "MacOS", want: OSTypeMacOS},
		{name: "unknown", want: OSTypeUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOSType(tt.name); got != tt.want {
				t.Errorf("GetOSType(%q) = %q, want %q", tt.name, got, tt.want)
			}
		})
	}
}
