// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package sheets

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/suggest"
)

// ─── +batch-update sub-op dispatch ─────────────────────────────────────
//
// 用户传给 +batch-update --operations 的形态是 CLI 视角的 {shortcut, input}：
//
//     [{"shortcut": "+range-copy", "input": {"sheet_id":"...","source-range":"A1:B2","target-range":"A10"}}, ...]
//
// input 里用的是该 shortcut 的 **CLI flag 名**（与 standalone 调用一致；连字符 /
// 下划线两种写法都接受）。底层 MCP batch_update tool 要的是
// {tool_name, input(MCP body)} —— body 的字段名往往与 CLI flag 名不同
// （如 +range-copy 的 source-range/target-range 要翻成 range/destination_range）。
//
// 关键：每个子操作复用 **standalone shortcut 同一套 flag→body translator**
// （那些 *Input 构建函数，现在统一接收 flagView 接口）。这样 batch 子操作
// 产出的 MCP body 与该 shortcut 单独调用产出的 body 完全一致（由
// batch-vs-standalone 契约测试保证）。dispatch 表只列**可纳入 atomic batch
// 的 write shortcut**——读操作、fan-out wrapper（+batch-update 自身、
// +cells-batch-set-style、+cells-batch-clear、+dropdown-{update,delete}）一律不放进表里，
// 用户传到 +batch-update 里会被 translator 拒绝。

// batchTranslateFn turns a sub-op's CLI-shape input (via flagView) into the MCP
// tool body for the underlying batch_update sub-tool. token is the
// +batch-update top-level spreadsheet token; sheetID/sheetName are the resolved
// sheet selector for this sub-op. The returned body already carries excel_id
// and (where the tool needs one) the operation discriminator — exactly as the
// standalone shortcut would emit.
type batchTranslateFn func(fv flagView, token, sheetID, sheetName string) (map[string]interface{}, error)

type batchOpMapping struct {
	// mcpToolName 是底层 MCP batch_update 接受的 tool_name。
	mcpToolName string
	// translate 复用 standalone 的 *Input 构建逻辑，产出 MCP body。
	translate batchTranslateFn
}

// sheetSelectorFlagsForSubOp returns the (id, name) flag names a +batch-update
// sub-op uses to express its placement / context sheet. Defaults are
// `sheet-id` / `sheet-name`; +pivot-create deviates because its create
// shortcut renamed the placement selector to `target-sheet-id` /
// `target-sheet-name` (the data-source sheet is encoded in --source as
// `'SheetName'!Range`, not in a sheet selector flag). Update / delete on
// pivot still use the default names — only the create create-side
// shortcut was renamed.
func sheetSelectorFlagsForSubOp(shortcut string) (string, string) {
	if shortcut == "+pivot-create" {
		return "target-sheet-id", "target-sheet-name"
	}
	return "sheet-id", "sheet-name"
}

// objCreateTranslate / objUpdateTranslate / objDeleteTranslate bind an object
// CRUD spec to the shared object_crud builders.
func objCreateTranslate(spec objectCRUDSpec) batchTranslateFn {
	return func(fv flagView, token, sheetID, sheetName string) (map[string]interface{}, error) {
		return objectCreateInput(fv, token, sheetID, sheetName, spec)
	}
}

func objUpdateTranslate(spec objectCRUDSpec) batchTranslateFn {
	return func(fv flagView, token, sheetID, sheetName string) (map[string]interface{}, error) {
		return objectUpdateInput(fv, token, sheetID, sheetName, spec)
	}
}

func objDeleteTranslate(spec objectCRUDSpec) batchTranslateFn {
	return func(fv flagView, token, sheetID, sheetName string) (map[string]interface{}, error) {
		return objectDeleteInput(fv, token, sheetID, sheetName, spec)
	}
}

// batchOpDispatch covers every write shortcut that can join an atomic batch.
// Each entry plugs the shortcut's standalone xxxInput builder into the
// batch translator path — so the body is byte-identical to the standalone
// invocation (locked by TestBatchOp_BodyMatchesStandalone) and the missing-
// flag error is identical too (locked by TestBatchOp_ErrorEquivalence).
var batchOpDispatch = map[string]batchOpMapping{
	// ─── 单元格内容 ──────────────────────────────────────────────────
	"+cells-set": {"set_cell_range", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		// The --writes plural form expands into its own atomic batch and
		// cannot nest; sub-ops carry one range+cells each.
		if fv.Changed("writes") {
			return nil, sheetsValidationForFlag("writes", `"writes" is not supported inside +batch-update (it expands into its own atomic batch); call +cells-set --writes standalone, or give each sub-op a single range + cells`)
		}
		return cellsSetInput(fv, token, sid, sname)
	}},
	"+cells-set-style": {"set_cell_range", cellsSetStyleInput},
	"+cells-clear":     {"clear_cell_range", cellsClearInput},
	"+cells-replace":   {"replace_data", replaceInput},
	"+csv-put":         {"set_range_from_csv", csvPutInput},
	"+dropdown-set":    {"set_cell_range", dropdownSetInput},

	// ─── 单元格合并 (merge_cells, operation 区分) ────────────────────
	"+cells-merge": {"merge_cells", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return mergeInput(fv, token, sid, sname, "merge", true)
	}},
	"+cells-unmerge": {"merge_cells", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return mergeInput(fv, token, sid, sname, "unmerge", false)
	}},

	// ─── 行列结构 (modify_sheet_structure, operation 区分) ──────────
	"+dim-insert": {"modify_sheet_structure", dimInsertInput},
	"+dim-delete": {"modify_sheet_structure", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		// The --ranges plural form expands into its own atomic batch and
		// cannot nest; sub-ops carry one range each.
		if fv.Changed("ranges") {
			return nil, sheetsValidationForFlag("ranges", `"ranges" is not supported inside +batch-update (it expands into its own atomic batch); call +dim-delete --ranges standalone, or give each sub-op a single "range"`)
		}
		return dimRangeOpInput(fv, token, sid, sname, "delete")
	}},
	"+dim-hide": {"modify_sheet_structure", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return dimRangeOpInput(fv, token, sid, sname, "hide")
	}},
	"+dim-unhide": {"modify_sheet_structure", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return dimRangeOpInput(fv, token, sid, sname, "unhide")
	}},
	"+dim-freeze": {"modify_sheet_structure", dimFreezeInput},
	"+dim-group": {"modify_sheet_structure", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return dimGroupInput(fv, token, sid, sname, "group")
	}},
	"+dim-ungroup": {"modify_sheet_structure", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return dimGroupInput(fv, token, sid, sname, "ungroup")
	}},

	// ─── 行高列宽 (resize_range, 无 operation 字段) ─────────────────
	// The map form (--heights/--widths) fans out into its own batch_update
	// and cannot nest inside +batch-update; sub-ops must use the uniform
	// single-range form (range + height/width or type).
	"+rows-resize": {"resize_range", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		if err := rejectResizeMapInBatch(fv, "row"); err != nil {
			return nil, err
		}
		return resizeInput(fv, token, sid, sname, "row")
	}},
	"+cols-resize": {"resize_range", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		if err := rejectResizeMapInBatch(fv, "column"); err != nil {
			return nil, err
		}
		return resizeInput(fv, token, sid, sname, "column")
	}},

	// ─── 区域操作 (transform_range, operation 区分) ─────────────────
	"+range-move": {"transform_range", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return transformMoveCopyInput(fv, token, sid, sname, "move", false)
	}},
	"+range-copy": {"transform_range", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		return transformMoveCopyInput(fv, token, sid, sname, "copy", true)
	}},
	"+range-fill": {"transform_range", rangeFillInput},
	"+range-sort": {"transform_range", rangeSortInput},

	// ─── 工作簿 / 子表 (modify_workbook_structure, operation 区分) ──
	"+sheet-create": {"modify_workbook_structure", func(fv flagView, token, _, _ string) (map[string]interface{}, error) {
		return sheetCreateInput(fv, token)
	}},
	"+sheet-delete": {"modify_workbook_structure", sheetDeleteInput},
	"+sheet-rename": {"modify_workbook_structure", sheetRenameInput},
	"+sheet-move":   {"modify_workbook_structure", sheetMoveBatchInput},
	"+sheet-copy":   {"modify_workbook_structure", sheetCopyInput},
	"+sheet-hide": {"modify_workbook_structure", func(fv flagView, t, sid, sn string) (map[string]interface{}, error) {
		return sheetVisibilityInput(fv, t, sid, sn, "hide")
	}},
	"+sheet-unhide": {"modify_workbook_structure", func(fv flagView, t, sid, sn string) (map[string]interface{}, error) {
		return sheetVisibilityInput(fv, t, sid, sn, "unhide")
	}},
	"+sheet-set-tab-color": {"modify_workbook_structure", sheetSetTabColorInput},
	"+sheet-show-gridline": {"modify_workbook_structure", func(fv flagView, t, sid, sn string) (map[string]interface{}, error) {
		return sheetVisibilityInput(fv, t, sid, sn, "show_gridline")
	}},
	"+sheet-hide-gridline": {"modify_workbook_structure", func(fv flagView, t, sid, sn string) (map[string]interface{}, error) {
		return sheetVisibilityInput(fv, t, sid, sn, "hide_gridline")
	}},

	// ─── 对象族 CRUD (manage_*_object, operation 区分) ─────────────
	"+chart-create": {"manage_chart_object", objCreateTranslate(chartSpec)},
	"+chart-update": {"manage_chart_object", objUpdateTranslate(chartSpec)},
	"+chart-delete": {"manage_chart_object", objDeleteTranslate(chartSpec)},

	"+pivot-create": {"manage_pivot_table_object", objCreateTranslate(pivotSpec)},
	"+pivot-update": {"manage_pivot_table_object", objUpdateTranslate(pivotSpec)},
	"+pivot-delete": {"manage_pivot_table_object", objDeleteTranslate(pivotSpec)},

	"+cond-format-create": {"manage_conditional_format_object", objCreateTranslate(condFormatSpec)},
	"+cond-format-update": {"manage_conditional_format_object", objUpdateTranslate(condFormatSpec)},
	"+cond-format-delete": {"manage_conditional_format_object", objDeleteTranslate(condFormatSpec)},

	"+filter-create": {"manage_filter_object", filterCreateInput},
	"+filter-update": {"manage_filter_object", filterUpdateInput},
	"+filter-delete": {"manage_filter_object", filterDeleteInput},

	"+filter-view-create": {"manage_filter_view_object", objCreateTranslate(filterViewSpec)},
	"+filter-view-update": {"manage_filter_view_object", objUpdateTranslate(filterViewSpec)},
	"+filter-view-delete": {"manage_filter_view_object", objDeleteTranslate(filterViewSpec)},

	"+sparkline-create": {"manage_sparkline_object", objCreateTranslate(sparklineSpec)},
	"+sparkline-update": {"manage_sparkline_object", objUpdateTranslate(sparklineSpec)},
	"+sparkline-delete": {"manage_sparkline_object", objDeleteTranslate(sparklineSpec)},

	"+float-image-create": {"manage_float_image_object", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		if err := rejectLocalImageInBatch(fv); err != nil {
			return nil, err
		}
		return floatImageWriteInput(fv, token, sid, sname, "create", false, "")
	}},
	"+float-image-update": {"manage_float_image_object", func(fv flagView, token, sid, sname string) (map[string]interface{}, error) {
		if err := rejectLocalImageInBatch(fv); err != nil {
			return nil, err
		}
		return floatImageWriteInput(fv, token, sid, sname, "update", true, "")
	}},
	"+float-image-delete": {"manage_float_image_object", objDeleteTranslate(floatImageDeleteSpec)},
}

// allowedBatchShortcuts lists every shortcut accepted inside +batch-update,
// sorted, for the not-allowed error hint.
func allowedBatchShortcuts() []string {
	out := make([]string, 0, len(batchOpDispatch))
	for sc := range batchOpDispatch {
		out = append(out, sc)
	}
	sort.Strings(out)
	return out
}

// subOpInputContract renders one shortcut's complete sub-op key vocabulary
// (wire-style underscore names) for the translator-failure hint: required
// flags are marked, the sheet selector pair collapses to a choose-one, and
// spreadsheet locators are omitted (reserved for the batch top level).
// Returns "" for shortcuts without a flag-defs entry.
func subOpInputContract(sc string) string {
	defs, _ := loadFlagDefs()
	spec, ok := defs[sc]
	if !ok {
		return ""
	}
	idFlag, nameFlag := sheetSelectorFlagsForSubOp(sc)
	var keys []string
	sheetSelector := ""
	for _, df := range spec.Flags {
		if df.Kind == "system" || df.Hidden {
			continue
		}
		switch df.Name {
		case "url", "spreadsheet-token":
			continue // reserved: supplied by +batch-update top level
		case idFlag, nameFlag:
			sheetSelector = strings.ReplaceAll(idFlag, "-", "_") + "|" + strings.ReplaceAll(nameFlag, "-", "_") + " (choose one)"
			continue
		}
		key := strings.ReplaceAll(df.Name, "-", "_")
		if df.Required == "required" {
			key += " (required)"
		}
		keys = append(keys, key)
	}
	if sheetSelector != "" {
		keys = append([]string{sheetSelector}, keys...)
	}
	return strings.Join(keys, ", ")
}

// rejectLocalImageInBatch blocks the local-file --image source inside
// +batch-update: a batch sub-op has no upload phase, so the file could not be
// turned into a file_token. Callers must pass --image-token / --image-uri.
func rejectLocalImageInBatch(fv flagView) error {
	if strings.TrimSpace(fv.Str("image")) != "" {
		return sheetsValidationForFlag("image", "--image (local upload) is not supported inside +batch-update; pass --image-token or --image-uri instead")
	}
	return nil
}

// sheetMoveBatchInput translates +sheet-move inside a batch. Unlike the
// standalone shortcut it cannot issue the get_workbook_structure read that
// auto-derives sheet_id / source_index, so both must be supplied explicitly.
func sheetMoveBatchInput(fv flagView, token, sheetID, sheetName string) (map[string]interface{}, error) {
	if sheetID == "" {
		return nil, sheetsValidationForFlag("sheet-id", "+sheet-move in +batch-update requires sheet_id (sheet_name needs a network lookup unavailable mid-batch)")
	}
	if !fv.Changed("source-index") {
		return nil, sheetsValidationForFlag("source-index", "+sheet-move in +batch-update requires source_index (auto-derive needs a network lookup unavailable mid-batch)")
	}
	if fv.Int("source-index") < 0 {
		return nil, sheetsValidationForFlag("source-index", "--source-index must be >= 0")
	}
	// Standalone +sheet-move requires --index (see SheetMove.Validate). A batch
	// sub-op skips that path, and mapFlagView falls back to the flag default (0),
	// which would silently move the sheet to the front. Require it explicitly so
	// the batch contract matches the standalone one.
	if !fv.Changed("index") {
		return nil, sheetsValidationForFlag("index", "+sheet-move in +batch-update requires index")
	}
	if fv.Int("index") < 0 {
		return nil, sheetsValidationForFlag("index", "--index must be >= 0")
	}
	return map[string]interface{}{
		"excel_id":     token,
		"operation":    "move",
		"sheet_id":     sheetID,
		"source_index": fv.Int("source-index"),
		"target_index": fv.Int("index"),
	}, nil
}

// reservedSubOpKeys 是禁止用户在 sub-op input 里手填的 key —— 它们由
// +batch-update 顶层 --url/--token 统一提供（excel_id / spreadsheet_token / url）。
var reservedSubOpKeys = []string{"excel_id", "spreadsheet_token", "url"}

// subOpKeyVocabulary returns the set of hyphen-canonical flag names a sub-op
// input may carry for `sc`: every non-system flag in flag-defs except the
// spreadsheet locators (reserved for the batch top level). Nil when the
// shortcut has no flag-defs entry (vocabulary checks are then skipped).
func subOpKeyVocabulary(sc string) map[string]bool {
	defs, _ := loadFlagDefs()
	spec, ok := defs[sc]
	if !ok {
		return nil
	}
	vocab := make(map[string]bool, len(spec.Flags))
	for _, df := range spec.Flags {
		if df.Kind == "system" || df.Name == "url" || df.Name == "spreadsheet-token" {
			continue
		}
		vocab[df.Name] = true
	}
	return vocab
}

// camelToKebab converts a lowerCamelCase key to its kebab form
// (sheetName → sheet-name). Returns "" when the key carries no uppercase
// letter (nothing to convert).
func camelToKebab(key string) string {
	if strings.ToLower(key) == key {
		return ""
	}
	var b strings.Builder
	for i, r := range key {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('-')
			}
			b.WriteRune(r + ('a' - 'A'))
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

// normalizeSubOpInputKeys validates every sub-op input key against the
// shortcut's flag vocabulary, rewriting habitual spellings in place and
// rejecting anything that matches nothing. Eval traces show unknown keys were
// previously ignored silently, which turned "wrong key" (size for width,
// camelCase sheetName, an invented styles object) into misleading
// "missing required flag" errors downstream — the single largest batch error
// cluster. Rewrites applied, in order:
//
//   - underscore ↔ hyphen forms of a declared flag (already tolerated by
//     mapFlagView — accepted here as-is)
//   - lowerCamelCase → the declared flag (sheetName → sheet_name)
//   - the command's intuitive-alias table (size → width/height on the resize
//     pair) — the same commandFlagAliases the cobra path applies
//   - "ranges" with a single-entry array unwraps onto "range"; a multi-entry
//     array gets a split-into-sub-ops prescription instead
//
// Anything else errors with a did-you-mean. Returns a bare error; the caller
// wraps it with the operations[i] (<shortcut>) context and key contract.
func normalizeSubOpInputKeys(sc string, input map[string]interface{}) error {
	vocab := subOpKeyVocabulary(sc)
	if vocab == nil {
		return nil
	}
	keys := make([]string, 0, len(input))
	for k := range input {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	aliases := commandFlagAliases[sc]
	for _, k := range keys {
		hv := strings.ReplaceAll(k, "_", "-")
		if vocab[hv] {
			continue
		}
		if kebab := camelToKebab(k); kebab != "" && vocab[kebab] {
			input[strings.ReplaceAll(kebab, "-", "_")] = input[k]
			delete(input, k)
			continue
		}
		if target, ok := aliases[strings.ToLower(hv)]; ok && vocab[target] {
			if _, taken := input[target]; !taken {
				if _, taken := input[strings.ReplaceAll(target, "-", "_")]; !taken {
					input[target] = input[k]
					delete(input, k)
					continue
				}
			}
		}
		if strings.ToLower(hv) == "ranges" && vocab["range"] && !vocab["ranges"] {
			if arr, isArr := input[k].([]interface{}); isArr {
				if len(arr) == 1 {
					if s, isStr := arr[0].(string); isStr {
						input["range"] = s
						delete(input, k)
						continue
					}
				}
				return fmt.Errorf("%s takes a single \"range\" per sub-op, got %d entries in %q — split them into %d sub-ops (one per range)", sc, len(arr), k, len(arr)) //nolint:forbidigo // intermediate error; the batch dispatcher wraps it into a typed operations validation error
			}
			if s, isStr := input[k].(string); isStr {
				input["range"] = s
				delete(input, k)
				continue
			}
		}
		msg := fmt.Sprintf("unknown input key %q", k)
		display := make([]string, 0, len(vocab))
		for name := range vocab {
			display = append(display, strings.ReplaceAll(name, "-", "_"))
		}
		sort.Strings(display)
		if match := suggest.Closest(strings.ToLower(hv), display, 1); len(match) > 0 {
			msg += fmt.Sprintf(" — did you mean %q?", match[0])
		}
		return fmt.Errorf("%s", msg) //nolint:forbidigo // intermediate error; the batch dispatcher wraps it into a typed operations validation error
	}
	return nil
}

// translateBatchOp 把一个 CLI 视角的 {shortcut, input} 翻成底层 MCP
// batch_update 的 {tool_name, input}。`index` 用于错误信息定位。input 用
// shortcut 的 CLI flag 名（连字符/下划线均可），经该 shortcut 的 standalone
// translator 翻成 MCP body。
//
// 失败场景：
//   - shortcut 字段缺失 / 非 string
//   - shortcut 不在 dispatch 表（拼写错；read 操作；嵌套 fan-out wrapper）
//   - input 不是 object
//   - input 里手填了 operation（由 shortcut 名隐含，禁手填以防 mismatch）
//   - input 里手填了 excel_id / spreadsheet_token / url
//   - 子操作的 translator 报错（如缺必填字段）
func translateBatchOp(raw interface{}, token string, index int) (map[string]interface{}, error) {
	op, ok := raw.(map[string]interface{})
	if !ok {
		return nil, sheetsValidationForFlag("operations", "operations[%d] must be a JSON object", index)
	}
	scRaw, present := op["shortcut"]
	if !present {
		return nil, sheetsValidationForFlag("operations", "operations[%d]: 'shortcut' field is required", index).
			WithHint(`each entry must look like {"shortcut":"+cells-set","input":{"sheet_name":"…","range":"A1:B2","cells":[[…]]}} — input uses the shortcut's own flag names`)
	}
	sc, ok := scRaw.(string)
	if !ok || sc == "" {
		return nil, sheetsValidationForFlag("operations", "operations[%d]: 'shortcut' must be a non-empty string (got %T)", index, scRaw)
	}
	mapping, ok := batchOpDispatch[sc]
	if !ok {
		// Inline the full allow-list: an agent that guessed a read op or a
		// fan-out wrapper can pick the right shortcut immediately instead of
		// spending a --print-schema round trip on the operations enum.
		return nil, sheetsValidationForFlag(
			"operations",
			"operations[%d]: shortcut %q not allowed in +batch-update "+
				"(read ops / fan-out wrappers like +batch-update / +styles-put / +cells-batch-set-style / +cells-batch-clear / +dropdown-{update,delete} are excluded)",
			index, sc,
		).WithHint("allowed shortcuts: %s", strings.Join(allowedBatchShortcuts(), ", "))
	}
	inputRaw, hasInput := op["input"]
	var input map[string]interface{}
	if !hasInput || inputRaw == nil {
		input = map[string]interface{}{}
	} else {
		input, ok = inputRaw.(map[string]interface{})
		if !ok {
			return nil, sheetsValidationForFlag("operations", "operations[%d] (%s): 'input' must be a JSON object (got %T)", index, sc, inputRaw)
		}
	}
	// 禁手填 operation —— 由 shortcut 名表达，手填易与 shortcut 不一致。
	if _, has := input["operation"]; has {
		return nil, sheetsValidationForFlag(
			"operations",
			"operations[%d] (%s): do not pass input.operation manually — it is implied by the shortcut name",
			index, sc,
		)
	}
	// 禁在 sub-op 重复填 spreadsheet 定位 —— 由 +batch-update 顶层 --url/--token 统一提供。
	// 连字符 / 下划线两种写法都算命中（spreadsheet-token 与 spreadsheet_token 同罪）。
	for userKey := range input {
		normalized := strings.ReplaceAll(userKey, "-", "_")
		for _, k := range reservedSubOpKeys {
			if normalized == k {
				return nil, sheetsValidationForFlag(
					"operations",
					"operations[%d] (%s): do not pass input.%s — it is already set from +batch-update top-level --url / --token",
					index, sc, userKey,
				)
			}
		}
	}
	// 拒绝任何额外的 sub-op 顶层 key（防御未来 schema drift / 用户笔误）。
	for k := range op {
		if k != "shortcut" && k != "input" {
			return nil, sheetsValidationForFlag("operations", "operations[%d] (%s): unknown top-level key %q (expected only 'shortcut' and 'input')", index, sc, k)
		}
	}
	// Reject / rewrite off-vocabulary input keys BEFORE any value reads: an
	// unknown key silently ignored surfaces later as a misleading
	// "missing required flag" error (the top batch error cluster in evals).
	if err := normalizeSubOpInputKeys(sc, input); err != nil {
		verr := sheetsValidationForFlag("operations", "operations[%d] (%s): %v", index, sc, err)
		if contract := subOpInputContract(sc); contract != "" {
			verr = verr.WithHint("%s input keys: %s", sc, contract)
		}
		return nil, verr
	}
	fv := newMapFlagViewForCommand(sc, input)
	// operations is skipped by parse-time schema validation, so type-check the
	// sub-op's scalar fields here before the translator reads them via
	// Int/Bool/Float64 (which would otherwise coerce a wrong type to zero).
	if err := fv.validateRawTypes(); err != nil {
		return nil, sheetsValidationForFlag("operations", "operations[%d] (%s): %v", index, sc, err)
	}
	if err := fv.normalizeAndValidateEnums(); err != nil {
		return nil, sheetsValidationForFlag("operations", "operations[%d] (%s): %v", index, sc, err)
	}
	sheetIDFlag, sheetNameFlag := sheetSelectorFlagsForSubOp(sc)
	sheetID := strings.TrimSpace(fv.Str(sheetIDFlag))
	sheetName := strings.TrimSpace(fv.Str(sheetNameFlag))
	body, err := mapping.translate(fv, token, sheetID, sheetName)
	if err != nil {
		// The inner error names one problem at a time (first missing flag);
		// the hint lists the sub-op's complete key contract so an agent fixes
		// every gap in a single retry instead of iterating flag by flag.
		verr := sheetsValidationForFlag("operations", "operations[%d] (%s): %v", index, sc, err)
		if contract := subOpInputContract(sc); contract != "" {
			verr = verr.WithHint("%s input keys: %s", sc, contract)
		}
		return nil, verr
	}
	return map[string]interface{}{
		"tool_name": mapping.mcpToolName,
		"input":     body,
	}, nil
}

// maxBatchOperations caps how many sub-operations a single +batch-update may
// carry. Every translated op (with its own cells/properties payload) is held in
// the out slice at once before the whole batch is marshaled, so an unbounded
// operation count is the same unbounded-materialization hazard as the fan-out
// matrix, on the operations axis.
const maxBatchOperations = 100

// maxAggregatedOpErrors caps how many per-op validation errors ride in one
// aggregated message; past it the count is summarized. Keeps a 100-op batch
// where everything is wrong from producing a page-long error.
const maxAggregatedOpErrors = 10

// translateBatchOperations 翻译整个 ops 数组。校验错误**聚合上报**：翻译
// 所有 op、每个失败 op 记录首错，一次性返回全部——评测里首错即断曾把一题
// 拖成最多 7 轮"修一个错、重发、再报下一个"的往返（table-put --styles 的
// 聚合先例已验证一次报全的收益）。单错时原样返回，保持与 standalone 报错
// 逐字一致（batch-vs-standalone 契约测试锁定）。
func translateBatchOperations(rawOps []interface{}, token string) ([]interface{}, error) {
	if len(rawOps) == 0 {
		return nil, sheetsValidationForFlag("operations", "--operations must be a non-empty JSON array")
	}
	if len(rawOps) > maxBatchOperations {
		batches := (len(rawOps) + maxBatchOperations - 1) / maxBatchOperations
		return nil, sheetsValidationForFlag("operations", "--operations accepts at most %d entries; got %d", maxBatchOperations, len(rawOps)).
			WithHint("split the operations into %d separate +batch-update calls of at most %d entries each", batches, maxBatchOperations)
	}
	out := make([]interface{}, 0, len(rawOps))
	var totalCells int64
	var opErrs []*errs.ValidationError
	for i, raw := range rawOps {
		translated, err := translateBatchOp(raw, token, i)
		if err != nil {
			var verr *errs.ValidationError
			if !errors.As(err, &verr) {
				return nil, err
			}
			opErrs = append(opErrs, verr)
			continue
		}
		// The cell budget only means anything for a batch that can still run;
		// once an op has failed validation the batch is rejected anyway.
		if len(opErrs) == 0 {
			totalCells += translatedCellCount(translated)
			if totalCells > maxStampMatrixCells {
				return nil, sheetsValidationForFlag("operations",
					"--operations materialize %d cells total, over the %d-cell safety cap; reduce the number or size of cell operations",
					totalCells, maxStampMatrixCells)
			}
		}
		out = append(out, translated)
	}
	switch len(opErrs) {
	case 0:
		return out, nil
	case 1:
		return nil, opErrs[0]
	}
	return nil, aggregateOpErrors(opErrs)
}

// aggregateOpErrors folds several per-op validation errors into one, keeping
// each op's own operations[i] (<shortcut>) context in the message and the
// distinct hints (per-shortcut key contracts) joined in the hint.
func aggregateOpErrors(opErrs []*errs.ValidationError) *errs.ValidationError {
	shown := opErrs
	var more int
	if len(shown) > maxAggregatedOpErrors {
		shown = shown[:maxAggregatedOpErrors]
		more = len(opErrs) - maxAggregatedOpErrors
	}
	parts := make([]string, 0, len(shown))
	var hints []string
	seenHint := map[string]bool{}
	for _, e := range shown {
		parts = append(parts, e.Message)
		if e.Hint != "" && !seenHint[e.Hint] && len(hints) < 3 {
			seenHint[e.Hint] = true
			hints = append(hints, e.Hint)
		}
	}
	msg := fmt.Sprintf("%d operations failed validation (fix them all, then resend once): %s",
		len(opErrs), strings.Join(parts, "; "))
	if more > 0 {
		msg += fmt.Sprintf("; … and %d more", more)
	}
	verr := sheetsValidationForFlag("operations", "%s", msg)
	if len(hints) > 0 {
		verr = verr.WithHint("%s", strings.Join(hints, " | "))
	}
	return verr
}

func translatedCellCount(op map[string]interface{}) int64 {
	input, _ := op["input"].(map[string]interface{})
	switch cells := input["cells"].(type) {
	case [][]interface{}:
		var total int64
		for _, row := range cells {
			total += int64(len(row))
		}
		return total
	case []interface{}:
		var total int64
		for _, rawRow := range cells {
			if row, ok := rawRow.([]interface{}); ok {
				total += int64(len(row))
			}
		}
		return total
	default:
		return 0
	}
}
