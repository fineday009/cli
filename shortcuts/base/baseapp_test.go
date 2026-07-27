// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"strings"
	"testing"

	"github.com/larksuite/cli/shortcuts/common"
)

func TestDryRunWorkspaceOps(t *testing.T) {
	ctx := context.Background()

	createRT := newBaseTestRuntime(map[string]string{"name": "Growth", "icon": "icon_1"}, nil, nil)
	assertDryRunContains(t, dryRunWorkspaceCreate(ctx, createRT), "POST /open-apis/base/v3/workspaces", `"name":"Growth"`, `"icon":"icon_1"`)

	listRT := newBaseTestRuntime(map[string]string{"workspace-token": "ws_x", "type": "BaseApp"}, nil, map[string]int{"page-size": 50})
	assertDryRunContains(t, dryRunWorkspaceEntityList(ctx, listRT), "GET /open-apis/base/v3/workspaces/ws_x/entities", "page_size=50", "type=baseapp")

	moveInRT := newBaseTestRuntime(map[string]string{"workspace-token": "ws_x", "entity-token": "bascn_1"}, nil, nil)
	assertDryRunContains(t, dryRunWorkspaceMoveIn(ctx, moveInRT), "POST /open-apis/base/v3/workspaces/ws_x/move_in", `"entity_token":"bascn_1"`)
}

func TestDryRunBaseappOps(t *testing.T) {
	ctx := context.Background()

	createRT := newBaseTestRuntime(map[string]string{"name": "Sales app", "workspace-token": "ws_x", "base-name": "Sales data", "table-name": "Orders"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappCreate(ctx, createRT),
		"POST /open-apis/base/v3/base_apps",
		`"name":"Sales app"`,
		`"workspace_token":"ws_x"`,
		"POST /open-apis/base/v3/bases",
		`"name":"Sales data"`,
		"POST /open-apis/base/v3/workspaces/ws_x/move_in",
		`"entity_token"`,
	)

	minimalCreateRT := newBaseTestRuntime(map[string]string{"name": "Blank app"}, nil, nil)
	minimalOut := dryRunBaseappCreate(ctx, minimalCreateRT).Format()
	for _, want := range []string{
		`POST /open-apis/base/v3/workspaces`,
		`"name":"Blank app"`,
		`"workspace_token"`,
		`created_workspace_token`,
	} {
		if !strings.Contains(minimalOut, want) {
			t.Fatalf("app create without workspace token must contain %q:\n%s", want, minimalOut)
		}
	}
	if strings.Contains(minimalOut, `"base_token"`) {
		t.Fatalf("app create must not expose a base-token input:\n%s", minimalOut)
	}

	getRT := newBaseTestRuntime(map[string]string{"app-token": "app_x"}, map[string]bool{"with-pages": true}, nil)
	assertDryRunContains(t, dryRunBaseappGet(ctx, getRT), "GET /open-apis/base/v3/base_apps/app_x", "with_pages=true")

	renameRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "name": "New name"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappRename(ctx, renameRT), "PATCH /open-apis/drive/v1/files/app_x", "type=bitable", `"new_title":"New name"`)
}

func TestAppCreateDoesNotExposeBaseToken(t *testing.T) {
	for _, flag := range BaseAppCreate.Flags {
		if flag.Name == "base-token" {
			t.Fatal("+app-create must not expose --base-token")
		}
	}
}

func TestDryRunBaseappPageOps(t *testing.T) {
	ctx := context.Background()

	listRT := newBaseTestRuntime(map[string]string{"app-token": "app_x"}, nil, map[string]int{"page-size": 100})
	assertDryRunContains(t, dryRunBaseappPageList(ctx, listRT), "GET /open-apis/base/v3/base_apps/app_x/pages", "page_size=100")

	getRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1"}, map[string]bool{"with-components": true}, nil)
	assertDryRunContains(t, dryRunBaseappPageGet(ctx, getRT), "GET /open-apis/base/v3/base_apps/app_x/pages/pg_1", "with_components=true")

	createRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "name": "Overview", "parent-page-id": "pg_root"}, map[string]bool{"to-last": true}, nil)
	assertDryRunContains(t, dryRunBaseappPageCreate(ctx, createRT), "POST /open-apis/base/v3/base_apps/app_x/pages", `"name":"Overview"`, `"parent_page_id":"pg_root"`, `"to_last":true`)

	renameRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1", "name": "Sales"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappPageRename(ctx, renameRT), "PATCH /open-apis/base/v3/base_apps/app_x/pages/pg_1", `"name":"Sales"`)

	deleteRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappPageDelete(ctx, deleteRT), "DELETE /open-apis/base/v3/base_apps/app_x/pages/pg_1")
}

func TestDryRunAppBlockOps(t *testing.T) {
	ctx := context.Background()

	listRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1", "type": "line"}, nil, map[string]int{"page-size": 20})
	assertDryRunContains(t, dryRunAppBlockList(ctx, listRT), "GET /open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks", "type=line", "page_size=20")

	getRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1", "block-id": "wid_1", "user-id-type": "open_id"}, nil, nil)
	assertDryRunContains(t, dryRunAppBlockGet(ctx, getRT), "GET /open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks/wid_1", "user_id_type=open_id")

	createRT := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"name":        "Sales by month",
		"type":        "line",
		"data-config": `{"base_token":"basx","data_sources":[{"table_name":"Orders","series":[{"field_name":"Amount","rollup":"SUM"}]}]}`,
	}, nil, nil)
	assertDryRunContains(t, dryRunAppBlockCreate(ctx, createRT),
		"POST /open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks",
		`"type":"line"`,
		`"name":"Sales by month"`,
		`"base_token":"basx"`,
		`"data_sources"`,
	)

	listCreateRT := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"name":        "Orders",
		"type":        "list",
		"sub-type":    "card",
		"data-config": `{"base_token":"bas_x","table_name":"Orders","fields":[],"card_config":{}}`,
	}, nil, nil)
	assertDryRunContains(t, dryRunAppBlockCreate(ctx, listCreateRT), `"type":"list"`, `"sub_type":"card"`, `"base_token":"bas_x"`)

	standardListRT := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"name":        "Orders",
		"type":        "list",
		"data-config": `{"base_token":"bas_x","table_name":"Orders"}`,
	}, nil, nil)
	if out := dryRunAppBlockCreate(ctx, standardListRT).Format(); strings.Contains(out, `"sub_type"`) {
		t.Fatalf("default standard sub_type must be omitted:\n%s", out)
	}

	updateRT := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"block-id":    "wid_1",
		"name":        "Monthly sales",
		"data-config": `{"filter":{"conjunction":"and","conditions":[]}}`,
	}, nil, nil)
	updateDR := dryRunAppBlockUpdate(ctx, updateRT)
	assertDryRunContains(t, updateDR, "PATCH /open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks/wid_1", `"name":"Monthly sales"`, `"conjunction":"and"`)
	if out := updateDR.Format(); strings.Contains(out, `"type"`) {
		t.Fatalf("update must not send a block type:\n%s", out)
	}
	if out := updateDR.Format(); strings.Contains(out, `"table_name"`) || strings.Contains(out, `"series"`) {
		t.Fatalf("update must not inject omitted data_config fields:\n%s", out)
	}
}

// richText is the CLI-facing alias for the rich-text widget; on the wire the
// API type is "text", so the request body must carry "text".
func TestAppRichTextTypeMapsToText(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"name":        "说明",
		"type":        "richText",
		"data-config": `{"text":"hi"}`,
	}, nil, nil)
	dr := dryRunAppBlockCreate(ctx, rt)
	assertDryRunContains(t, dr, `"type":"text"`, `"text":"hi"`)
	if out := dr.Format(); strings.Contains(out, `"richText"`) {
		t.Fatalf("richText must map to the wire type text:\n%s", out)
	}
}

// +app-block-get-data is a thin wrapper: it must hit exactly the same method
// and path as +dashboard-block-get-data, and must take --base-token instead of
// the --app-token every other +app-block-* command uses.
func TestAppBlockGetDataMirrorsDashboard(t *testing.T) {
	ctx := context.Background()
	rt := newBaseTestRuntime(map[string]string{"base-token": "app_x", "block-id": "blk_chart"}, nil, nil)

	appOut := BaseAppBlockGetData.DryRun(ctx, rt).Format()
	dashboardOut := BaseDashboardBlockGetData.DryRun(ctx, rt).Format()
	if appOut != dashboardOut {
		t.Fatalf("dry-run drifted from dashboard\napp:\n%s\ndashboard:\n%s", appOut, dashboardOut)
	}
	if !strings.Contains(appOut, "GET /open-apis/base/v3/bases/app_x/dashboards/blocks/blk_chart/data") {
		t.Fatalf("unexpected path:\n%s", appOut)
	}
}

func TestAppBlockGetDataRequiredFlags(t *testing.T) {
	required := map[string]bool{}
	for _, flag := range BaseAppBlockGetData.Flags {
		if flag.Required {
			required[flag.Name] = true
		}
	}
	if !required["base-token"] || !required["block-id"] {
		t.Fatalf("required flags=%v want base-token and block-id", required)
	}
	if required["app-token"] || required["page-id"] {
		t.Fatalf("app-token/page-id must stay optional compatibility flags: %v", required)
	}
}

// The app and dashboard command spaces must not cross: dashboard commands never
// take --app-token, and app block commands never take --dashboard-id.
func TestAppAndDashboardCommandSpacesDoNotCross(t *testing.T) {
	appBlockCommands := map[string]bool{
		"+app-block-list": true, "+app-block-get": true,
		"+app-block-create": true, "+app-block-update": true,
	}
	for _, shortcut := range Shortcuts() {
		for _, flag := range shortcut.Flags {
			if strings.HasPrefix(shortcut.Command, "+dashboard-") && flag.Name == "app-token" {
				t.Fatalf("%s must not accept --app-token", shortcut.Command)
			}
			if appBlockCommands[shortcut.Command] && flag.Name == "dashboard-id" {
				t.Fatalf("%s must not accept --dashboard-id", shortcut.Command)
			}
		}
	}
}

func TestBaseappRisksAndScopes(t *testing.T) {
	cases := map[string]struct {
		shortcut common.Shortcut
		risk     string
		scope    string
	}{
		"+workspace-move-in":  {BaseWorkspaceMoveIn, "write", "base:workspace:update"},
		"+app-page-delete":    {BaseAppPageDelete, "high-risk-write", "base:appmode_page:delete"},
		"+app-rename":         {BaseAppRename, "write", "base:appmode:update"},
		"+app-block-create":   {BaseAppBlockCreate, "write", "base:appmode_block:create"},
		"+app-block-get-data": {BaseAppBlockGetData, "read", "base:dashboard:read"},
	}
	if got := strings.Join(BaseAppCreate.Scopes, ","); got != "base:appmode:create,base:workspace:create,base:workspace:update" {
		t.Errorf("+app-create scopes=%v", BaseAppCreate.Scopes)
	}
	for name, tc := range cases {
		if tc.shortcut.Risk != tc.risk {
			t.Errorf("%s risk=%q want=%q", name, tc.shortcut.Risk, tc.risk)
		}
		if len(tc.shortcut.Scopes) != 1 || tc.shortcut.Scopes[0] != tc.scope {
			t.Errorf("%s scopes=%v want=[%s]", name, tc.shortcut.Scopes, tc.scope)
		}
	}
}

func TestValidateListDataConfig(t *testing.T) {
	t.Run("accepts a minimal list config", func(t *testing.T) {
		problems := validateAppListDataConfig("standard", map[string]interface{}{
			"base_token": "basx",
			"table_name": "Orders",
		})
		if len(problems) != 0 {
			t.Fatalf("problems=%v", problems)
		}
	})

	t.Run("accepts omitted optional fields for every subtype", func(t *testing.T) {
		for _, subType := range appListSubTypes {
			problems := validateAppListDataConfig(subType, map[string]interface{}{
				"base_token": "basx",
				"table_name": "Orders",
			})
			if len(problems) != 0 {
				t.Fatalf("%s problems=%v", subType, problems)
			}
		}
	})

	t.Run("accepts explicitly empty optional arrays", func(t *testing.T) {
		for _, tc := range []struct {
			subType string
			key     string
		}{
			{subType: "standard", key: "columns"},
			{subType: "grouped", key: "group_by"},
			{subType: "collapsible", key: "sort_by"},
			{subType: "card", key: "fields"},
			{subType: "detail", key: "fields"},
		} {
			cfg := map[string]interface{}{
				"base_token": "basx",
				"table_name": "Orders",
				tc.key:       []interface{}{},
			}
			if problems := validateAppListDataConfig(tc.subType, cfg); len(problems) != 0 {
				t.Fatalf("%s.%s problems=%v", tc.subType, tc.key, problems)
			}
		}
	})

	t.Run("requires a data source", func(t *testing.T) {
		problems := validateAppListDataConfig("card", map[string]interface{}{})
		if len(problems) != 2 || !strings.Contains(strings.Join(problems, " "), "base_token") {
			t.Fatalf("problems=%v", problems)
		}
	})

	t.Run("rejects fields from another subtype", func(t *testing.T) {
		problems := validateAppListDataConfig("grouped", map[string]interface{}{
			"base_token":  "basx",
			"table_name":  "Orders",
			"card_config": map[string]interface{}{},
		})
		if len(problems) != 1 || !strings.Contains(problems[0], "card_config") {
			t.Fatalf("problems=%v", problems)
		}
	})

	t.Run("does not apply chart rules to list blocks", func(t *testing.T) {
		problems := validateAppListDataConfig("detail", map[string]interface{}{"base_token": "basx", "table_name": "Orders"})
		for _, problem := range problems {
			if strings.Contains(problem, "series") || strings.Contains(problem, "count_all") {
				t.Fatalf("chart rule leaked into list validation: %v", problems)
			}
		}
	})

	t.Run("validates optional nested fields only when present", func(t *testing.T) {
		tests := []struct {
			name    string
			subType string
			extra   map[string]interface{}
			want    string
		}{
			{
				name:    "filter requires conjunction",
				subType: "standard",
				extra:   map[string]interface{}{"filter": map[string]interface{}{"conditions": []interface{}{map[string]interface{}{"field_name": "Status", "operator": "is", "value": "Open"}}}},
				want:    "filter.conjunction",
			},
			{
				name:    "filter requires conditions",
				subType: "standard",
				extra:   map[string]interface{}{"filter": map[string]interface{}{"conjunction": "and"}},
				want:    "filter.conditions",
			},
			{
				name:    "sort item requires field name",
				subType: "standard",
				extra:   map[string]interface{}{"sort_by": []interface{}{map[string]interface{}{"order": "asc"}}},
				want:    "sort_by[0].field_name",
			},
			{
				name:    "group order is enumerated",
				subType: "grouped",
				extra:   map[string]interface{}{"group_by": []interface{}{map[string]interface{}{"field_name": "Status", "order": "up"}}},
				want:    "group_by[0].order",
			},
			{
				name:    "field column requires field name",
				subType: "standard",
				extra:   map[string]interface{}{"columns": []interface{}{map[string]interface{}{"type": "field"}}},
				want:    "columns[0].field_name",
			},
			{
				name:    "combined column requires field names",
				subType: "standard",
				extra:   map[string]interface{}{"columns": []interface{}{map[string]interface{}{"type": "combined", "field_names": []interface{}{}}}},
				want:    "columns[0].field_names",
			},
			{
				name:    "card fields are strings",
				subType: "card",
				extra:   map[string]interface{}{"fields": []interface{}{123}},
				want:    "fields[0]",
			},
			{
				name:    "card config values are strings",
				subType: "card",
				extra:   map[string]interface{}{"card_config": map[string]interface{}{"title_field_name": true}},
				want:    "card_config.title_field_name",
			},
			{
				name:    "detail config values are strings",
				subType: "detail",
				extra:   map[string]interface{}{"detail_config": map[string]interface{}{"image_field_name": true}},
				want:    "detail_config.image_field_name",
			},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				cfg := map[string]interface{}{"base_token": "basx", "table_name": "Orders"}
				for key, value := range tc.extra {
					cfg[key] = value
				}
				problems := validateAppListDataConfig(tc.subType, cfg)
				if !strings.Contains(strings.Join(problems, " "), tc.want) {
					t.Fatalf("problems=%v want substring %q", problems, tc.want)
				}
			})
		}
	})

	t.Run("accepts valid optional nested fields", func(t *testing.T) {
		problems := validateAppListDataConfig("standard", map[string]interface{}{
			"base_token": "basx",
			"table_name": "Orders",
			"filter": map[string]interface{}{
				"conjunction": "and",
				"conditions": []interface{}{
					map[string]interface{}{"field_name": "Status", "operator": "is", "value": "Open"},
				},
			},
			"sort_by": []interface{}{map[string]interface{}{"field_name": "Created", "order": "desc"}},
			"columns": []interface{}{
				map[string]interface{}{"type": "field", "field_name": "Status"},
				map[string]interface{}{"type": "combined", "field_names": []interface{}{"Owner", "Created"}},
			},
		})
		if len(problems) != 0 {
			t.Fatalf("problems=%v", problems)
		}
	})
}

func TestValidateTextDataConfigCoversRichText(t *testing.T) {
	if problems := validateBlockDataConfig("richText", map[string]interface{}{"text": "# Title"}); len(problems) != 0 {
		t.Fatalf("problems=%v", problems)
	}
	problems := validateBlockDataConfig("richText", map[string]interface{}{})
	if len(problems) != 1 || !strings.Contains(problems[0], "richText") {
		t.Fatalf("problems=%v", problems)
	}
	// dashboard 的 text 类型行为保持不变
	dashboardProblems := validateBlockDataConfig("text", map[string]interface{}{})
	if len(dashboardProblems) != 1 || dashboardProblems[0] != "text 类型组件缺少必填字段 text" {
		t.Fatalf("dashboard text message changed: %v", dashboardProblems)
	}
}

func TestValidateAppTextDataConfigIsOptional(t *testing.T) {
	if problems := validateAppBlockDataConfig("richText", map[string]interface{}{}); len(problems) != 0 {
		t.Fatalf("problems=%v", problems)
	}
}

func TestValidateAppChartRejectsFieldsOutsideProtocol(t *testing.T) {
	problems := validateAppBlockDataConfig("line", map[string]interface{}{
		"base_token": "basx",
		"data_sources": []interface{}{
			map[string]interface{}{"table_name": "Orders", "count_all": true},
		},
		"show_title": true,
	})
	if !strings.Contains(strings.Join(problems, " "), "show_title") {
		t.Fatalf("problems=%v", problems)
	}
}

func TestValidateWorkspaceOrderingRejectsBoth(t *testing.T) {
	rt := newBaseTestRuntime(map[string]string{"prev-page-id": "456"}, map[string]bool{"to-last": true}, nil)
	if err := validateWorkspaceOrdering(rt, "prev-page-id"); err == nil || !strings.Contains(err.Error(), "to-last") {
		t.Fatalf("err=%v", err)
	}
}

func TestNormalizeEntityType(t *testing.T) {
	if got, err := normalizeEntityType(" BaseApp "); err != nil || got != "baseapp" {
		t.Fatalf("got=%q err=%v", got, err)
	}
	if _, err := normalizeEntityType("sheet"); err == nil {
		t.Fatal("expected validation error for unsupported entity type")
	}
}
