// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/deviceinfo"
	"github.com/larksuite/cli/internal/envvars"
)

func unsetDeviceInfoEnvironment(t *testing.T) {
	t.Helper()
	value, set := os.LookupEnv(envvars.CliDeviceInfoCollection)
	if err := os.Unsetenv(envvars.CliDeviceInfoCollection); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if set {
			_ = os.Setenv(envvars.CliDeviceInfoCollection, value)
		} else {
			_ = os.Unsetenv(envvars.CliDeviceInfoCollection)
		}
	})
}

func TestDeviceInfoCommandOffWritesGlobalConfigAndPreservesProfiles(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetDeviceInfoEnvironment(t)
	originalWorkspace := core.CurrentWorkspace()
	t.Cleanup(func() { core.SetCurrentWorkspace(originalWorkspace) })
	core.SetCurrentWorkspace(core.WorkspaceOpenClaw)

	if err := core.SaveBaseMultiAppConfig(&core.MultiAppConfig{
		CurrentApp: "prod",
		Apps: []core.AppConfig{{
			Name:  "prod",
			AppId: "cli_prod",
		}},
	}); err != nil {
		t.Fatal(err)
	}
	f, stdout, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigDeviceInfoCollection(f)
	cmd.SetArgs([]string{"off"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	var status deviceInfoStatus
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatalf("status JSON: %v\n%s", err, stdout.String())
	}
	if status.Enabled || status.Source != deviceinfo.DeviceInfoCollectionSourceConfig {
		t.Fatalf("status = %+v, want disabled from config", status)
	}
	saved, err := core.LoadBaseMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Apps) != 1 || saved.Apps[0].AppId != "cli_prod" || saved.CurrentApp != "prod" {
		t.Fatalf("saved config = %+v, want profile preserved", saved)
	}
	if saved.DeviceInfoCollection == nil || *saved.DeviceInfoCollection {
		t.Fatalf("saved setting = %v, want false", saved.DeviceInfoCollection)
	}
}

func TestDeviceInfoCommandEnvironmentHasHighestPriority(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv(envvars.CliDeviceInfoCollection, "true")
	disabled := false
	if err := core.SaveBaseMultiAppConfig(&core.MultiAppConfig{
		DeviceInfoCollection: &disabled,
		Apps:                 []core.AppConfig{},
	}); err != nil {
		t.Fatal(err)
	}

	f, stdout, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigDeviceInfoCollection(f)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var status deviceInfoStatus
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Enabled || status.Source != deviceinfo.DeviceInfoCollectionSourceEnvironment {
		t.Fatalf("status = %+v, want enabled from environment", status)
	}
	if !status.Config.Set || status.Config.Value == nil || *status.Config.Value {
		t.Fatalf("config status = %+v, want persisted false shown as overridden", status.Config)
	}
}

func TestDeviceInfoCommandInvalidEnvironmentDoesNotWrite(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv(envvars.CliDeviceInfoCollection, "invalid")
	enabled := true
	if err := core.SaveBaseMultiAppConfig(&core.MultiAppConfig{
		DeviceInfoCollection: &enabled,
		Apps:                 []core.AppConfig{},
	}); err != nil {
		t.Fatal(err)
	}

	f, _, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigDeviceInfoCollection(f)
	cmd.SetArgs([]string{"off"})
	err := cmd.Execute()
	var configErr *errs.ConfigError
	if !errors.As(err, &configErr) || configErr.Subtype != errs.SubtypeInvalidConfig {
		t.Fatalf("Execute() error = %T %v, want invalid ConfigError", err, err)
	}
	saved, loadErr := core.LoadBaseMultiAppConfig()
	if loadErr != nil {
		t.Fatal(loadErr)
	}
	if saved.DeviceInfoCollection == nil || !*saved.DeviceInfoCollection {
		t.Fatalf("setting changed after invalid environment: %v", saved.DeviceInfoCollection)
	}
}

func TestDeviceInfoCommandBypassesExternalCredentialGuard(t *testing.T) {
	f := newConfigFactoryWithExternalProvider(t)
	unsetDeviceInfoEnvironment(t)
	cmd := NewCmdConfig(f)
	cmd.SetArgs([]string{"device-info-collection", "off"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v, device-info-collection must not require built-in credentials", err)
	}
	decision, err := deviceinfo.ResolveDeviceInfoCollection()
	if err != nil || decision.Enabled {
		t.Fatalf("Resolve() = %+v, %v; want disabled", decision, err)
	}
}

func TestDeviceInfoCommandResetRestoresDefault(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetDeviceInfoEnvironment(t)
	if err := deviceinfo.SetDeviceInfoCollection(false); err != nil {
		t.Fatal(err)
	}

	f, stdout, _, _ := cmdutil.TestFactory(t, nil)
	cmd := NewCmdConfigDeviceInfoCollection(f)
	cmd.SetArgs([]string{"--reset"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	var status deviceInfoStatus
	if err := json.Unmarshal(stdout.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if !status.Enabled || status.Source != deviceinfo.DeviceInfoCollectionSourceDefault || status.Config.Set {
		t.Fatalf("status = %+v, want enabled default with no saved preference", status)
	}
}

func TestConfigInitAndRemovePreserveDeviceInfoSetting(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetDeviceInfoEnvironment(t)
	disabled := false
	existing := &core.MultiAppConfig{
		DeviceInfoCollection: &disabled,
		Apps: []core.AppConfig{{
			AppId: "cli_old",
		}},
	}
	f, _, _, _ := cmdutil.TestFactory(t, nil)
	if err := saveInitConfig("", existing, f, "cli_new", core.PlainSecret("secret"), core.BrandFeishu, ""); err != nil {
		t.Fatalf("saveInitConfig() error = %v", err)
	}
	afterInit, err := core.LoadMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if afterInit.DeviceInfoCollection == nil || *afterInit.DeviceInfoCollection {
		t.Fatalf("setting after init = %v, want false", afterInit.DeviceInfoCollection)
	}

	if err := configRemoveRun(&ConfigRemoveOptions{Factory: f}); err != nil {
		t.Fatalf("configRemoveRun() error = %v", err)
	}
	afterRemove, err := core.LoadMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(afterRemove.Apps) != 0 || afterRemove.DeviceInfoCollection == nil || *afterRemove.DeviceInfoCollection {
		t.Fatalf("config after remove = %+v, want settings-only disabled config", afterRemove)
	}
}
