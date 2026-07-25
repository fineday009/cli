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
		{Name: "type", Desc: "block type: chart(column|bar|line|pie|ring|area|combo|scatter|funnel|wordCloud|radar|statistics) | richText | list. Read lark-base-baseapp-block-data-config.md before creating.", Required: true, Enum: appBlockTypes()},
		{Name: "sub-type", Desc: "list subtype: standard|grouped|collapsible|card|detail; defaults to standard", Enum: appListSubTypes},
		{Name: "data-config", Desc: "data_config JSON object; read lark-base-baseapp-block-data-config.md for the SSOT"},
		{Name: "position", Desc: `block position JSON object, e.g. {"x":0,"y":0,"w":12,"h":8}; omit to let the platform place it`},
		{Name: "user-id-type", Desc: "user ID type for user fields in filters: open_id / union_id / user_id"},
		{Name: "no-validate", Type: "bool", Desc: "skip local data_config validation and normalization; send data_config as-is"},
	},
	Tips: []string{
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Order Count" --type statistics --data-config '{"base_token":"basxxx","data_sources":[{"table_name":"Orders","count_all":true}]}'`,
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Monthly sales" --type column --data-config '{"base_token":"basxxx","data_sources":[{"table_name":"Orders","series":[{"field_name":"Amount","rollup":"SUM"}],"group_by":[{"field_name":"Month","sort":{"type":"group","order":"asc"}}]}]}'`,
		"Chart blocks use multi-datasource data_config: one top-level base_token shared by all sources, with table_name/series/count_all/group_by/filter inside each data_sources[] element (richText needs none). App block commands carry no --base-token.",
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Notes" --type richText --data-config '{"text":"# Sales overview"}'`,
		`lark-cli base +app-block-create --app-token <app_token> --page-id <page_id> --name "Open orders" --type list --sub-type standard --data-config '{"base_token":"basxxx","table_name":"Orders","columns":[]}'`,
		"Before creating data-backed blocks, use +table-list and +field-list to confirm real table and field names.",
		"A list accepts exactly one base_token, and that Base must be in the same Workspace as the App.",
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
		if strings.EqualFold(blockType, "list") {
			subType, ok := normalizeAppListSubType(runtime.Str("sub-type"))
			if !ok {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "--sub-type 仅支持 %s", strings.Join(appListSubTypes, "|")).WithParam("--sub-type")
			}
			if raw := strings.TrimSpace(runtime.Str("data-config")); raw != "" && !runtime.Bool("no-validate") {
				cfg, err := parseJSONObject(newParseCtx(runtime), raw, "data-config")
				if err != nil {
					return err
				}
				if problems := validateAppListDataConfig(subType, cfg); len(problems) > 0 {
					return formatDataConfigErrors(problems)
				}
			}
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
		norm := cfg
		if !strings.EqualFold(blockType, "list") {
			// Chart blocks use the multi-datasource ChartDataConfig shape
			// (base_token top-level, table_name/series/count_all/group_by/filter
			// per data_sources[] element); richText keeps the flat text shape.
			if isChartBlockType(blockType) {
				norm = normalizeAppChartDataConfig(cfg)
			} else {
				norm = normalizeDataConfig(cfg)
			}
			if problems := validateAppBlockDataConfig(blockType, norm); len(problems) > 0 {
				return formatDataConfigErrors(problems)
			}
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
