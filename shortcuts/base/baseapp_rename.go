// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppRename = common.Shortcut{
	Service:     "base",
	Command:     "+baseapp-rename",
	Description: "Rename a BaseApp",
	Risk:        "write",
	Scopes:      []string{"base:appmode:write"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		{Name: "name", Desc: "new BaseApp name", Required: true},
	},
	Tips: []string{
		`lark-cli base +baseapp-rename --app-token <app_token> --name "Sales app v2"`,
		"Renaming the app does not rename the base behind it.",
	},
	DryRun: dryRunBaseappRename,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappRename(runtime)
	},
}
