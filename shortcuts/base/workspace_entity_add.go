// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"

	"github.com/larksuite/cli/shortcuts/common"
)

var BaseWorkspaceEntityAdd = common.Shortcut{
	Service:     "base",
	Command:     "+workspace-entity-add",
	Description: "Add a base or BaseApp to a workspace",
	Risk:        "write",
	Scopes:      []string{"base:workspace:write"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		workspaceTokenFlag(true),
		{Name: "type", Desc: "entity type: base|baseapp", Required: true, Enum: entityTypeValues},
		{Name: "token", Desc: "base_token when --type base, app_token when --type baseapp", Required: true},
		{Name: "prev-entity-id", Desc: "insert after this entity_id; omit to insert first"},
		{Name: "to-last", Type: "bool", Desc: "append to the end of the workspace instead of using --prev-entity-id"},
	},
	Tips: []string{
		"lark-cli base +workspace-entity-add --workspace-token <workspace_token> --type base --token <base_token> --to-last",
		"--prev-entity-id and --to-last both control ordering; pass at most one.",
		"This only adds the entity to the workspace tree; it does not create the base or app.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		if _, err := normalizeEntityType(runtime.Str("type")); err != nil {
			return err
		}
		return validateWorkspaceOrdering(runtime, "prev-entity-id")
	},
	DryRun: dryRunWorkspaceEntityAdd,
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeWorkspaceEntityAdd(runtime)
	},
}
