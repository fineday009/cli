// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppPageGet = common.Shortcut{
	Service:     "base",
	Command:     "+baseapp-page-get",
	Description: "Get a BaseApp page by ID",
	Risk:        "read",
	Scopes:      []string{"base:appmode_page:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		pageIDFlag(true),
		{Name: "with-components", Type: "bool", Desc: "include the page blocks in the response"},
	},
	Tips: []string{
		"lark-cli base +baseapp-page-get --app-token <app_token> --page-id <page_id> --with-components",
		"Without --with-components this returns page metadata only; use +app-block-list when you need paginated blocks.",
	},
	DryRun: dryRunBaseappPageGet,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappPageGet(runtime)
	},
}
