// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppBlockUpdate = common.Shortcut{
	Service:     "base",
	Command:     "+app-block-update",
	Description: "Update a block on a BaseApp page",
	Risk:        "write",
	Scopes:      []string{"base:appmode_block:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		pageIDFlag(true),
		appBlockIDFlag(true),
		{Name: "name", Desc: "new block name"},
		{Name: "data-config", Desc: "data_config JSON object; read lark-base-baseapp-block-data-config.md for the SSOT"},
		{Name: "position", Desc: `block position JSON object, e.g. {"x":0,"y":0,"w":12,"h":8}`},
		{Name: "user-id-type", Desc: "user ID type for user fields in filters: open_id / union_id / user_id"},
		{Name: "no-validate", Type: "bool", Desc: "skip local data_config normalization; send data_config as-is"},
	},
	Tips: []string{
		`lark-cli base +app-block-update --app-token <app_token> --page-id <page_id> --block-id <block_id> --name "Monthly sales"`,
		`lark-cli base +app-block-update --app-token <app_token> --page-id <page_id> --block-id <block_id> --data-config '{"filter":{"conjunction":"and","conditions":[{"field_name":"Status","operator":"is","value":"Closed"}]}}'`,
		"Read lark-base-baseapp-block-data-config.md as the SSOT; do not invent data_config from natural language.",
		"Use +app-block-get first to inspect the current data_config before replacing nested values.",
		"Block type cannot be changed, and this phase has no delete command; a wrong type can only be fixed in the UI.",
		"Only explicitly provided data_config fields are sent. Omitted fields remain unchanged; each provided array/object field is replaced according to the API protocol.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		if _, err := parseBlockPosition(runtime); err != nil {
			return err
		}
		if runtime.Bool("no-validate") {
			return nil
		}
		raw := strings.TrimSpace(runtime.Str("data-config"))
		if raw == "" {
			return nil
		}
		pc := newParseCtx(runtime)
		cfg, err := parseJSONObject(pc, raw, "data-config")
		if err != nil {
			return err
		}
		// update 不传 type，无法做强类型校验；只归一化后交给后端验证具体字段
		norm := normalizeDataConfig(cfg)
		b, _ := json.Marshal(norm)
		_ = runtime.Cmd.Flags().Set("data-config", string(b))
		return nil
	},
	DryRun: dryRunAppBlockUpdate,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeAppBlockUpdate(runtime)
	},
}
