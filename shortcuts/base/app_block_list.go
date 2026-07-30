// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppBlockList = common.Shortcut{
	Service:     "base",
	Command:     "+app-block-list",
	Description: "List blocks on a BaseApp page",
	Risk:        "read",
	Scopes:      []string{"base:appmode_block:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		pageIDFlag(true),
		{Name: "type", Desc: "filter returned blocks by type", Enum: appBlockTypes()},
		{Name: "page-size", Type: "int", Default: "20", Desc: "page size; must be positive"},
		{Name: "page-token", Desc: "pagination token"},
	},
	Tips: []string{
		"lark-cli base +app-block-list --app-token <app_token> --page-id <page_id>",
		"Filter one block type: lark-cli base +app-block-list --app-token <app_token> --page-id <page_id> --type statistics",
		"--type filters each returned page locally. Continue with page_token while has_more=true to cover the whole page.",
		"Use block_id for +app-block-get/update. For chart data, pass chart_token to +app-block-get-data --block-id.",
		"These are page blocks, not dashboard blocks: do not pass a block_id from here to +dashboard-block-get.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		_, err := common.ValidatePageSizeTyped(runtime, "page-size", 20, 1, int(^uint(0)>>1))
		return err
	},
	DryRun: dryRunAppBlockList,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeAppBlockList(runtime)
	},
}
