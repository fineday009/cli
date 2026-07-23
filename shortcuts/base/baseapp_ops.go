// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"encoding/json"
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

// parseBlockPosition parses the --position JSON object and validates it against
// the shared position shape {"x":0,"y":0,"w":12,"h":8}.
func parseBlockPosition(runtime *common.RuntimeContext) (map[string]interface{}, error) {
	raw := strings.TrimSpace(runtime.Str("position"))
	if raw == "" {
		return nil, nil
	}
	pc := newParseCtx(runtime)
	parsed, err := parseJSONObject(pc, raw, "position")
	if err != nil {
		return nil, err
	}
	for _, key := range []string{"x", "y", "w", "h"} {
		value, has := parsed[key]
		if !has {
			continue
		}
		switch value.(type) {
		case float64, int, int64, json.Number:
		default:
			return nil, errs.NewValidationError(errs.SubtypeInvalidArgument, "--position.%s 必须是数字", key).WithParam("--position")
		}
	}
	return parsed, nil
}

// appBlockBody builds the shared create/update request body for app page
// blocks. show_title lives at the top level of the OpenAPI body but is
// authored inside --data-config, so it is lifted out here.
func appBlockBody(runtime *common.RuntimeContext, includeType bool) (map[string]interface{}, error) {
	body := map[string]interface{}{}
	if name := strings.TrimSpace(runtime.Str("name")); name != "" {
		body["name"] = name
	}
	if includeType {
		if blockType := strings.TrimSpace(runtime.Str("type")); blockType != "" {
			body["type"] = blockType
		}
	}
	if raw := strings.TrimSpace(runtime.Str("data-config")); raw != "" {
		pc := newParseCtx(runtime)
		parsed, err := parseJSONObject(pc, raw, "data-config")
		if err != nil {
			return nil, err
		}
		if showTitle, has := parsed["show_title"]; has {
			body["show_title"] = showTitle
			delete(parsed, "show_title")
		}
		body["data_config"] = parsed
	}
	position, err := parseBlockPosition(runtime)
	if err != nil {
		return nil, err
	}
	if position != nil {
		body["position"] = position
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

func workspaceEntityAddBody(runtime *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{}
	if entityType, err := normalizeEntityType(runtime.Str("type")); err == nil && entityType != "" {
		body["entity_type"] = entityType
	}
	if token := strings.TrimSpace(runtime.Str("token")); token != "" {
		body["token"] = token
	}
	if prev := strings.TrimSpace(runtime.Str("prev-entity-id")); prev != "" {
		body["prev_entity_id"] = prev
	}
	if runtime.Bool("to-last") {
		body["to_last"] = true
	}
	return body
}

func dryRunWorkspaceEntityAdd(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/workspaces/:workspace_token/entities").
		Set("workspace_token", runtime.Str("workspace-token")).
		Body(workspaceEntityAddBody(runtime))
}

func dryRunWorkspaceEntityRemove(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		DELETE("/open-apis/base/v3/workspaces/:workspace_token/entities/:entity_id").
		Set("workspace_token", runtime.Str("workspace-token")).
		Set("entity_id", runtime.Str("entity-id"))
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

func executeWorkspaceEntityAdd(runtime *common.RuntimeContext) error {
	if _, err := normalizeEntityType(runtime.Str("type")); err != nil {
		return err
	}
	data, err := baseV3Call(runtime, "POST", baseV3Path("workspaces", runtime.Str("workspace-token"), "entities"), nil, workspaceEntityAddBody(runtime))
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"entity": data, "created": true}, nil)
	return nil
}

func executeWorkspaceEntityRemove(runtime *common.RuntimeContext) error {
	_, err := baseV3Call(runtime, "DELETE", baseV3Path("workspaces", runtime.Str("workspace-token"), "entities", runtime.Str("entity-id")), nil, nil)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"deleted": true, "entity_id": runtime.Str("entity-id")}, nil)
	return nil
}

// ── BaseApp: dry-run ─────────────────────────────────────────────────

func baseappCreateBody(runtime *common.RuntimeContext) map[string]interface{} {
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	if workspaceToken := strings.TrimSpace(runtime.Str("workspace-token")); workspaceToken != "" {
		body["workspace_token"] = workspaceToken
	}
	baseSpec := map[string]interface{}{}
	if baseName := strings.TrimSpace(runtime.Str("base-name")); baseName != "" {
		baseSpec["name"] = baseName
	}
	if tableName := strings.TrimSpace(runtime.Str("table-name")); tableName != "" {
		baseSpec["table_name"] = tableName
	}
	if len(baseSpec) > 0 {
		body["base"] = baseSpec
	}
	return body
}

func dryRunBaseappCreate(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		POST("/open-apis/base/v3/apps").
		Body(baseappCreateBody(runtime))
}

func baseappGetParams(runtime *common.RuntimeContext) map[string]interface{} {
	params := map[string]interface{}{}
	if runtime.Bool("with-pages") {
		params["with_pages"] = true
	}
	if runtime.Bool("with-components") {
		params["with_components"] = true
	}
	return params
}

func dryRunBaseappGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/apps/:app_token").
		Set("app_token", runtime.Str("app-token")).
		Params(baseappGetParams(runtime))
}

func dryRunBaseappRename(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		PATCH("/open-apis/base/v3/apps/:app_token").
		Set("app_token", runtime.Str("app-token")).
		Body(map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))})
}

// ── BaseApp: execute ─────────────────────────────────────────────────

func executeBaseappCreate(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "POST", baseV3Path("apps"), nil, baseappCreateBody(runtime))
	if err != nil {
		return err
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	data["created"] = true
	runtime.Out(data, nil)
	return nil
}

func executeBaseappGet(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("apps", runtime.Str("app-token")), baseappGetParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeBaseappRename(runtime *common.RuntimeContext) error {
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	data, err := baseV3Call(runtime, "PATCH", baseV3Path("apps", runtime.Str("app-token")), nil, body)
	if err != nil {
		return err
	}
	if data == nil {
		data = map[string]interface{}{}
	}
	data["updated"] = true
	runtime.Out(data, nil)
	return nil
}

// ── Page: dry-run ────────────────────────────────────────────────────

func dryRunBaseappPageList(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/apps/:app_token/pages").
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
		GET("/open-apis/base/v3/apps/:app_token/pages/:page_id").
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
		POST("/open-apis/base/v3/apps/:app_token/pages").
		Set("app_token", runtime.Str("app-token")).
		Body(baseappPageCreateBody(runtime))
}

func dryRunBaseappPageRename(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		PATCH("/open-apis/base/v3/apps/:app_token/pages/:page_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Body(map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))})
}

func dryRunBaseappPageDelete(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		DELETE("/open-apis/base/v3/apps/:app_token/pages/:page_id").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id"))
}

// ── Page: execute ────────────────────────────────────────────────────

func executeBaseappPageList(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("apps", runtime.Str("app-token"), "pages"), pagingParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeBaseappPageGet(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id")), baseappPageGetParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"page": data}, nil)
	return nil
}

func executeBaseappPageCreate(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "POST", baseV3Path("apps", runtime.Str("app-token"), "pages"), nil, baseappPageCreateBody(runtime))
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"page": data, "created": true}, nil)
	return nil
}

func executeBaseappPageRename(runtime *common.RuntimeContext) error {
	body := map[string]interface{}{"name": strings.TrimSpace(runtime.Str("name"))}
	data, err := baseV3Call(runtime, "PATCH", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id")), nil, body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"page": data, "updated": true}, nil)
	return nil
}

func executeBaseappPageDelete(runtime *common.RuntimeContext) error {
	_, err := baseV3Call(runtime, "DELETE", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id")), nil, nil)
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
		GET("/open-apis/base/v3/apps/:app_token/pages/:page_id/blocks").
		Set("app_token", runtime.Str("app-token")).
		Set("page_id", runtime.Str("page-id")).
		Params(params)
}

func dryRunAppBlockGet(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/apps/:app_token/pages/:page_id/blocks/:block_id").
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
		POST("/open-apis/base/v3/apps/:app_token/pages/:page_id/blocks").
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
		PATCH("/open-apis/base/v3/apps/:app_token/pages/:page_id/blocks/:block_id").
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
	data, err := baseV3Call(runtime, "GET", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks"), params, nil)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}

func executeAppBlockGet(runtime *common.RuntimeContext) error {
	data, err := baseV3Call(runtime, "GET", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks", runtime.Str("block-id")), userIDTypeParams(runtime), nil)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"block": data}, nil)
	return nil
}

func executeAppBlockCreate(runtime *common.RuntimeContext) error {
	body, err := appBlockBody(runtime, true)
	if err != nil {
		return err
	}
	data, err := baseV3Call(runtime, "POST", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks"), userIDTypeParams(runtime), body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"block": data, "created": true}, nil)
	return nil
}

func executeAppBlockUpdate(runtime *common.RuntimeContext) error {
	body, err := appBlockBody(runtime, false)
	if err != nil {
		return err
	}
	data, err := baseV3Call(runtime, "PATCH", baseV3Path("apps", runtime.Str("app-token"), "pages", runtime.Str("page-id"), "blocks", runtime.Str("block-id")), userIDTypeParams(runtime), body)
	if err != nil {
		return err
	}
	runtime.Out(map[string]interface{}{"block": data, "updated": true}, nil)
	return nil
}
