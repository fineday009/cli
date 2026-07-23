// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"fmt"
	"strings"

	"github.com/larksuite/cli/errs"
)

// ── Public block types ───────────────────────────────────────────────

// chartBlockTypes are the chart and statistics block types shared by dashboard
// blocks and BaseApp page blocks.
var chartBlockTypes = []string{
	"column", "bar", "line", "pie", "ring", "scatter",
	"funnel", "wordCloud", "area", "combo", "radar", "statistics",
}

// textBlockTypes are the text-ish block types. "text" is the dashboard
// spelling, "richText" is the BaseApp page spelling; both carry the same
// data_config shape ({"text": "..."}).
var textBlockTypes = []string{"text", "richText"}

func matchesBlockType(blockType string, candidates []string) bool {
	trimmed := strings.ToLower(strings.TrimSpace(blockType))
	for _, candidate := range candidates {
		if trimmed == strings.ToLower(candidate) {
			return true
		}
	}
	return false
}

func isTextBlockType(blockType string) bool { return matchesBlockType(blockType, textBlockTypes) }

func isChartBlockType(blockType string) bool { return matchesBlockType(blockType, chartBlockTypes) }

// appBlockTypes are all block types accepted by the BaseApp page block
// commands, in the order they appear in the protocol design.
func appBlockTypes() []string {
	types := make([]string, 0, len(chartBlockTypes)+2)
	types = append(types, chartBlockTypes...)
	types = append(types, "richText")
	types = append(types, "list")
	return types
}

func isAppBlockType(blockType string) bool {
	return isChartBlockType(blockType) || matchesBlockType(blockType, []string{"richText", "list"})
}

// ── data_config normalization & validation ───────────────────────────

// normalizeDataConfig normalizes data_config fields for chart blocks.
// It converts series[].rollup to uppercase and group_by[].sort fields to lowercase.
func normalizeDataConfig(cfg map[string]interface{}) map[string]interface{} {
	if cfg == nil {
		return nil
	}
	out := cloneMap(cfg)
	// series[].rollup → 大写
	if arr, ok := out["series"].([]interface{}); ok {
		for i, it := range arr {
			if m, ok := it.(map[string]interface{}); ok {
				if r, ok := m["rollup"].(string); ok && r != "" {
					m["rollup"] = strings.ToUpper(strings.TrimSpace(r))
				}
				arr[i] = m
			}
		}
		out["series"] = arr
	}
	// group_by.sort 的 type/order → 小写
	if gb, ok := out["group_by"].([]interface{}); ok {
		for i, g := range gb {
			if m, ok := g.(map[string]interface{}); ok {
				if md, ok := m["mode"].(string); ok {
					m["mode"] = strings.ToLower(strings.TrimSpace(md))
				}
				if sub, ok := m["sort"].(map[string]interface{}); ok {
					sortType := ""
					if t, ok := sub["type"].(string); ok {
						sortType = strings.ToLower(strings.TrimSpace(t))
						sub["type"] = sortType
					}
					// Only lowercase a string order; leave a present-but-non-string
					// order untouched so validateBlockDataConfig can reject it
					// instead of it being silently coerced below.
					_, hasOrderKey := sub["order"]
					orderStr, orderIsString := sub["order"].(string)
					if orderIsString {
						sub["order"] = strings.ToLower(strings.TrimSpace(orderStr))
					}
					// Default only when the order key is truly absent. A present
					// key (even an illegal type/value) must survive to validation.
					if !hasOrderKey && (sortType == "group" || sortType == "view") {
						sub["order"] = "asc"
					}
					m["sort"] = sub
				}
				gb[i] = m
			}
		}
		out["group_by"] = gb
	}
	return out
}

// validateBlockDataConfig validates data_config based on block type.
// Text blocks only need a text field; everything else falls through to the
// dashboard chart rules. BaseApp list validation lives in
// app_list_block_data_config.go and never enters this dashboard path.
func validateBlockDataConfig(blockType string, cfg map[string]interface{}) []string {
	switch {
	case isTextBlockType(blockType):
		return validateTextDataConfig(blockType, cfg)
	default:
		return validateChartDataConfig(cfg)
	}
}

// validateTextDataConfig validates the text/richText data_config shape.
func validateTextDataConfig(blockType string, cfg map[string]interface{}) []string {
	var problems []string
	if txt, _ := cfg["text"].(string); strings.TrimSpace(txt) == "" {
		problems = append(problems, fmt.Sprintf("%s 类型组件缺少必填字段 text", strings.TrimSpace(blockType)))
	}
	return problems
}

// validateChartDataConfig validates the chart/statistics data_config shape.
func validateChartDataConfig(cfg map[string]interface{}) []string {
	var errs []string

	// 图表类型通用校验
	// table_name 必填
	if tn, _ := cfg["table_name"].(string); strings.TrimSpace(tn) == "" {
		errs = append(errs, "缺少必填字段 table_name")
	}
	// series 与 count_all 互斥且必有其一
	_, hasSeries := cfg["series"]
	_, hasCountAll := cfg["count_all"]
	if !(hasSeries || hasCountAll) {
		errs = append(errs, "series 与 count_all 二选一，至少提供其一")
	}
	if hasSeries && hasCountAll {
		errs = append(errs, "series 与 count_all 互斥，不可同时存在")
	}
	// series 校验
	if hasSeries {
		arr, ok := cfg["series"].([]interface{})
		if !ok || len(arr) == 0 {
			errs = append(errs, "series 必须是非空数组")
		} else {
			// rollup 支持：SUM / MAX / MIN / AVERAGE（不支持 COUNTA；计数请使用 count_all）
			allowed := map[string]bool{"SUM": true, "MAX": true, "MIN": true, "AVERAGE": true}
			for i, it := range arr {
				m, ok := it.(map[string]interface{})
				if !ok {
					errs = append(errs, fmt.Sprintf("series[%d] 必须是对象", i))
					continue
				}
				fn, _ := m["field_name"].(string)
				if strings.TrimSpace(fn) == "" {
					errs = append(errs, fmt.Sprintf("series[%d].field_name 不能为空", i))
				}
				r, _ := m["rollup"].(string)
				r = strings.ToUpper(strings.TrimSpace(r))
				if !allowed[r] {
					errs = append(errs, fmt.Sprintf("series[%d].rollup 不在允许枚举内: %s", i, r))
				}
			}
		}
	}
	// group_by 最多 2 个，字段名必填，sort 合法
	if gb, ok := cfg["group_by"].([]interface{}); ok {
		if len(gb) > 2 {
			errs = append(errs, "group_by 最多支持 2 个维度")
		}
		for i, g := range gb {
			m, ok := g.(map[string]interface{})
			if !ok {
				errs = append(errs, fmt.Sprintf("group_by[%d] 必须是对象", i))
				continue
			}
			fn, _ := m["field_name"].(string)
			if strings.TrimSpace(fn) == "" {
				errs = append(errs, fmt.Sprintf("group_by[%d].field_name 不能为空", i))
			}
			if sub, ok := m["sort"].(map[string]interface{}); ok {
				t, _ := sub["type"].(string)
				t = strings.ToLower(strings.TrimSpace(t))
				if t != "group" && t != "value" && t != "view" {
					errs = append(errs, fmt.Sprintf("group_by[%d].sort.type 仅支持 group|value|view", i))
				}
				orderRaw, hasOrder := sub["order"]
				o, orderIsString := orderRaw.(string)
				o = strings.ToLower(strings.TrimSpace(o))
				switch {
				case !hasOrder:
					errs = append(errs, fmt.Sprintf("group_by[%d].sort.order 缺失；sort 存在时必须设置 order 为 asc 或 desc，例如 \"sort\":{\"type\":\"group\",\"order\":\"asc\"}", i))
				case !orderIsString || (o != "asc" && o != "desc"):
					errs = append(errs, fmt.Sprintf("group_by[%d].sort.order 仅支持 asc|desc", i))
				}
			}
		}
	}
	// filter 基本结构
	errs = append(errs, validateBlockFilter(cfg, "filter", false)...)
	return errs
}

// validateBlockFilter validates the filter object shared by chart and list
// data_config. key is the config key holding the filter ("filter").
// allowFieldID lets list blocks reference a field by ID; chart blocks keep the
// dashboard rule of field_name only.
func validateBlockFilter(cfg map[string]interface{}, key string, allowFieldID bool) []string {
	f, ok := cfg[key].(map[string]interface{})
	if !ok {
		return nil
	}
	var problems []string
	conj := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", f["conjunction"])))
	if conj == "" {
		conj = "and"
	}
	if conj != "and" && conj != "or" {
		problems = append(problems, key+".conjunction 仅支持 and|or")
	}
	conds, ok := f["conditions"].([]interface{})
	if !ok {
		return problems
	}
	allowedOps := map[string]bool{"is": true, "isnot": true, "contains": true, "doesnotcontain": true, "isempty": true, "isnotempty": true, "isgreater": true, "isgreaterequal": true, "isless": true, "islessequal": true}
	for i, it := range conds {
		m, ok := it.(map[string]interface{})
		if !ok {
			problems = append(problems, fmt.Sprintf("%s.conditions[%d] 必须是对象", key, i))
			continue
		}
		fn, _ := m["field_name"].(string)
		hasRef := strings.TrimSpace(fn) != ""
		if !hasRef && allowFieldID {
			fid, _ := m["field_id"].(string)
			hasRef = strings.TrimSpace(fid) != ""
		}
		if !hasRef {
			problems = append(problems, fmt.Sprintf("%s.conditions[%d].field_name 不能为空", key, i))
		}
		op, _ := m["operator"].(string)
		opKey := strings.ToLower(strings.ReplaceAll(strings.TrimSpace(op), " ", ""))
		if !allowedOps[opKey] {
			problems = append(problems, fmt.Sprintf("%s.conditions[%d].operator 不支持: %s", key, i, op))
		}
		if opKey != "isempty" && opKey != "isnotempty" {
			if _, has := m["value"]; !has {
				problems = append(problems, fmt.Sprintf("%s.conditions[%d].value 缺失", key, i))
			}
		}
	}
	return problems
}

func formatDataConfigErrors(problems []string) error {
	if len(problems) == 0 {
		return nil
	}
	return errs.NewValidationError(errs.SubtypeInvalidArgument, "data_config 校验失败:\n- %s\n参考: skills/lark-base/references/dashboard-block-data-config.md（应用页面组件见 lark-base-baseapp-block-data-config.md）", strings.Join(problems, "\n- ")).WithParam("--data-config")
}
