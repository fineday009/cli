// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppBlockCreate = common.Shortcut{
	Service:     "base",
	Command:     "+app-block-create",
	Description: "Create a block on a BaseApp page",
	Risk:        "write",
	Scopes:      []string{"base:appmode_block:create"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		pageIDFlag(true),
		{Name: "name", Desc: "block name", Required: true},
		{Name: "type", Desc: "block type: chart(column|bar|line|pie|ring|area|combo|scatter|funnel|wordCloud|radar|statistics) | richText | list(standardList|detailList|cardList|groupedList|collapsibleList). Read lark-base-baseapp-block-data-config.md before creating.", Required: true, Enum: appBlockTypes()},
		{Name: "data-config", Desc: "data_config JSON object; read lark-base-baseapp-block-data-config.md for the SSOT"},
		{Name: "position", Desc: `block position JSON object, e.g. {"x":0,"y":0,"w":12,"h":8}; omit to let the platform place it`},
		{Name: "user-id-type", Desc: "user ID type for user fields in filters: open_id / union_id / user_id"},
		{Name: "no-validate", Type: "bool", Desc: "skip local data_config validation and normalization; send data_config as-is"},
	},
	Tips: []string{
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Order Count" --type statistics --data-config '{"table_name":"Orders","count_all":true}'`,
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Notes" --type richText --data-config '{"text":"# Sales overview"}'`,
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Open orders" --type standardList --data-config '{"table_id":"tblxxx","view_id":"viwxxx","fields":["fldxxx"]}'`,
		"Before creating data-backed blocks, use +table-list and +field-list to confirm real table and field names.",
		"Chart data_config uses table and field names; list data_config accepts table_id/view_id/field IDs.",
		"Read lark-base-baseapp-block-data-config.md as the SSOT for chart, list and richText config; do not invent data_config from natural language.",
		"Block type cannot be changed after creation and this phase has no delete command, so a wrong --type can only be fixed in the UI. Confirm the type before creating.",
		"There is no page-level arrange command; omit --position to accept the platform layout.",
		"Record the returned block_id; +app-block-update and +app-block-get-data need it.",
		"Create blocks sequentially; do not parallelize multiple block creates for the same page.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		blockType := strings.TrimSpace(runtime.Str("type"))
		if !isAppBlockType(blockType) {
			return errs.NewValidationError(errs.SubtypeInvalidArgument, "--type %q 不在支持的 block 类型内: %s", blockType, strings.Join(appBlockTypes(), ", ")).WithParam("--type")
		}
		if _, err := parseBlockPosition(runtime); err != nil {
			return err
		}
		if runtime.Bool("no-validate") {
			return nil
		}
		pc := newParseCtx(runtime)
		raw := strings.TrimSpace(runtime.Str("data-config"))
		if raw == "" {
			if isTextBlockType(blockType) {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "%s 类型组件必须提供 data-config，包含必填字段 text", blockType).WithParam("--data-config")
			}
			return nil
		}
		cfg, err := parseJSONObject(pc, raw, "data-config")
		if err != nil {
			return err
		}
		norm := normalizeDataConfig(cfg)
		if problems := validateBlockDataConfig(blockType, norm); len(problems) > 0 {
			return formatDataConfigErrors(problems)
		}
		b, _ := json.Marshal(norm)
		_ = runtime.Cmd.Flags().Set("data-config", string(b))
		return nil
	},
	DryRun: dryRunAppBlockCreate,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeAppBlockCreate(runtime)
	},
}
