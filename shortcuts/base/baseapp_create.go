// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppCreate = common.Shortcut{
	Service:     "base",
	Command:     "+app-create",
	Description: "Create a blank BaseApp with a blank base",
	Risk:        "write",
	Scopes: []string{
		"base:appmode:create",
		"base:app:create",
		"base:workspace:write",
	},
	AuthTypes: authTypes(),
	Flags: []common.Flag{
		{Name: "name", Desc: "BaseApp name", Required: true},
		workspaceTokenFlag(false),
		{Name: "base-name", Desc: "name of the blank base created with the app; defaults to the platform default"},
		{Name: "table-name", Desc: "name of the first table in the blank base; defaults to the platform default"},
	},
	Tips: []string{
		`lark-cli base +app-create --name "Sales app" --workspace-token <workspace_token>`,
		"After the app is created, the CLI creates one blank Base and moves it into the same Workspace.",
		"Record both tokens: page/block commands take app_token, while table/field/record commands take base_token.",
		"Omitting --workspace-token lets the platform pick the default location.",
		"If Base creation or moving fails after the app is created, the result is marked partial and includes a retry command. Do not create the app again.",
	},
	DryRun: dryRunBaseappCreate,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappCreate(runtime)
	},
}
