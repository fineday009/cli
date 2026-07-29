// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package base

import (
	"context"
	"net/http"
	"strings"

	"github.com/larksuite/cli/shortcuts/common"
	larkcore "github.com/larksuite/oapi-sdk-go/v3/core"
)

const appTokenPersistHeader = "rpc-persist-x-base-apptoken"

var BaseAppBlockGetData = common.Shortcut{
	Service:     "base",
	Command:     "+app-block-get-data",
	Description: "Get computed data for a BaseApp page chart block",
	Risk:        "read",
	Scopes:      []string{"base:appmode_block:read"},
	AuthTypes:   authTypes(),
	Flags: []common.Flag{
		appTokenFlag(true),
		baseTokenFlag(true),
		{Name: "block-id", Desc: "chart_token returned by the App chart component; this endpoint identifies the chart by chart_token", Required: true},
	},
	Tips: []string{
		"lark-cli base +app-block-get-data --app-token <app_token> --base-token <base_token> --block-id <chart_token>",
		"Despite the flag name, --block-id must be the chart_token from +app-block-list/get, not the component block_id.",
		"Read --base-token from the chart block data_config.base_token; do not choose an arbitrary +app-get ref key when the app references multiple Bases.",
		"The response uses the same computed chart data protocol as +dashboard-block-get-data.",
		"List and text blocks have no computed data; use +app-block-get for their metadata instead.",
	},
	DryRun: func(ctx context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
		return dryRunAppBlockGetData(ctx, runtime)
	},
	Execute: func(ctx context.Context, runtime *common.RuntimeContext) error {
		return executeAppBlockGetData(runtime)
	},
}

func dryRunAppBlockGetData(_ context.Context, runtime *common.RuntimeContext) *common.DryRunAPI {
	return common.NewDryRunAPI().
		GET("/open-apis/base/v3/bases/:base_token/dashboards/blocks/:block_id/data").
		Set("base_token", runtime.Str("base-token")).
		Set("block_id", runtime.Str("block-id")).
		Header(appTokenPersistHeader, strings.TrimSpace(runtime.Str("app-token")))
}

func executeAppBlockGetData(runtime *common.RuntimeContext) error {
	req := &larkcore.ApiReq{
		HttpMethod: "GET",
		ApiPath:    baseV3Path("bases", runtime.Str("base-token"), "dashboards", "blocks", runtime.Str("block-id"), "data"),
	}
	resp, err := runtime.DoAPI(req, larkcore.WithHeaders(http.Header{
		appTokenPersistHeader: []string{strings.TrimSpace(runtime.Str("app-token"))},
	}))
	if err != nil {
		return err
	}
	data, err := runtime.ClassifyAPIResponse(resp)
	if err != nil {
		return err
	}
	runtime.Out(data, nil)
	return nil
}
