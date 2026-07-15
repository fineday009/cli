// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import (
	"strconv"
	"strings"
)

// chromiumLinuxDistroMaxBytes is the maximum Linux distribution name length
// used by Chromium's operating-system information collector.
const chromiumLinuxDistroMaxBytes = 128

// parseOSRelease parses key-value entries from os-release content.
func parseOSRelease(content string) map[string]string {
	values := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		values[key] = unquoteOSReleaseValue(strings.TrimSpace(value))
	}
	return values
}

// unquoteOSReleaseValue removes supported quotes from an os-release value.
func unquoteOSReleaseValue(value string) string {
	if len(value) < 2 {
		return value
	}
	if value[0] == '"' && value[len(value)-1] == '"' {
		if unquoted, err := strconv.Unquote(value); err == nil {
			return unquoted
		}
	}
	if value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

// firstNonEmpty returns the first non-empty value, or an empty string if none exists.
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

// infoFromOSRelease converts valid os-release content into operating-system information.
func infoFromOSRelease(content string) (Info, bool) {
	values := parseOSRelease(content)
	prettyName := strings.Trim(values["PRETTY_NAME"], " \t\r\n\v\f")
	if prettyName == "" {
		return Info{}, false
	}
	if len(prettyName) > chromiumLinuxDistroMaxBytes {
		prettyName = prettyName[:chromiumLinuxDistroMaxBytes]
	}

	return Info{
		Name:        firstNonEmpty(values["NAME"], values["ID"], "Linux"),
		Version:     prettyName,
		Build:       values["BUILD_ID"],
		DisplayName: prettyName,
	}, true
}
