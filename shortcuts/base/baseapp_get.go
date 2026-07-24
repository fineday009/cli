// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppGet = common.Shortcut{
	Service:     "base",
	Command:     "+app-get",
	Description: "Get BaseApp info",
	Risk:        "read",
	Scopes:      []string{"base:appmode:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		{Name: "with-pages", Type: "bool", Desc: "include the page list in the response"},
		{Name: "with-components", Type: "bool", Desc: "include page blocks in the response; implies a heavier payload"},
	},
	Tips: []string{
		"lark-cli base +app-get --app-token <app_token> --with-pages",
		"base_tokens tells you which bases back this app; table/field/record commands take those tokens.",
		"For block-level detail prefer +app-page-get --with-components or +app-block-list over --with-components here.",
	},
	DryRun: dryRunBaseappGet,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappGet(runtime)
	},
}
