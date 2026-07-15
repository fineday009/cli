// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

// Package deviceinfo collects the platform hardware product model and the
// platform values used by device-related risk-control headers.
package deviceinfo

import (
	"runtime"
	"strings"
	"sync"
	"unicode"
	"unicode/utf8"

	"golang.org/x/net/http/httpguts"
)

const (
	// TerminalTypePC is the fixed X-Agent-Terminal-Type value for the CLI.
	TerminalTypePC = "1"

	// Unknown is used when the hardware product model cannot be collected.
	Unknown = "Unknown"

	// deviceModelMaxBytes bounds the value added to X-Agent-Device-Type.
	// Device models are short identifiers; a larger value is treated as
	// malformed rather than truncated so the header never misrepresents it.
	deviceModelMaxBytes = 256
)

// OS type enum values for X-Agent-Os-Type.
const (
	OSTypeUnknown = "0"
	OSTypeWindows = "1"
	OSTypeLinux   = "2"
	OSTypeMacOS   = "3"
)

// collector reads the device model at most once and caches the result safely
// across goroutines. An empty value is also final for the process lifetime so
// restricted environments do not repeat failed platform probes per request.
type collector struct {
	once  sync.Once
	value string
	read  func() string
}

// defaultCollector is the process-wide collector used by Get.
var defaultCollector = &collector{read: readDeviceModel}

// Get returns the cached platform hardware product model. Collection failures
// degrade to Unknown and remain cached for the process lifetime.
func Get() string {
	if model := defaultCollector.get(); model != "" {
		return model
	}
	return Unknown
}

// get returns the cached device model, reading it at most once.
func (c *collector) get() string {
	c.once.Do(func() {
		c.value = normalizeModel(c.read())
	})
	return c.value
}

// normalizeModel removes non-printable characters and returns a model only
// when the remaining text is safe to use as an HTTP header value. Input that
// cannot produce a valid model is rejected so Get can fall back to Unknown.
func normalizeModel(model string) string {
	if !utf8.ValidString(model) {
		return ""
	}
	model = strings.Map(func(r rune) rune {
		switch {
		case r == '\r' || r == '\n' || r == '\x00':
			return -1
		case unicode.IsSpace(r):
			return ' '
		case unicode.IsPrint(r):
			return r
		default:
			return -1
		}
	}, model)

	model = strings.Join(strings.Fields(model), " ")

	if model == "" || len(model) > deviceModelMaxBytes {
		return ""
	}
	if !httpguts.ValidHeaderFieldValue(model) {
		return ""
	}
	return model
}

// GetOSType maps a platform name to the X-Agent-Os-Type enum.
func GetOSType(osName string) string {
	switch osName {
	case "Windows":
		return OSTypeWindows
	case "Linux":
		return OSTypeLinux
	case "MacOS":
		return OSTypeMacOS
	default:
		return OSTypeUnknown
	}
}

// OSName returns the platform name used by GetOSType.
func OSName() string {
	switch runtime.GOOS {
	case "darwin":
		return "MacOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	default:
		return runtime.GOOS
	}
}
