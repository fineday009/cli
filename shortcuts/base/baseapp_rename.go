// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppRename = common.Shortcut{
	Service:     "base",
	Command:     "+app-rename",
	Description: "Rename a BaseApp",
	Risk:        "write",
	Scopes:      []string{"base:appmode:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		{Name: "name", Desc: "new BaseApp name", Required: true},
	},
	Tips: []string{
		`lark-cli base +app-rename --app-token <app_token> --name "Sales app v2"`,
		"BaseApp and Base both use type=bitable in Drive file APIs.",
		"Renaming the app does not rename the base behind it.",
	},
	DryRun: dryRunBaseappRename,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappRename(runtime)
	},
}
