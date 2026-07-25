// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseWorkspaceDryRun(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+workspace-create", "--name", "Growth")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/workspaces")
		assert.Contains(t, output, `"method": "POST"`)
		assert.Contains(t, output, `"name": "Growth"`)
	})

	t.Run("entity-list", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+workspace-entity-list", "--workspace-token", "ws_x", "--type", "baseapp")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/workspaces/ws_x/entities")
		assert.Contains(t, output, `"method": "GET"`)
		assert.Contains(t, output, "baseapp")
	})

	t.Run("move-in", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+workspace-move-in",
			"--workspace-token", "ws_x", "--entity-token", "bascn_1")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/workspaces/ws_x/move_in")
		assert.Contains(t, output, `"entity_token": "bascn_1"`)
	})

	t.Run("entity-remove", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+workspace-entity-remove", "--workspace-token", "ws_x", "--entity-id", "789")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/workspaces/ws_x/entities/789")
		assert.Contains(t, output, `"method": "DELETE"`)
	})
}

func TestBaseappDryRun(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-create",
			"--name", "Sales app", "--workspace-token", "ws_x", "--base-name", "Sales data", "--table-name", "Orders")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps")
		assert.Contains(t, output, `"method": "POST"`)
		assert.Contains(t, output, "/open-apis/base/v3/bases")
		assert.Contains(t, output, "/open-apis/base/v3/workspaces/ws_x/move_in")
	})

	t.Run("get", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-get", "--app-token", "app_x", "--with-pages")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x")
		assert.Contains(t, output, "with_pages")
	})

	t.Run("rename", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-rename", "--app-token", "app_x", "--name", "New name")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/drive/v1/files/app_x")
		assert.Contains(t, output, `"method": "PATCH"`)
		assert.Contains(t, output, `"new_title": "New name"`)
		assert.Contains(t, output, "bitable")
	})
}

func TestBaseappPageDryRun(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-page-list", "--app-token", "app_x")
		assert.Contains(t, result.Stdout, "/open-apis/base/v3/base_apps/app_x/pages")
	})

	t.Run("get", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-page-get", "--app-token", "app_x", "--page-id", "pg_1", "--with-components")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages/pg_1")
		assert.Contains(t, output, "with_components")
	})

	t.Run("create", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-page-create", "--app-token", "app_x", "--name", "Overview", "--to-last")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages")
		assert.Contains(t, output, `"name": "Overview"`)
		assert.Contains(t, output, `"to_last": true`)
	})

	t.Run("create rejects conflicting ordering flags", func(t *testing.T) {
		result := runBaseDryRun(t, 2, "base", "+app-page-create",
			"--app-token", "app_x", "--name", "Overview", "--prev-page-id", "pg_0", "--to-last")
		assert.Contains(t, result.Stderr, "to-last")
	})

	t.Run("rename", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-page-update", "--app-token", "app_x", "--page-id", "pg_1", "--name", "Sales")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages/pg_1")
		assert.Contains(t, output, `"method": "PATCH"`)
	})

	t.Run("delete", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-page-delete", "--app-token", "app_x", "--page-id", "pg_1")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages/pg_1")
		assert.Contains(t, output, `"method": "DELETE"`)
	})
}

func TestAppBlockDryRun(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-block-list", "--app-token", "app_x", "--page-id", "pg_1", "--type", "line")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks")
		assert.Contains(t, output, "line")
	})

	t.Run("get", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-block-get", "--app-token", "app_x", "--page-id", "pg_1", "--block-id", "wid_1")
		assert.Contains(t, result.Stdout, "/open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks/wid_1")
	})

	t.Run("create chart", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-block-create",
			"--app-token", "app_x", "--page-id", "pg_1",
			"--name", "Sales by month", "--type", "line",
			"--data-config", `{"base_token":"basx","data_sources":[{"table_name":"Orders","series":[{"field_name":"Amount","rollup":"sum"}]}]}`,
			"--position", `{"x":0,"y":0,"w":12,"h":8}`)
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks")
		assert.Contains(t, output, `"method": "POST"`)
		assert.Contains(t, output, `"type": "line"`)
		// App 图表：顶层 base_token + 多数据源 data_sources
		assert.Contains(t, output, `"base_token": "basx"`)
		assert.Contains(t, output, "data_sources")
		// normalizeAppChartDataConfig 把每个数据源的 rollup 归一化为大写
		assert.Contains(t, output, "SUM")
	})

	t.Run("create chart rejects missing base_token", func(t *testing.T) {
		result := runBaseDryRun(t, 2, "base", "+app-block-create",
			"--app-token", "app_x", "--page-id", "pg_1",
			"--name", "Sales by month", "--type", "line",
			"--data-config", `{"data_sources":[{"table_name":"Orders","series":[{"field_name":"Amount","rollup":"SUM"}]}]}`)
		assert.Contains(t, result.Stderr, "base_token")
	})

	t.Run("create list", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-block-create",
			"--app-token", "app_x", "--page-id", "pg_1",
			"--name", "Open orders", "--type", "list", "--sub-type", "standard",
			"--data-config", `{"base_token":"basx","table_name":"Orders","columns":[]}`)
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, `"type": "list"`)
		assert.Contains(t, output, `"sub_type": "standard"`)
		assert.Contains(t, output, "basx")
	})

	t.Run("create rejects an unsupported type", func(t *testing.T) {
		result := runBaseDryRun(t, 2, "base", "+app-block-create",
			"--app-token", "app_x", "--page-id", "pg_1", "--name", "X", "--type", "gantt")
		assert.NotEqual(t, 0, result.ExitCode)
	})

	t.Run("create rejects an invalid chart data_config", func(t *testing.T) {
		result := runBaseDryRun(t, 2, "base", "+app-block-create",
			"--app-token", "app_x", "--page-id", "pg_1", "--name", "X", "--type", "line",
			"--data-config", `{"base_token":"basx","data_sources":[{"series":[{"field_name":"Amount","rollup":"SUM"}]}]}`)
		assert.Contains(t, result.Stderr, "table_name")
	})

	t.Run("create multi-datasource chart", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-block-create",
			"--app-token", "app_x", "--page-id", "pg_1",
			"--name", "Sales vs cost", "--type", "combo",
			"--data-config", `{"base_token":"basx","data_source_mode":"compare","data_sources":[{"table_name":"Sales","series":[{"field_name":"Amount","rollup":"SUM"}]},{"table_name":"Cost","series":[{"field_name":"Cost","rollup":"SUM"}]}],"sort":{"type":"group","order":"asc"}}`)
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, `"type": "combo"`)
		assert.Contains(t, output, `"compare"`)
		assert.Contains(t, output, "Sales")
		assert.Contains(t, output, "Cost")
	})

	t.Run("update", func(t *testing.T) {
		result := runBaseDryRun(t, 0, "base", "+app-block-update",
			"--app-token", "app_x", "--page-id", "pg_1", "--block-id", "wid_1", "--name", "Monthly sales")
		output := strings.TrimSpace(result.Stdout)
		assert.Contains(t, output, "/open-apis/base/v3/base_apps/app_x/pages/pg_1/blocks/wid_1")
		assert.Contains(t, output, `"method": "PATCH"`)
	})
}

// +app-block-get-data is the one command in the group that takes --base-token
// and reuses the dashboard endpoint verbatim.
func TestAppBlockGetDataDryRun(t *testing.T) {
	result := runBaseDryRun(t, 0, "base", "+app-block-get-data", "--base-token", "app_x", "--block-id", "blk_chart")
	output := strings.TrimSpace(result.Stdout)
	assert.Contains(t, output, "/open-apis/base/v3/bases/app_x/dashboards/blocks/blk_chart/data")
	assert.Contains(t, output, `"method": "GET"`)

	missing := runBaseDryRun(t, 2, "base", "+app-block-get-data", "--app-token", "app_x", "--block-id", "blk_chart")
	assert.Contains(t, missing.Stderr, "base-token")
}
