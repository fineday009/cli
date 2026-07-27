// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

// BaseAppBlockGetData is a thin wrapper over the dashboard block data endpoint.
// App page blocks and dashboard blocks are the same underlying entity living in
// the same base, and that endpoint takes base_token + block_id with no
// container ID, so it serves both. The execute and dry-run hooks are shared
// with +dashboard-block-get-data on purpose: the behaviour is defined once and
// cannot drift. The flag exception (--base-token, no --app-token / --page-id)
// is spelled out in the description and tips because it breaks the habit every
// other +app-block-* command sets up.
var BaseAppBlockGetData = common.Shortcut{
	Service:     "base",
	Command:     "+app-block-get-data",
	Description: "Get computed data for a BaseApp page chart block (takes --base-token, not --app-token)",
	Risk:        "read",
	Scopes:      []string{"base:dashboard:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		baseTokenFlag(true),
		appBlockIDFlag(true),
		{Name: "app-token", Desc: "hidden compatibility flag accepted by app block commands; ignored by get-data", Hidden: true},
		{Name: "page-id", Desc: "hidden compatibility flag accepted by app block commands; ignored by get-data", Hidden: true},
	},
	Tips: []string{
		"lark-cli base +app-block-get-data --base-token <base_token> --block-id <block_id>",
		"Unlike every other +app-block-* command this one takes --base-token and needs neither --app-token nor --page-id.",
		"--base-token is the Base backing the app; read it from a +app-get ref key or +app-create.",
		"It shares the dashboard endpoint, so the response is the same chart protocol JSON as +dashboard-block-get-data.",
		"List and richText blocks have no computed data; use +app-block-get for their metadata instead.",
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		return dryRunDashboardBlockGetData(ctx, runtime)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeDashboardBlockGetData(runtime)
	},
}
