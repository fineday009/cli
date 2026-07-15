// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package config

import (
	"github.com/spf13/cobra"

	"github.com/larksuite/cli/errs"
	"github.com/larksuite/cli/internal/cmdutil"
	"github.com/larksuite/cli/internal/deviceinfo"
	"github.com/larksuite/cli/internal/envvars"
	"github.com/larksuite/cli/internal/output"
)

type deviceInfoOptions struct {
	Factory *cmdutil.Factory
	Reset   bool
}

type deviceInfoEnvironmentStatus struct {
	Name  string `json:"name"`
	Set   bool   `json:"set"`
	Value string `json:"value,omitempty"`
}

type deviceInfoConfigStatus struct {
	File  string `json:"file"`
	Set   bool   `json:"set"`
	Value *bool  `json:"value,omitempty"`
}

type deviceInfoStatus struct {
	Enabled     bool                                  `json:"enabled"`
	Source      deviceinfo.DeviceInfoCollectionSource `json:"source"`
	Environment deviceInfoEnvironmentStatus           `json:"environment"`
	Config      deviceInfoConfigStatus                `json:"config"`
}

// NewCmdConfigDeviceInfoCollection creates the global device information
// collection preference command.
func NewCmdConfigDeviceInfoCollection(f *cmdutil.Factory) *cobra.Command {
	opts := &deviceInfoOptions{Factory: f}
	cmd := &cobra.Command{
		Use:   "device-info-collection [on|off]",
		Short: "Control collection of device model and OS information",
		Long: `Show or change device information collection for request headers.

The setting is global and stored in the root config.json. Collection is enabled
by default. LARKSUITE_CLI_DEVICE_INFO_COLLECTION has the highest priority.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return errs.NewValidationError(errs.SubtypeInvalidArgument,
					"device-info-collection accepts at most one argument: on or off")
			}
			if len(args) == 1 && args[0] != "on" && args[0] != "off" {
				return errs.NewValidationError(errs.SubtypeInvalidArgument,
					"invalid device-info-collection value %q", args[0]).
					WithHint("use `lark-cli config device-info-collection on` or `lark-cli config device-info-collection off`")
			}
			if opts.Reset && len(args) != 0 {
				return errs.NewValidationError(errs.SubtypeInvalidArgument,
					"--reset cannot be combined with on or off").WithParam("--reset")
			}
			return nil
		},
		// This preference is independent of credential ownership and must remain
		// available when an extension provides credentials.
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SilenceUsage = true
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDeviceInfo(opts, args)
		},
	}
	cmd.Flags().BoolVar(&opts.Reset, "reset", false, "Remove the saved preference (an environment override still applies)")
	cmdutil.SetRisk(cmd, "write")
	return cmd
}

func runDeviceInfo(opts *deviceInfoOptions, args []string) error {
	if opts.Reset || len(args) == 1 {
		if err := deviceinfo.ValidateDeviceInfoCollectionEnvironment(); err != nil {
			return err
		}
		var err error
		if opts.Reset {
			err = deviceinfo.ResetDeviceInfoCollection()
		} else {
			err = deviceinfo.SetDeviceInfoCollection(args[0] == "on")
		}
		if err != nil {
			if _, ok := errs.ProblemOf(err); ok {
				return err
			}
			return errs.NewInternalError(errs.SubtypeStorage,
				"failed to save device information collection setting: %v", err).WithCause(err)
		}
	}

	decision, err := deviceinfo.ResolveDeviceInfoCollection()
	if err != nil {
		return err
	}
	status := deviceInfoStatus{
		Enabled: decision.Enabled,
		Source:  decision.Source,
		Environment: deviceInfoEnvironmentStatus{
			Name:  envvars.CliDeviceInfoCollection,
			Set:   decision.EnvironmentSet,
			Value: decision.EnvironmentValue,
		},
		Config: deviceInfoConfigStatus{
			File:  decision.ConfigPath,
			Set:   decision.ConfigValue != nil,
			Value: decision.ConfigValue,
		},
	}
	output.PrintJson(opts.Factory.IOStreams.Out, status)
	return nil
}
