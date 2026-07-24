// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"strings"
)

var appListSubTypes = []string{"standard", "grouped", "collapsible", "card", "detail"}

func normalizeAppListSubType(raw string) (string, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "standard", true
	}
	for _, candidate := range appListSubTypes {
		if value == candidate {
			return candidate, true
		}
	}
	return value, false
}

// validateAppListDataConfig validates only the published protocol shape. It
// intentionally does not infer UI semantics such as title/group_by cardinality
// or field roles.
func validateAppListDataConfig(subType string, cfg map[string]interface{}) []string {
	var problems []string
	allowed := map[string]bool{
		"base_token": true, "table_name": true, "filter": true, "sort_by": true,
	}
	switch subType {
	case "standard", "grouped", "collapsible":
		allowed["columns"] = true
		allowed["group_by"] = true
	case "card":
		allowed["fields"] = true
		allowed["card_config"] = true
	case "detail":
		allowed["fields"] = true
		allowed["detail_config"] = true
	}
	for key := range cfg {
		if !allowed[key] {
			problems = append(problems, fmt.Sprintf("%s 列表不支持字段 %s", subType, key))
		}
	}
	if token, _ := cfg["base_token"].(string); strings.TrimSpace(token) == "" {
		problems = append(problems, "缺少必填字段 base_token")
	}
	if tableName, _ := cfg["table_name"].(string); strings.TrimSpace(tableName) == "" {
		problems = append(problems, "缺少必填字段 table_name")
	}
	for _, key := range []string{"columns", "fields", "group_by", "sort_by"} {
		if raw, ok := cfg[key]; ok {
			if _, isArray := raw.([]interface{}); !isArray {
				problems = append(problems, key+" 必须是数组")
			}
		}
	}
	for _, key := range []string{"filter", "card_config", "detail_config"} {
		if raw, ok := cfg[key]; ok {
			if _, isObject := raw.(map[string]interface{}); !isObject {
				problems = append(problems, key+" 必须是对象")
			}
		}
	}
	return problems
}
