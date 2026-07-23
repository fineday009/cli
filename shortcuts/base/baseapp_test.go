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

	addRT := newBaseTestRuntime(map[string]string{"workspace-token": "ws_x", "type": "base", "token": "bascn_1"}, map[string]bool{"to-last": true}, nil)
	assertDryRunContains(t, dryRunWorkspaceEntityAdd(ctx, addRT), "POST /open-apis/base/v3/workspaces/ws_x/entities", `"entity_type":"base"`, `"token":"bascn_1"`, `"to_last":true`)

	removeRT := newBaseTestRuntime(map[string]string{"workspace-token": "ws_x", "entity-id": "789"}, nil, nil)
	assertDryRunContains(t, dryRunWorkspaceEntityRemove(ctx, removeRT), "DELETE /open-apis/base/v3/workspaces/ws_x/entities/789")
}

func TestDryRunBaseappOps(t *testing.T) {
	ctx := context.Background()

	createRT := newBaseTestRuntime(map[string]string{"name": "Sales app", "workspace-token": "ws_x", "base-name": "Sales data", "table-name": "Orders"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappCreate(ctx, createRT),
		"POST /open-apis/base/v3/apps",
		`"name":"Sales app"`,
		`"workspace_token":"ws_x"`,
		`"base":{"name":"Sales data","table_name":"Orders"}`,
	)

	minimalCreateRT := newBaseTestRuntime(map[string]string{"name": "Blank app"}, nil, nil)
	if out := dryRunBaseappCreate(ctx, minimalCreateRT).Format(); strings.Contains(out, `"base"`) || strings.Contains(out, "workspace_token") {
		t.Fatalf("blank create must not send base/workspace keys:\n%s", out)
	}

	getRT := newBaseTestRuntime(map[string]string{"app-token": "app_x"}, map[string]bool{"with-pages": true}, nil)
	assertDryRunContains(t, dryRunBaseappGet(ctx, getRT), "GET /open-apis/base/v3/apps/app_x", "with_pages=true")

	renameRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "name": "New name"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappRename(ctx, renameRT), "PATCH /open-apis/base/v3/apps/app_x", `"name":"New name"`)
}

func TestDryRunBaseappPageOps(t *testing.T) {
	ctx := context.Background()

	listRT := newBaseTestRuntime(map[string]string{"app-token": "app_x"}, nil, map[string]int{"page-size": 100})
	assertDryRunContains(t, dryRunBaseappPageList(ctx, listRT), "GET /open-apis/base/v3/apps/app_x/pages", "page_size=100")

	getRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1"}, map[string]bool{"with-components": true}, nil)
	assertDryRunContains(t, dryRunBaseappPageGet(ctx, getRT), "GET /open-apis/base/v3/apps/app_x/pages/pg_1", "with_components=true")

	createRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "name": "Overview", "parent-page-id": "pg_root"}, map[string]bool{"to-last": true}, nil)
	assertDryRunContains(t, dryRunBaseappPageCreate(ctx, createRT), "POST /open-apis/base/v3/apps/app_x/pages", `"name":"Overview"`, `"parent_page_id":"pg_root"`, `"to_last":true`)

	renameRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1", "name": "Sales"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappPageRename(ctx, renameRT), "PATCH /open-apis/base/v3/apps/app_x/pages/pg_1", `"name":"Sales"`)

	deleteRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1"}, nil, nil)
	assertDryRunContains(t, dryRunBaseappPageDelete(ctx, deleteRT), "DELETE /open-apis/base/v3/apps/app_x/pages/pg_1")
}

func TestDryRunAppBlockOps(t *testing.T) {
	ctx := context.Background()

	listRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1", "type": "line"}, nil, map[string]int{"page-size": 20})
	assertDryRunContains(t, dryRunAppBlockList(ctx, listRT), "GET /open-apis/base/v3/apps/app_x/pages/pg_1/blocks", "type=line", "page_size=20")

	getRT := newBaseTestRuntime(map[string]string{"app-token": "app_x", "page-id": "pg_1", "block-id": "wid_1", "user-id-type": "open_id"}, nil, nil)
	assertDryRunContains(t, dryRunAppBlockGet(ctx, getRT), "GET /open-apis/base/v3/apps/app_x/pages/pg_1/blocks/wid_1", "user_id_type=open_id")

	createRT := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"name":        "Sales by month",
		"type":        "line",
		"data-config": `{"table_name":"Orders","show_title":true,"series":[{"field_name":"Amount","rollup":"SUM"}]}`,
		"position":    `{"x":0,"y":0,"w":12,"h":8}`,
	}, nil, nil)
	assertDryRunContains(t, dryRunAppBlockCreate(ctx, createRT),
		"POST /open-apis/base/v3/apps/app_x/pages/pg_1/blocks",
		`"type":"line"`,
		`"name":"Sales by month"`,
		`"show_title":true`,
		`"position":{"h":8,"w":12,"x":0,"y":0}`,
	)
	if out := dryRunAppBlockCreate(ctx, createRT).Format(); strings.Contains(out, `"data_config":{"show_title"`) {
		t.Fatalf("show_title must be lifted out of data_config:\n%s", out)
	}

	updateRT := newBaseTestRuntime(map[string]string{
		"app-token":   "app_x",
		"page-id":     "pg_1",
		"block-id":    "wid_1",
		"name":        "Monthly sales",
		"data-config": `{"filter":{"conjunction":"and","conditions":[]}}`,
	}, nil, nil)
	updateDR := dryRunAppBlockUpdate(ctx, updateRT)
	assertDryRunContains(t, updateDR, "PATCH /open-apis/base/v3/apps/app_x/pages/pg_1/blocks/wid_1", `"name":"Monthly sales"`, `"conjunction":"and"`)
	if out := updateDR.Format(); strings.Contains(out, `"type"`) {
		t.Fatalf("update must not send a block type:\n%s", out)
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
		"+workspace-entity-remove": {BaseWorkspaceEntityRemove, "high-risk-write", "base:workspace:write"},
		"+baseapp-page-delete":     {BaseAppPageDelete, "high-risk-write", "base:appmode_page:delete"},
		"+baseapp-create":          {BaseAppCreate, "write", "base:appmode:create"},
		"+app-block-create":        {BaseAppBlockCreate, "write", "base:appmode_block:create"},
		"+app-block-get-data":      {BaseAppBlockGetData, "read", "base:dashboard:read"},
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
		problems := validateBlockDataConfig("standardList", map[string]interface{}{
			"table_id": "tblx",
			"view_id":  "viwx",
			"fields":   []interface{}{"fldx"},
			"sort":     []interface{}{map[string]interface{}{"field_id": "fldx", "order": "asc"}},
			"display":  map[string]interface{}{"show_title": true},
		})
		if len(problems) != 0 {
			t.Fatalf("problems=%v", problems)
		}
	})

	t.Run("requires a data source", func(t *testing.T) {
		problems := validateBlockDataConfig("cardList", map[string]interface{}{})
		if len(problems) != 1 || !strings.Contains(problems[0], "table_id") {
			t.Fatalf("problems=%v", problems)
		}
	})

	t.Run("rejects a bad sort order", func(t *testing.T) {
		problems := validateBlockDataConfig("groupedList", map[string]interface{}{
			"table_name": "Orders",
			"sort":       []interface{}{map[string]interface{}{"field_id": "fldx", "order": "up"}},
		})
		if len(problems) != 1 || !strings.Contains(problems[0], "sort[0].order") {
			t.Fatalf("problems=%v", problems)
		}
	})

	t.Run("does not apply chart rules to list blocks", func(t *testing.T) {
		problems := validateBlockDataConfig("detailList", map[string]interface{}{"table_id": "tblx"})
		for _, problem := range problems {
			if strings.Contains(problem, "series") || strings.Contains(problem, "count_all") {
				t.Fatalf("chart rule leaked into list validation: %v", problems)
			}
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

func TestParseBlockPositionRejectsNonNumeric(t *testing.T) {
	rt := newBaseTestRuntime(map[string]string{"position": `{"x":"0","y":0,"w":12,"h":8}`}, nil, nil)
	if _, err := parseBlockPosition(rt); err == nil || !strings.Contains(err.Error(), "position.x") {
		t.Fatalf("err=%v", err)
	}

	okRT := newBaseTestRuntime(map[string]string{"position": `{"x":0,"y":0,"w":12,"h":8}`}, nil, nil)
	position, err := parseBlockPosition(okRT)
	if err != nil || position == nil {
		t.Fatalf("position=%v err=%v", position, err)
	}
}

func TestValidateWorkspaceOrderingRejectsBoth(t *testing.T) {
	rt := newBaseTestRuntime(map[string]string{"prev-entity-id": "456"}, map[string]bool{"to-last": true}, nil)
	if err := validateWorkspaceOrdering(rt, "prev-entity-id"); err == nil || !strings.Contains(err.Error(), "to-last") {
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
