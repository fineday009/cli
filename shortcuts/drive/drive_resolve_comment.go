// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package drive

import (
	"context"
	"fmt"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/validate"
	"github.com/larksuite/cli/shortcuts/common"
)

const (
	driveResolveCommentActionResolve = "resolve"
	driveResolveCommentActionRestore = "restore"
)

var driveResolveCommentOp = driveCommentOp{
	Label: "comment resolve/restore",
	Types: []string{"doc", "docx", "sheet", "file", "slides", "bitable", "apps"},
}

type driveResolveCommentSpec struct {
	Ref       driveCommentRef
	CommentID string
	Action    string
}

func (s driveResolveCommentSpec) IsSolved() bool {
	return s.Action == driveResolveCommentActionResolve
}

func (s driveResolveCommentSpec) RequestBody() map[string]interface{} {
	return map[string]interface{}{
		"is_solved": s.IsSolved(),
	}
}

// DriveResolveComment solves or restores a comment through the Drive comment
// patch API, while accepting Wiki URLs/tokens and resolving them to the
// underlying object.
var DriveResolveComment = common.Shortcut{
	Service:           "drive",
	Command:           "+resolve-comment",
	Description:       "Resolve or restore a comment on doc/docx/sheet/file/slides/base(bitable)/apps, with URL parsing and Wiki token unwrapping",
	Risk:              "write",
	Scopes:            []string{"docs:document.comment:write_only"},
	ConditionalScopes: []string{"wiki:node:read"},
	AuthTypes:         []string{"user", "bot"},
	Flags: append(driveCommentTargetFlags(driveResolveCommentOp),
		common.Flag{Name: "comment-id", Desc: "comment ID to resolve or restore (from drive +list-comments)", Required: true},
		common.Flag{Name: "action", Desc: "resolve marks the comment solved; restore reopens a solved comment", Required: true, Enum: []string{driveResolveCommentActionResolve, driveResolveCommentActionRestore}},
	),
	Tips: []string{
		"Comment IDs come from `drive +list-comments` (items[].comment_id).",
		"--action resolve sends is_solved=true; --action restore sends is_solved=false.",
		"Back-to-back state flips on the same comment can hit server rate limiting (HTTP 429); space out consecutive calls or retry after a short delay.",
		"Wiki URLs/tokens are resolved to the underlying document automatically.",
	},
	Validate: func(ctx context.Context, runtime *common.RuntimeContext) error {
		_, err := readDriveResolveCommentSpec(runtime)
		return err
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		spec, err := readDriveResolveCommentSpec(runtime)
		if err != nil {
			return common.NewDryRunAPI().Set("error", err.Error())
		}
		return buildDriveResolveCommentDryRun(spec)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		spec, err := readDriveResolveCommentSpec(runtime)
		if err != nil {
			return err
		}

		target, err := resolveDriveCommentTarget(ctx, runtime, driveResolveCommentOp, spec.Ref)
		if err != nil {
			return err
		}

		verb := "Resolving"
		if !spec.IsSolved() {
			verb = "Restoring"
		}
		fmt.Fprintf(runtime.IO().ErrOut, "%s comment %s in %s...\n", verb, spec.CommentID, common.MaskToken(target.FileToken))
		path := fmt.Sprintf(
			"/open-apis/drive/v1/files/%s/comments/%s",
			validate.EncodePathSegment(target.FileToken),
			validate.EncodePathSegment(spec.CommentID),
		)
		if _, err := runtime.CallAPITyped(
			"PATCH",
			path,
			map[string]interface{}{"file_type": target.FileType},
			spec.RequestBody(),
		); err != nil {
			return err
		}

		runtime.Out(driveCommentTargetOutput(target, map[string]interface{}{
			"comment_id": spec.CommentID,
			"action":     spec.Action,
			"is_solved":  spec.IsSolved(),
			"updated":    true,
		}), nil)
		return nil
	},
}

func readDriveResolveCommentSpec(runtime *common.RuntimeContext) (driveResolveCommentSpec, error) {
	ref, err := resolveDriveCommentInput(driveResolveCommentOp, runtime.Str("url"), runtime.Str("token"), runtime.Str("type"))
	if err != nil {
		return driveResolveCommentSpec{}, err
	}
	commentID := strings.TrimSpace(runtime.Str("comment-id"))
	if err := validateDriveCommentPathID(commentID, "--comment-id"); err != nil {
		return driveResolveCommentSpec{}, err
	}
	action, err := parseDriveResolveCommentAction(runtime.Str("action"))
	if err != nil {
		return driveResolveCommentSpec{}, err
	}
	return driveResolveCommentSpec{
		Ref:       ref,
		CommentID: commentID,
		Action:    action,
	}, nil
}

// parseDriveResolveCommentAction normalizes and validates the --action value.
// The flag's Enum already rejects unknown values from the CLI, so the error
// branch only guards direct callers.
func parseDriveResolveCommentAction(raw string) (string, error) {
	action := strings.ToLower(strings.TrimSpace(raw))
	if action != driveResolveCommentActionResolve && action != driveResolveCommentActionRestore {
		return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "invalid --action %q; allowed: %s, %s", action, driveResolveCommentActionResolve, driveResolveCommentActionRestore).WithParam("--action")
	}
	return action, nil
}

func buildDriveResolveCommentDryRun(spec driveResolveCommentSpec) *common.DryRunAPI {
	if spec.Ref.Type == "wiki" {
		return common.NewDryRunAPI().
			Desc("2-step orchestration: resolve wiki -> patch comment solved state").
			GET("/open-apis/wiki/v2/spaces/get_node").
			Desc("[1] Resolve wiki node to underlying document").
			Params(map[string]interface{}{"token": spec.Ref.Token}).
			PATCH("/open-apis/drive/v1/files/<obj_token from step 1>/comments/:comment_id").
			Desc("[2] Patch comment solved state on resolved document").
			Params(map[string]interface{}{"file_type": "<obj_type from step 1>"}).
			Body(spec.RequestBody()).
			Set("comment_id", spec.CommentID)
	}

	return common.NewDryRunAPI().
		Desc("1-step request: patch comment solved state").
		PATCH("/open-apis/drive/v1/files/:file_token/comments/:comment_id").
		Params(map[string]interface{}{"file_type": spec.Ref.Type}).
		Body(spec.RequestBody()).
		Set("file_token", spec.Ref.Token).
		Set("comment_id", spec.CommentID)
}
