// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package deviceinfo

import (
	"errors"
	"io/fs"
	"os"
	"strings"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/core"
	"github.com/larksuite/cli/internal/envvars"
)

// DeviceInfoCollectionDefaultEnabled is used when neither an environment
// override nor a persisted preference is present.
const DeviceInfoCollectionDefaultEnabled = true

// DeviceInfoCollectionSource identifies the winning preference source.
type DeviceInfoCollectionSource string

const (
	DeviceInfoCollectionSourceEnvironment DeviceInfoCollectionSource = "environment"
	DeviceInfoCollectionSourceConfig      DeviceInfoCollectionSource = "config"
	DeviceInfoCollectionSourceDefault     DeviceInfoCollectionSource = "default"
)

// DeviceInfoCollectionDecision is the effective collection preference and
// where it came from.
type DeviceInfoCollectionDecision struct {
	Enabled          bool
	Source           DeviceInfoCollectionSource
	EnvironmentSet   bool
	EnvironmentValue string
	ConfigValue      *bool
	ConfigPath       string
}

// ResolveDeviceInfoCollection applies environment > global config > default
// precedence.
func ResolveDeviceInfoCollection() (DeviceInfoCollectionDecision, error) {
	decision := DeviceInfoCollectionDecision{
		Enabled:    DeviceInfoCollectionDefaultEnabled,
		Source:     DeviceInfoCollectionSourceDefault,
		ConfigPath: core.GetBaseConfigPath(),
	}

	enabled, raw, set, err := deviceInfoCollectionEnvironmentOverride()
	if err != nil {
		return DeviceInfoCollectionDecision{}, err
	}
	if set {
		decision.Enabled = enabled
		decision.Source = DeviceInfoCollectionSourceEnvironment
		decision.EnvironmentSet = true
		decision.EnvironmentValue = raw
		if config, loadErr := core.LoadBaseMultiAppConfig(); loadErr == nil && config.DeviceInfoCollection != nil {
			value := *config.DeviceInfoCollection
			decision.ConfigValue = &value
		}
		return decision, nil
	}

	config, err := core.LoadBaseMultiAppConfig()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return decision, nil
		}
		return DeviceInfoCollectionDecision{}, errs.NewConfigError(errs.SubtypeInvalidConfig,
			"failed to load global device information collection setting: %v", err).
			WithHint("fix %s or set %s to true or false", core.GetBaseConfigPath(), envvars.CliDeviceInfoCollection).
			WithCause(err)
	}
	if config.DeviceInfoCollection != nil {
		value := *config.DeviceInfoCollection
		decision.Enabled = value
		decision.Source = DeviceInfoCollectionSourceConfig
		decision.ConfigValue = &value
	}
	return decision, nil
}

// ValidateDeviceInfoCollectionEnvironment rejects invalid explicit overrides
// before a command mutates the persisted preference.
func ValidateDeviceInfoCollectionEnvironment() error {
	_, _, _, err := deviceInfoCollectionEnvironmentOverride()
	return err
}

func deviceInfoCollectionEnvironmentOverride() (enabled bool, raw string, set bool, err error) {
	raw, set = os.LookupEnv(envvars.CliDeviceInfoCollection)
	if !set {
		return false, "", false, nil
	}
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch normalized {
	case "1", "true", "on", "yes":
		return true, normalized, true, nil
	case "0", "false", "off", "no":
		return false, normalized, true, nil
	default:
		return false, raw, true, errs.NewConfigError(errs.SubtypeInvalidConfig,
			"invalid %s value %q", envvars.CliDeviceInfoCollection, raw).
			WithField(envvars.CliDeviceInfoCollection).
			WithHint("use one of: true, false, 1, 0, on, off, yes, no")
	}
}

// SetDeviceInfoCollection persists the global collection preference while
// preserving profiles and other settings in the root config file.
func SetDeviceInfoCollection(enabled bool) error {
	config, err := core.LoadBaseMultiAppConfig()
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return errs.NewConfigError(errs.SubtypeInvalidConfig,
				"failed to load global device information collection setting: %v", err).
				WithHint("fix %s before changing the setting", core.GetBaseConfigPath()).
				WithCause(err)
		}
		config = &core.MultiAppConfig{Apps: []core.AppConfig{}}
	}
	value := enabled
	config.DeviceInfoCollection = &value
	return core.SaveBaseMultiAppConfig(config)
}

// ResetDeviceInfoCollection removes the explicit preference so resolution
// falls back to the environment override or enabled-by-default behavior.
func ResetDeviceInfoCollection() error {
	config, err := core.LoadBaseMultiAppConfig()
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return errs.NewConfigError(errs.SubtypeInvalidConfig,
			"failed to load global device information collection setting: %v", err).
			WithHint("fix %s before resetting the setting", core.GetBaseConfigPath()).
			WithCause(err)
	}
	config.DeviceInfoCollection = nil
	return core.SaveBaseMultiAppConfig(config)
}
