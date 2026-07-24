// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseWorkspaceEntityRemove = common.Shortcut{
	Service:     "base",
	Command:     "+workspace-entity-remove",
	Description: "Remove a base or BaseApp from a workspace",
	Risk:        "high-risk-write",
	Scopes:      []string{"base:workspace:update"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		workspaceTokenFlag(true),
		{Name: "entity-id", Desc: "workspace entity ID from +workspace-entity-list", Required: true},
	},
	Tips: []string{
		"lark-cli base +workspace-entity-remove --workspace-token <workspace_token> --entity-id <entity_id> --yes",
		"This removes the entity from the workspace tree only; the underlying base or BaseApp is not deleted.",
		"--entity-id is the entity_id returned by +workspace-entity-list, not a base_token or app_token.",
		baseHighRiskYesTip,
	},
	DryRun: dryRunWorkspaceEntityRemove,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeWorkspaceEntityRemove(runtime)
	},
}
