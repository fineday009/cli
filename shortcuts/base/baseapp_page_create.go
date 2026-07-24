// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseAppPageCreate = common.Shortcut{
	Service:     "base",
	Command:     "+app-page-create",
	Description: "Create a page in a BaseApp",
	Risk:        "write",
	Scopes:      []string{"base:appmode_page:create"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		{Name: "name", Desc: "page name", Required: true},
		{Name: "parent-page-id", Desc: "parent page ID; omit to create a top-level page"},
		{Name: "prev-page-id", Desc: "insert after this page ID; omit to insert first"},
		{Name: "to-last", Type: "bool", Desc: "append to the end instead of using --prev-page-id"},
	},
	Tips: []string{
		`lark-cli base +app-page-create --app-token <app_token> --name "Overview" --to-last`,
		"Page names must be unique within an app; the CLI checks existing pages before creation.",
		"--prev-page-id and --to-last both control ordering; pass at most one.",
		"Record the returned page_id; every +app-block-* command needs it.",
		"Only page nodes are supported in this phase; page groups are not creatable through the CLI.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return validateWorkspaceOrdering(runtime, "prev-page-id")
	},
	DryRun: dryRunBaseappPageCreate,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeBaseappPageCreate(runtime)
	},
}
