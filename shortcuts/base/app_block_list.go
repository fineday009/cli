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
		{Name: "type", Desc: "filter by block type, e.g. line or list; omit to list all types", Enum: appBlockTypes()},
		{Name: "page-size", Type: "int", Default: "100", Desc: "page size, range 1-100"},
		{Name: "page-token", Desc: "pagination token"},
	},
	Tips: []string{
		"lark-cli base +app-block-list --app-token <app_token> --page-id <page_id>",
		"Use the returned block_id for +app-block-get/update; chart blocks also accept it in +app-block-get-data.",
		"These are page blocks, not dashboard blocks: do not pass a block_id from here to +dashboard-block-get.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		_, err := common.ValidatePageSizeTyped(runtime, "page-size", 100, 1, 100)
		return err
	},
	DryRun: dryRunAppBlockList,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeAppBlockList(runtime)
	},
}
