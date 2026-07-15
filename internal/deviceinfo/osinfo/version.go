// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package osinfo

import (
	"fmt"
	"strconv"
	"strings"
)

// normalizeVersionTriplet converts a numeric version to major.minor.patch form.
func normalizeVersionTriplet(version string) string {
	if version == "" {
		return ""
	}

	parts := strings.Split(version, ".")
	numbers := [3]int{}
	for i := 0; i < len(parts) && i < len(numbers); i++ {
		number, err := strconv.Atoi(parts[i])
		if err != nil {
			return version
		}
		numbers[i] = number
	}
	return fmt.Sprintf("%d.%d.%d", numbers[0], numbers[1], numbers[2])
}
