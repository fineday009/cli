// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package deviceinfo

import (
	"errors"
	"os"
	"testing"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/envvars"
)

func unsetCollectionEnvironment(t *testing.T) {
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

func boolPointer(value bool) *bool { return &value }

func TestResolveDefaultsEnabled(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetCollectionEnvironment(t)

	decision, err := ResolveDeviceInfoCollection()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !decision.Enabled || decision.Source != DeviceInfoCollectionSourceDefault {
		t.Fatalf("Resolve() = %+v, want enabled from default", decision)
	}
}

func TestResolveUsesGlobalConfig(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetCollectionEnvironment(t)
	if err := core.SaveBaseMultiAppConfig(&core.MultiAppConfig{
		DeviceInfoCollection: boolPointer(false),
		Apps:                 []core.AppConfig{},
	}); err != nil {
		t.Fatal(err)
	}

	decision, err := ResolveDeviceInfoCollection()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if decision.Enabled || decision.Source != DeviceInfoCollectionSourceConfig {
		t.Fatalf("Resolve() = %+v, want disabled from config", decision)
	}
}

func TestResolveEnvironmentOverridesConfig(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv(envvars.CliDeviceInfoCollection, "yes")
	if err := core.SaveBaseMultiAppConfig(&core.MultiAppConfig{
		DeviceInfoCollection: boolPointer(false),
		Apps:                 []core.AppConfig{},
	}); err != nil {
		t.Fatal(err)
	}

	decision, err := ResolveDeviceInfoCollection()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if !decision.Enabled || decision.Source != DeviceInfoCollectionSourceEnvironment {
		t.Fatalf("Resolve() = %+v, want enabled from environment", decision)
	}
	if decision.ConfigValue == nil || *decision.ConfigValue {
		t.Fatalf("Resolve() config value = %v, want persisted false", decision.ConfigValue)
	}
}

func TestResolveEnvironmentBooleanSpellings(t *testing.T) {
	tests := []struct {
		value string
		want  bool
	}{
		{"true", true}, {"1", true}, {"on", true}, {"YES", true},
		{"false", false}, {"0", false}, {"off", false}, {"No", false},
	}
	for _, test := range tests {
		t.Run(test.value, func(t *testing.T) {
			t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
			t.Setenv(envvars.CliDeviceInfoCollection, test.value)
			decision, err := ResolveDeviceInfoCollection()
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}
			if decision.Enabled != test.want || decision.Source != DeviceInfoCollectionSourceEnvironment {
				t.Fatalf("Resolve() = %+v, want enabled=%v from environment", decision, test.want)
			}
		})
	}
}

func TestResolveRejectsInvalidEnvironmentValue(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	t.Setenv(envvars.CliDeviceInfoCollection, "sometimes")

	_, err := ResolveDeviceInfoCollection()
	if err == nil {
		t.Fatal("Resolve() error = nil, want invalid config error")
	}
	var configErr *errs.ConfigError
	if !errors.As(err, &configErr) {
		t.Fatalf("Resolve() error type = %T, want *errs.ConfigError", err)
	}
	if configErr.Subtype != errs.SubtypeInvalidConfig || configErr.Field != envvars.CliDeviceInfoCollection {
		t.Fatalf("Resolve() error = %+v, want invalid_config for environment variable", configErr)
	}
}

func TestResolveIgnoresWorkspaceForGlobalSetting(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetCollectionEnvironment(t)
	original := core.CurrentWorkspace()
	t.Cleanup(func() { core.SetCurrentWorkspace(original) })

	if err := core.SaveBaseMultiAppConfig(&core.MultiAppConfig{
		DeviceInfoCollection: boolPointer(false),
		Apps:                 []core.AppConfig{},
	}); err != nil {
		t.Fatal(err)
	}
	core.SetCurrentWorkspace(core.WorkspaceOpenClaw)
	if err := core.SaveMultiAppConfig(&core.MultiAppConfig{Apps: []core.AppConfig{{AppId: "workspace-app"}}}); err != nil {
		t.Fatal(err)
	}

	decision, err := ResolveDeviceInfoCollection()
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}
	if decision.Enabled || decision.Source != DeviceInfoCollectionSourceConfig {
		t.Fatalf("Resolve() = %+v, want global root config to disable collection", decision)
	}
}

func TestSetPreservesExistingProfiles(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetCollectionEnvironment(t)
	seed := &core.MultiAppConfig{
		CurrentApp: "prod",
		Apps: []core.AppConfig{{
			Name:  "prod",
			AppId: "cli_prod",
		}},
	}
	if err := core.SaveBaseMultiAppConfig(seed); err != nil {
		t.Fatal(err)
	}

	if err := SetDeviceInfoCollection(false); err != nil {
		t.Fatalf("Set(false) error = %v", err)
	}
	saved, err := core.LoadBaseMultiAppConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Apps) != 1 || saved.Apps[0].AppId != "cli_prod" || saved.CurrentApp != "prod" {
		t.Fatalf("saved config = %+v, want existing profile preserved", saved)
	}
	if saved.DeviceInfoCollection == nil || *saved.DeviceInfoCollection {
		t.Fatalf("saved device preference = %v, want false", saved.DeviceInfoCollection)
	}
}

func TestSetAndResetWithoutAppConfiguration(t *testing.T) {
	t.Setenv("LARKSUITE_CLI_CONFIG_DIR", t.TempDir())
	unsetCollectionEnvironment(t)

	if err := SetDeviceInfoCollection(false); err != nil {
		t.Fatalf("Set(false) error = %v", err)
	}
	configured, err := ResolveDeviceInfoCollection()
	if err != nil || configured.Enabled || configured.Source != DeviceInfoCollectionSourceConfig {
		t.Fatalf("Resolve() after Set = %+v, %v; want disabled config", configured, err)
	}

	if err := ResetDeviceInfoCollection(); err != nil {
		t.Fatalf("Reset() error = %v", err)
	}
	reset, err := ResolveDeviceInfoCollection()
	if err != nil || !reset.Enabled || reset.Source != DeviceInfoCollectionSourceDefault {
		t.Fatalf("Resolve() after Reset = %+v, %v; want enabled default", reset, err)
	}
}
