// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppPageRename = common.Shortcut{
	Service:     "base",
	Command:     "+baseapp-page-rename",
	Description: "Rename a BaseApp page",
	Risk:        "write",
	Scopes:      []string{"base:appmode_page:write"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		pageIDFlag(true),
		{Name: "name", Desc: "new page name", Required: true},
	},
	Tips: []string{
		`lark-cli base +baseapp-page-rename --app-token <app_token> --page-id <page_id> --name "Overview"`,
		"Renaming does not move the page; ordering and parent stay unchanged.",
	},
	DryRun: dryRunBaseappPageRename,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappPageRename(runtime)
	},
}
