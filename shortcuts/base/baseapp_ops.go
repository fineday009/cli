// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"fmt"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/shortcuts/common"
)

// ── Shared flags ─────────────────────────────────────────────────────

func workspaceTokenFlag(required bool) common.Flag {
	return common.Flag{Name: "workspace-token", Desc: "workspace token", Required: required}
}

func appTokenFlag(required bool) common.Flag {
	return common.Flag{Name: "app-token", Desc: "BaseApp token", Required: required}
}

func pageIDFlag(required bool) common.Flag {
	return common.Flag{Name: "page-id", Desc: "BaseApp page ID", Required: required}
}

func appBlockIDFlag(required bool) common.Flag {
	return common.Flag{Name: "block-id", Desc: "BaseApp page block ID", Required: required}
}

// ── Shared helpers ───────────────────────────────────────────────────

// entityTypeValues are the entity types a workspace can hold.
var entityTypeValues = []string{"base", "baseapp"}

// normalizeEntityType lowercases and validates the workspace entity type.
func normalizeEntityType(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", nil
	}
	for _, candidate := range entityTypeValues {
		if trimmed == candidate {
			return trimmed, nil
		}
	}
	return "", errs.NewValidationError(errs.SubtypeInvalidArgument, "--type 仅支持 base|baseapp，当前值: %s", raw).WithParam("--type")
}

// validateWorkspaceOrdering rejects passing both an explicit predecessor and
// --to-last, because the backend would have to pick one silently.
func validateWorkspaceOrdering(runtime *common.RuntimeContext, prevFlag string) error {
	if strings.TrimSpace(runtime.Str(prevFlag)) != "" && runtime.Bool("to-last") {
		return errs.NewValidationError(errs.SubtypeInvalidArgument, "--%s 与 --to-last 互斥，只能指定一种排序方式", prevFlag).WithParam("--" + prevFlag)
	}
	return nil
}

// appBlockBody builds the public create/update request body for app page
// blocks. Layout, position, size and display settings are intentionally not
// exposed because they are outside the public protocol.
func appBlockBody(runtime *common.RuntimeContext, includeType bool) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	if name := strings.TrimSpace(runtime.Str("name")); name != "" {
		body["name"] = name
	}
	if includeType {
		if blockType := strings.TrimSpace(runtime.Str("type")); blockType != "" {
			// The rich-text widget's wire type is "text"; the CLI exposes the
			// friendlier "richText" alias, so map it back on send.
			if strings.EqualFold(blockType, "richText") {
				blockType = "text"
			}
			body["type"] = blockType
		}
		if strings.EqualFold(strings.TrimSpace(runtime.Str("type")), "list") {
			rawSubType := strings.TrimSpace(runtime.Str("sub-type"))
			if subType, ok := normalizeAppListSubType(rawSubType); ok && rawSubType != "" {
				body["sub_type"] = subType
			}
		}
	}
	if raw := strings.TrimSpace(runtime.Str("data-config")); raw != "" {
		pc := newParseCtx(runtime)
		parsed, err := parseJSONObject(pc, raw, "data-config")
		if err != nil {
			return nil, err
		}
		body["data_config"] = parsed
	}
	return body, nil
}

func userIDTypeParams(runtime *common.RuntimeContext) map[string]interface{} {
	params := map[string]interface{}{}
	if userIDType := strings.TrimSpace(runtime.Str("user-id-type")); userIDType != "" {
		params["user_id_type"] = userIDType
	}
	return params
}

func pagingParams(runtime *common.RuntimeContext) map[string]interface{} {
	params := map[string]interface{}{"page_size": runtime.Int("page-size")}
	if pageToken := strings.TrimSpace(runtime.Str("page-token")); pageToken != "" {
		params["page_token"] = pageToken
	}
	return params
}

// ── Workspace: dry-run ───────────────────────────────────────────────

func dryRunWorkspaceCreate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/workspaces").
		Body(workspaceCreateBody(runtime))
}

func workspaceCreateBody(runtime *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	if icon := strings.TrimSpace(runtime.Str("icon")); icon != "" {
		body["icon"] = icon
	}
	return body
}

func dryRunWorkspaceEntityList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	params := pagingParams(runtime)
	if entityType, err := normalizeEntityType(runtime.Str("type")); err == nil && entityType != "" {
		params["type"] = entityType
	}
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/workspaces/:workspace_token/entities").
		Set("workspace_token", runtime.Str("workspace-token")).
		Params(params)
}

func workspaceMoveInBody(runtime *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{}
	if token := strings.TrimSpace(runtime.Str("entity-token")); token != "" {
		body["entity_token"] = token
	}
	return body
}

func dryRunWorkspaceMoveIn(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/workspaces/:workspace_token/move_in").
		Set("workspace_token", runtime.Str("workspace-token")).
		Body(workspaceMoveInBody(runtime))
}

// ── Workspace: execute ───────────────────────────────────────────────

func executeWorkspaceCreate(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "POST", baseV3Path("workspaces"), nil, workspaceCreateBody(runtime))
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"workspace": data, "created": true}, nil)
	return nil
}

func executeWorkspaceEntityList(runtime *common.RuntimeContext) error {
	params := pagingParams(runtime)
	entityType, err := normalizeEntityType(runtime.Str("type"))
	if err != nil {
		return err
	}
	if entityType != "" {
		params["type"] = entityType
	}
	data, err := baseV3Call(runtime, "GET", baseV3Path("workspaces", runtime.Str("workspace-token"), "entities"), params, nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeWorkspaceMoveIn(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "POST", baseV3Path("workspaces", runtime.Str("workspace-token"), "move_in"), nil, workspaceMoveInBody(runtime))
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"entity": data, "moved_in": true}, nil)
	return nil
}

// ── BaseApp: dry-run ─────────────────────────────────────────────────

func baseappCreateBodyWithWorkspace(runtime *common.RuntimeContext, workspaceToken string) map[string]interface{} {
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	if workspaceToken != "" {
		body["workspace_token"] = workspaceToken
	}
	return body
}

func dryRunBaseappCreate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	workspaceToken := strings.TrimSpace(runtime.Str("workspace-token"))
	dryRun := common.NewDryRunAPI()
	if workspaceToken == "" {
		dryRun.POST("/open-apis/base/v3/workspaces").
			Body(map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}).
			Desc("No Workspace was specified, so create one first with the same name as the app.")
		workspaceToken = "<created_workspace_token>"
	}
	dryRun.POST("/open-apis/base/v3/base_apps").
		Body(baseappCreateBodyWithWorkspace(runtime, workspaceToken)).
		Desc("Create the app in the selected or newly created Workspace.")
	dryRun.POST("/open-apis/base/v3/bases").
		Body(baseappBlankBaseBody(runtime)).
		Desc("After App creation succeeds, create a blank candidate Base.")
	dryRun.POST("/open-apis/base/v3/workspaces/:workspace_token/move_in").
		Set("workspace_token", workspaceToken).
		Body(map[string]interface{}{"entity_token": "<created_base_token>"}).
		Desc("Move the blank Base into the App Workspace. A failure here returns a partial-completion result and a retry command.")
	return dryRun
}

func baseappBlankBaseBody(runtime *common.RuntimeContext) map[string]interface{} {
	name := strings.TrimSpace(runtime.Str("base-name"))
	if name == "" {
		name = strings.TrimSpace(runtime.Str("name")) + " Base"
	}
	return map[string]interface{}{"name": name}
}

func baseappCreateRetryCommand(runtime *common.RuntimeContext, workspaceToken string) string {
	command := fmt.Sprintf(
		"lark-cli base +app-create --name %q --workspace-token %s",
		strings.TrimSpace(runtime.Str("name")),
		workspaceToken,
	)
	if baseName := strings.TrimSpace(runtime.Str("base-name")); baseName != "" {
		command += fmt.Sprintf(" --base-name %q", baseName)
	}
	if tableName := strings.TrimSpace(runtime.Str("table-name")); tableName != "" {
		command += fmt.Sprintf(" --table-name %q", tableName)
	}
	return command
}

func dryRunBaseappGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/base_apps/:app_token").
		Set("app_token", runtime.Str("app-token"))
}

func dryRunBaseappRename(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		PATCH("/open-apis/drive/v1/files/:file_token").
		Set("file_token", runtime.Str("app-token")).
		Params(map[string]interface{}{"type": "bitable"}).
		Body(map[string]interface{}{"new_title": strings.TrimSpace(runtime.Str("name"))})
}

// ── BaseApp: execute ─────────────────────────────────────────────────

func executeBaseappCreate(runtime *common.RuntimeContext) error {
	workspaceToken := strings.TrimSpace(runtime.Str("workspace-token"))
	var workspace map[string]interface{}
	workspaceCreated := false
	if workspaceToken == "" {
		var err error
		workspace, err = baseV3Call(runtime, "POST", baseV3Path("workspaces"), nil, map[string]interface{}{
			"name": strings.TrimSpace(runtime.Str("name")),
		})
		if err != nil {
			return err
		}
		workspaceCreated = true
		workspaceToken = firstNonEmpty(
			common.GetString(workspace, "workspace_token"),
			common.GetString(workspace, "token"),
		)
		if workspaceToken == "" {
			return runtime.OutPartialFailure(map[string]interface{}{
				"status":            "partial",
				"failed_step":       "workspace_token_resolve",
				"message":           "Workspace 已创建，但响应中缺少 workspace_token，CLI 无法继续创建应用模式；Workspace 未回滚。",
				"workspace":         workspace,
				"workspace_created": true,
				"app_created":       false,
				"completed_steps":   []interface{}{"workspace_create"},
			}, nil)
		}
	}

	app, err := baseV3Call(runtime, "POST", baseV3Path("base_apps"), nil, baseappCreateBodyWithWorkspace(runtime, workspaceToken))
	if err != nil {
		if workspaceCreated {
			out := map[string]interface{}{
				"status":            "partial",
				"failed_step":       "app_create",
				"cause":             err.Error(),
				"message":           "Workspace 已创建，但应用模式创建失败；Workspace 未回滚。请按 retry.command 在该 Workspace 中重试创建应用。",
				"workspace":         workspace,
				"workspace_token":   workspaceToken,
				"workspace_created": true,
				"app_created":       false,
				"completed_steps":   []interface{}{"workspace_create"},
				"retry": map[string]interface{}{
					"command": baseappCreateRetryCommand(runtime, workspaceToken),
				},
			}
			return runtime.OutPartialFailure(out, nil)
		}
		return err
	}
	completedSteps := []interface{}{"app_create"}
	if workspaceCreated {
		completedSteps = []interface{}{"workspace_create", "app_create"}
	}
	out := map[string]interface{}{
		"status":            "in_progress",
		"app":               app,
		"workspace_token":   workspaceToken,
		"workspace_created": workspaceCreated,
		"app_created":       true,
		"base_created":      false,
		"base_moved":        false,
		"completed_steps":   completedSteps,
	}
	if workspaceCreated {
		out["workspace"] = workspace
	}
	appToken := firstNonEmpty(common.GetString(app, "app_token"), common.GetString(app, "token"))

	base, err := baseV3Call(runtime, "POST", baseV3Path("bases"), nil, baseappBlankBaseBody(runtime))
	if err != nil {
		out["status"] = "partial"
		out["failed_step"] = "base_create"
		out["cause"] = err.Error()
		out["message"] = "App 已创建，但空 Base 创建失败；App 未回滚。请保留 app_token，并按 retry.command 重试后再把 Base 移入同一 Workspace。"
		out["retry"] = map[string]interface{}{
			"command": fmt.Sprintf("lark-cli base +base-create --name %q", common.GetString(baseappBlankBaseBody(runtime), "name")),
			"next":    fmt.Sprintf("lark-cli base +workspace-move-in --workspace-token %s --entity-token <base_token>", workspaceToken),
		}
		out["app_token"] = appToken
		return runtime.OutPartialFailure(out, nil)
	}
	baseToken := extractBasePermissionToken(base)
	out["base"] = base
	out["base_token"] = baseToken
	out["base_created"] = true
	out["completed_steps"] = append(completedSteps, "base_create")
	if baseToken == "" {
		out["status"] = "partial"
		out["failed_step"] = "base_token_resolve"
		out["message"] = "App 和空 Base 已创建，但 Base 创建响应缺少 base_token，CLI 无法继续移动；资源未回滚。"
		out["retry"] = map[string]interface{}{"command": "lark-cli base +title-resolve --title <base_name>"}
		return runtime.OutPartialFailure(out, nil)
	}

	if workspaceToken == "" {
		out["status"] = "partial"
		out["failed_step"] = "workspace_resolve"
		out["message"] = "App 和空 Base 已创建，但响应中没有 Workspace token，CLI 无法自动移动 Base；资源未回滚。"
		out["retry"] = map[string]interface{}{"command": fmt.Sprintf("lark-cli base +workspace-move-in --workspace-token <workspace_token> --entity-token %s", baseToken)}
		return runtime.OutPartialFailure(out, nil)
	}
	moveBody := map[string]interface{}{"entity_token": baseToken}
	entity, err := baseV3Call(runtime, "POST", baseV3Path("workspaces", workspaceToken, "move_in"), nil, moveBody)
	if err != nil {
		out["status"] = "partial"
		out["failed_step"] = "base_move"
		out["cause"] = err.Error()
		out["message"] = "App 和空 Base 已创建，但 Base 移入 App Workspace 失败；资源未回滚。再次执行 retry.command 即可继续，不要重复创建 App 或 Base。"
		out["retry"] = map[string]interface{}{"command": fmt.Sprintf("lark-cli base +workspace-move-in --workspace-token %s --entity-token %s", workspaceToken, baseToken)}
		return runtime.OutPartialFailure(out, nil)
	}
	out["workspace_token"] = workspaceToken
	out["workspace_entity"] = entity
	out["base_moved"] = true
	out["completed_steps"] = append(completedSteps, "base_create", "base_move")

	if strings.TrimSpace(runtime.Str("table-name")) != "" {
		renamedTable, _, renameErr := renameBaseDefaultTable(runtime, base)
		if renameErr != nil {
			out["status"] = "partial"
			out["failed_step"] = "base_initial_table_rename"
			out["cause"] = renameErr.Error()
			out["message"] = "App 和空 Base 已创建，Base 也已移入同一 Workspace，但首张表重命名失败；资源未回滚。"
			out["retry"] = map[string]interface{}{"command": fmt.Sprintf("lark-cli base +table-list --base-token %s", baseToken)}
			return runtime.OutPartialFailure(out, nil)
		}
		out["table"] = renamedTable
		out["completed_steps"] = append(completedSteps, "base_create", "base_move", "base_initial_table_rename")
	}
	out["status"] = "completed"
	runtime.Out(out, nil)
	return nil
}

func executeBaseappGet(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token")), nil, nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeBaseappRename(runtime *common.RuntimeContext) error {
	body := map[string]interface{}{"new_title": strings.TrimSpace(runtime.Str("name"))}
	data, err := runtime.CallAPITyped("PATCH", "/open-apis/drive/v1/files/"+runtime.Str("app-token"), map[string]interface{}{"type": "bitable"}, body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"file": data, "updated": true, "app_token": runtime.Str("app-token"), "type": "bitable"}, nil)
	return nil
}

// ── Page: dry-run ────────────────────────────────────────────────────

func dryRunBaseappPageList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/base_apps/:app_token/pages").
		Set("app_token", runtime.Str("app-token")).
		Params(pagingParams(runtime))
}

func baseappPageGetParams(runtime *common.RuntimeContext) map[string]interface{} {
	params := map[string]interface{}{}
	if runtime.Bool("with-components") {
		params["with_components"] = true
	}
	return params
}

func dryRunBaseappPageGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/base_apps/:app_token/pages/:page_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Params(baseappPageGetParams(runtime))
}

func baseappPageCreateBody(runtime *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	if parent := strings.TrimSpace(runtime.Str("parent-page-id")); parent != "" {
		body["parent_page_id"] = parent
	}
	if prev := strings.TrimSpace(runtime.Str("prev-page-id")); prev != "" {
		body["prev_page_id"] = prev
	}
	if runtime.Bool("to-last") {
		body["to_last"] = true
	}
	return body
}

func dryRunBaseappPageCreate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/base_apps/:app_token/pages").
		Set("app_token", runtime.Str("app-token")).
		Body(baseappPageCreateBody(runtime))
}

func dryRunBaseappPageRename(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		PATCH("/open-apis/base/v3/base_apps/:app_token/pages/:page_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Body(map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))})
}

func dryRunBaseappPageDelete(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		DELETE("/open-apis/base/v3/base_apps/:app_token/pages/:page_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id"))
}

// ── Page: execute ────────────────────────────────────────────────────

func executeBaseappPageList(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token"), "pages"), pagingParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeBaseappPageGet(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id")), baseappPageGetParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"page": data}, nil)
	return nil
}

func executeBaseappPageCreate(runtime *common.RuntimeContext) error {
	if err := ensureUniqueAppPageName(runtime, strings.TrimSpace(runtime.Str("name")), ""); err != nil {
		return err
	}
	data, err := baseV3Call(runtime, "POST", baseV3Path("base_apps", runtime.Str("app-token"), "pages"), nil, baseappPageCreateBody(runtime))
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"page": data, "created": true}, nil)
	return nil
}

func executeBaseappPageRename(runtime *common.RuntimeContext) error {
	if err := ensureUniqueAppPageName(runtime, strings.TrimSpace(runtime.Str("name")), runtime.Str("page-id")); err != nil {
		return err
	}
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	data, err := baseV3Call(runtime, "PATCH", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id")), nil, body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"page": data, "updated": true}, nil)
	return nil
}

func ensureUniqueAppPageName(runtime *common.RuntimeContext, name, excludePageID string) error {
	pageToken := ""
	for {
		params := map[string]interface{}{"page_size": 100}
		if pageToken != "" {
			params["page_token"] = pageToken
		}
		data, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token"), "pages"), params, nil)
		if err != nil {
			return err
		}
		for _, page := range appPageItems(data) {
			pageID := firstNonEmpty(common.GetString(page, "page_id"), common.GetString(page, "id"))
			if pageID == strings.TrimSpace(excludePageID) {
				continue
			}
			if strings.EqualFold(strings.TrimSpace(common.GetString(page, "name")), name) {
				return errs.NewValidationError(errs.SubtypeInvalidArgument, "同一应用内 Page 名称必须唯一，已存在名为 %q 的页面", name).WithParam("--name")
			}
		}
		hasMore, _ := data["has_more"].(bool)
		pageToken = firstNonEmpty(common.GetString(data, "page_token"), common.GetString(data, "next_page_token"))
		if !hasMore || pageToken == "" {
			return nil
		}
	}
}

func appPageItems(data map[string]interface{}) []map[string]interface{} {
	for _, key := range []string{"items", "pages"} {
		raw, ok := data[key].([]interface{})
		if !ok {
			continue
		}
		items := make([]map[string]interface{}, 0, len(raw))
		for _, item := range raw {
			if page, ok := item.(map[string]interface{}); ok {
				items = append(items, page)
			}
		}
		return items
	}
	return nil
}

func executeBaseappPageDelete(runtime *common.RuntimeContext) error {
	_, err := baseV3Call(runtime, "DELETE", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id")), nil, nil)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"deleted": true, "page_id": runtime.Str("page-id")}, nil)
	return nil
}

// ── App block: dry-run ───────────────────────────────────────────────

func dryRunAppBlockList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	params := pagingParams(runtime)
	if blockType := strings.TrimSpace(runtime.Str("type")); blockType != "" {
		params["type"] = blockType
	}
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/base_apps/:app_token/pages/:page_id/blocks").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Params(params)
}

func dryRunAppBlockGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/base_apps/:app_token/pages/:page_id/blocks/:block_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Set("block_id", runtime.Str("block-id")).
		Params(userIDTypeParams(runtime))
}

func dryRunAppBlockCreate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	body, err := appBlockBody(runtime, true)
	if err != nil {
		body = map[string]interface{}{}
	}
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/base_apps/:app_token/pages/:page_id/blocks").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Params(userIDTypeParams(runtime)).
		Body(body)
}

func dryRunAppBlockUpdate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	body, err := appBlockBody(runtime, false)
	if err != nil {
		body = map[string]interface{}{}
	}
	return common.NewDryRunAPI().
		PATCH("/open-apis/base/v3/base_apps/:app_token/pages/:page_id/blocks/:block_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Set("block_id", runtime.Str("block-id")).
		Params(userIDTypeParams(runtime)).
		Body(body)
}

// ── App block: execute ───────────────────────────────────────────────

func executeAppBlockList(runtime *common.RuntimeContext) error {
	params := pagingParams(runtime)
	if blockType := strings.TrimSpace(runtime.Str("type")); blockType != "" {
		params["type"] = blockType
	}
	data, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks"), params, nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeAppBlockGet(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks", runtime.Str("block-id")), userIDTypeParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"block": data}, nil)
	return nil
}

func executeAppBlockCreate(runtime *common.RuntimeContext) error {
	if strings.EqualFold(strings.TrimSpace(runtime.Str("type")), "list") {
		if err := validateListBaseWorkspace(runtime); err != nil {
			return err
		}
	}
	body, err := appBlockBody(runtime, true)
	if err != nil {
		return err
	}
	data, err := baseV3Call(runtime, "POST", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks"), userIDTypeParams(runtime), body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"block": data, "created": true}, nil)
	return nil
}

func validateListBaseWorkspace(runtime *common.RuntimeContext) error {
	raw := strings.TrimSpace(runtime.Str("data-config"))
	if raw == "" {
		return nil
	}
	cfg, err := parseJSONObject(newParseCtx(runtime), raw, "data-config")
	if err != nil {
		return err
	}
	baseToken := strings.TrimSpace(common.GetString(cfg, "base_token"))
	if baseToken == "" {
		return nil
	}
	app, err := baseV3Call(runtime, "GET", baseV3Path("base_apps", runtime.Str("app-token")), nil, nil)
	if err != nil {
		return err
	}
	if appRefContainsBase(app["ref"], baseToken) {
		return nil
	}
	workspaceToken := firstNonEmpty(
		strings.TrimSpace(common.GetString(app, "workspace_token")),
		strings.TrimSpace(common.GetString(app, "workspace_id")),
	)
	if workspaceToken == "" {
		return errs.NewValidationError(errs.SubtypeInvalidArgument, "无法确认列表 Base 与 App 是否位于同一 Workspace：App 响应缺少 workspace_token").WithParam("--data-config")
	}
	pageToken := ""
	for {
		params := map[string]interface{}{"page_size": 100, "type": "base"}
		if pageToken != "" {
			params["page_token"] = pageToken
		}
		entities, err := baseV3Call(runtime, "GET", baseV3Path("workspaces", workspaceToken, "entities"), params, nil)
		if err != nil {
			return err
		}
		if workspaceContainsBase(entities, baseToken) {
			return nil
		}
		hasMore, _ := entities["has_more"].(bool)
		pageToken = firstNonEmpty(common.GetString(entities, "page_token"), common.GetString(entities, "next_page_token"))
		if !hasMore || pageToken == "" {
			break
		}
	}
	return errs.NewValidationError(errs.SubtypeInvalidArgument, "列表组件只能选择 App 所在 Workspace 内的一个 Base；%s 不在当前 Workspace", baseToken).WithParam("--data-config")
}

func appRefContainsBase(raw interface{}, baseToken string) bool {
	switch refs := raw.(type) {
	case map[string]interface{}:
		_, ok := refs[baseToken]
		return ok
	case map[string][]string:
		_, ok := refs[baseToken]
		return ok
	default:
		return false
	}
}

func workspaceContainsBase(data map[string]interface{}, baseToken string) bool {
	for _, key := range []string{"items", "entities"} {
		items, _ := data[key].([]interface{})
		for _, raw := range items {
			entity, _ := raw.(map[string]interface{})
			token := firstNonEmpty(common.GetString(entity, "token"), common.GetString(entity, "entity_token"))
			if strings.TrimSpace(token) == baseToken {
				return true
			}
		}
	}
	return false
}

func executeAppBlockUpdate(runtime *common.RuntimeContext) error {
	body, err := appBlockBody(runtime, false)
	if err != nil {
		return err
	}
	data, err := baseV3Call(runtime, "PATCH", baseV3Path("base_apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks", runtime.Str("block-id")), userIDTypeParams(runtime), body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"block": data, "updated": true}, nil)
	return nil
}
