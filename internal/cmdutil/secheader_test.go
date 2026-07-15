// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"net/http"
	"testing"

	"github.com/larksuite/cli/extension/credential"
	envcred "github.com/larksuite/cli/extension/credential/env"
	"github.com/larksuite/cli/internal/build"
	"github.com/larksuite/cli/internal/deviceinfo"
	"github.com/larksuite/cli/internal/envvars"
	"github.com/larksuite/cli/internal/vfs/localfileio"
)

// ---------------------------------------------------------------------------
// isBuiltinProvider
// ---------------------------------------------------------------------------

// cmdutilLocalProvider has PkgPath under the official module
// ("github.com/larksuite/cli/internal/cmdutil") and should be classified
// as builtin.
type cmdutilLocalProvider struct{}

// Name intentionally returns a value that mimics an external provider; the
// PkgPath-based classifier must ignore it. See TestIsBuiltinProvider_PkgPathNotSpoofableByName.
func (cmdutilLocalProvider) Name() string { return "external-spoofed-provider" }
func (cmdutilLocalProvider) ResolveAccount(context.Context) (*credential.Account, error) {
	return nil, nil
}
func (cmdutilLocalProvider) ResolveToken(context.Context, credential.TokenSpec) (*credential.Token, error) {
	return nil, nil
}

func TestIsBuiltinProvider_Nil(t *testing.T) {
	if isBuiltinProvider(nil) {
		t.Fatal("isBuiltinProvider(nil) = true, want false")
	}
}

func TestIsBuiltinProvider_TypeUnderOfficialModule(t *testing.T) {
	if !isBuiltinProvider(&cmdutilLocalProvider{}) {
		t.Fatal("type under github.com/larksuite/cli/... should be builtin")
	}
}

func TestIsBuiltinProvider_StdlibTypeIsNotBuiltin(t *testing.T) {
	// A standard library type has PkgPath "net/http" — outside official module.
	// This covers the non-builtin branch, which we cannot trigger from inside
	// this test file using a locally-defined type.
	if isBuiltinProvider(&http.Server{}) {
		t.Fatal("stdlib type classified as builtin, PkgPath check is broken")
	}
}

func TestIsBuiltinProvider_PkgPathNotSpoofableByName(t *testing.T) {
	// Name() returns a string, but classification uses reflect.Type.PkgPath
	// which is compile-time fixed. The local type returns a name that looks
	// like an ISV provider; it must still classify as builtin.
	p := &cmdutilLocalProvider{}
	if p.Name() != "external-spoofed-provider" {
		t.Fatalf("sanity check: Name() = %q, spoof value lost", p.Name())
	}
	if !isBuiltinProvider(p) {
		t.Fatal("isBuiltinProvider should decide by PkgPath, not Name()")
	}
}

// TestIsBuiltinProvider_NonPointerValues covers the non-pointer reflect branch.
// The existing tests only exercise pointer receivers (&T{}); when a provider
// is passed by value the reflect.Kind is not Ptr and t.Elem() is skipped.
func TestIsBuiltinProvider_NonPointerValues(t *testing.T) {
	if !isBuiltinProvider(cmdutilLocalProvider{}) {
		t.Fatal("non-pointer local type should be builtin (PkgPath still under official module)")
	}
	// http.Server as a non-pointer — PkgPath "net/http", not under official.
	if isBuiltinProvider(http.Server{}) {
		t.Fatal("non-pointer stdlib type should not be builtin")
	}
}

// TestIsBuiltinProvider_RealBuiltinProviders locks down the classification
// for the concrete providers enumerated in design doc §3.3.2 as "官方自带":
// env credential provider and local fileio provider. If any of these is
// moved out of the official module tree in the future, this test must flip
// red so the new package path is explicitly considered.
//
// The sidecar providers (extension/credential/sidecar and
// extension/transport/sidecar) are guarded by the `authsidecar` build tag
// and covered in secheader_sidecar_test.go under that tag.
func TestIsBuiltinProvider_RealBuiltinProviders(t *testing.T) {
	cases := []struct {
		name     string
		provider any
	}{
		{"env credential provider", &envcred.Provider{}},
		{"local fileio provider", &localfileio.Provider{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if !isBuiltinProvider(tc.provider) {
				t.Fatalf("%T must be classified as builtin (PkgPath under %s)", tc.provider, officialModulePath)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// computeBuildKind
// ---------------------------------------------------------------------------

func TestComputeBuildKind_ReturnsKnownValue(t *testing.T) {
	// Under `go test`, Main.Path is typically the module being tested
	// ("github.com/larksuite/cli"); the concrete return may still be
	// official, extended, or unknown depending on Main.Path and the
	// registered providers. Just assert it's one of the defined values.
	got := computeBuildKind()
	switch got {
	case BuildKindOfficial, BuildKindExtended, BuildKindUnknown:
	default:
		t.Fatalf("computeBuildKind() = %q, want one of official/extended/unknown", got)
	}
}

// ---------------------------------------------------------------------------
// classifyBuild — pure branching logic
// ---------------------------------------------------------------------------
//
// These tests cover every branch of classifyBuild with explicit inputs,
// which is impossible from computeBuildKind alone because debug.ReadBuildInfo
// and the process-wide provider registries can't be reshaped in a test.

func TestClassifyBuild_NoBuildInfo_ReturnsUnknown(t *testing.T) {
	if got := classifyBuild("", false, nil, nil, nil); got != BuildKindUnknown {
		t.Fatalf("classifyBuild(haveBuildInfo=false) = %q, want %q", got, BuildKindUnknown)
	}
}

func TestClassifyBuild_ExtendedMainPath_ReturnsExtended(t *testing.T) {
	cases := []string{
		"github.com/acme/lark-cli-wrapper",
		"example.com/isv/lark",
		"gitlab.mycorp.internal/tools/lark-cli-fork",
	}
	for _, mp := range cases {
		t.Run(mp, func(t *testing.T) {
			if got := classifyBuild(mp, true, nil, nil, nil); got != BuildKindExtended {
				t.Fatalf("mainPath=%q classifyBuild = %q, want %q", mp, got, BuildKindExtended)
			}
		})
	}
}

func TestClassifyBuild_OfficialMainPath_NoProviders_ReturnsOfficial(t *testing.T) {
	if got := classifyBuild(officialModulePath, true, nil, nil, nil); got != BuildKindOfficial {
		t.Fatalf("classifyBuild(official, no providers) = %q, want %q", got, BuildKindOfficial)
	}
}

func TestClassifyBuild_EmptyMainPath_DoesNotTriggerExtended(t *testing.T) {
	// An empty Main.Path (rare, e.g. `go run` pre-1.18) must not be treated
	// as extended by itself — the classifier falls through to provider checks.
	if got := classifyBuild("", true, nil, nil, nil); got != BuildKindOfficial {
		t.Fatalf("classifyBuild(empty mainPath, no providers) = %q, want %q", got, BuildKindOfficial)
	}
}

func TestClassifyBuild_NonBuiltinCredentialProvider_ReturnsExtended(t *testing.T) {
	// Any non-builtin credential provider flips the verdict to extended.
	got := classifyBuild(officialModulePath, true, []any{&http.Server{}}, nil, nil)
	if got != BuildKindExtended {
		t.Fatalf("classifyBuild with external credential = %q, want %q", got, BuildKindExtended)
	}
}

func TestClassifyBuild_MixedCredentialProviders_ExtendedWins(t *testing.T) {
	// Even if most providers are builtin, a single external one decides.
	providers := []any{&cmdutilLocalProvider{}, &http.Server{}}
	if got := classifyBuild(officialModulePath, true, providers, nil, nil); got != BuildKindExtended {
		t.Fatalf("classifyBuild mixed providers = %q, want %q", got, BuildKindExtended)
	}
}

func TestClassifyBuild_NonBuiltinTransportProvider_ReturnsExtended(t *testing.T) {
	got := classifyBuild(officialModulePath, true, nil, &http.Server{}, nil)
	if got != BuildKindExtended {
		t.Fatalf("classifyBuild with external transport = %q, want %q", got, BuildKindExtended)
	}
}

func TestClassifyBuild_NonBuiltinFileioProvider_ReturnsExtended(t *testing.T) {
	got := classifyBuild(officialModulePath, true, nil, nil, &http.Server{})
	if got != BuildKindExtended {
		t.Fatalf("classifyBuild with external fileio = %q, want %q", got, BuildKindExtended)
	}
}

func TestClassifyBuild_AllBuiltinProviders_ReturnsOfficial(t *testing.T) {
	// All three slots filled with builtin providers must still classify as official.
	got := classifyBuild(
		officialModulePath, true,
		[]any{&cmdutilLocalProvider{}},
		&cmdutilLocalProvider{},
		&cmdutilLocalProvider{},
	)
	if got != BuildKindOfficial {
		t.Fatalf("classifyBuild all-builtin = %q, want %q", got, BuildKindOfficial)
	}
}

// TestClassifyBuild_MainPathPriorityOverProviders documents that the main
// module path takes precedence: even with only builtin providers, a non-
// official main path still yields extended.
func TestClassifyBuild_MainPathPriorityOverProviders(t *testing.T) {
	got := classifyBuild(
		"github.com/acme/lark-wrapper", true,
		[]any{&cmdutilLocalProvider{}},
		&cmdutilLocalProvider{},
		&cmdutilLocalProvider{},
	)
	if got != BuildKindExtended {
		t.Fatalf("main-path override failed: got %q, want %q", got, BuildKindExtended)
	}
}

// ---------------------------------------------------------------------------
// DetectBuildKind — sync.Once caching
// ---------------------------------------------------------------------------

func TestDetectBuildKind_StableAcrossCalls(t *testing.T) {
	a := DetectBuildKind()
	b := DetectBuildKind()
	if a != b {
		t.Fatalf("DetectBuildKind() returned different values on repeat: %q vs %q", a, b)
	}
}

// ---------------------------------------------------------------------------
// BaseSecurityHeaders
// ---------------------------------------------------------------------------

func TestBaseSecurityHeaders_IncludesBuildHeader(t *testing.T) {
	h := BaseSecurityHeaders(true, true)
	v := h.Get(HeaderBuild)
	if v == "" {
		t.Fatal("BaseSecurityHeaders missing X-Cli-Build header")
	}
	switch v {
	case BuildKindOfficial, BuildKindExtended, BuildKindUnknown:
	default:
		t.Fatalf("X-Cli-Build = %q, want one of official/extended/unknown", v)
	}
}

func TestUserAgentValueUsesEnhancedDeviceSuffix(t *testing.T) {
	originalUserAgent := buildDeviceUserAgent
	t.Cleanup(func() { buildDeviceUserAgent = originalUserAgent })
	buildDeviceUserAgent = func(isTTY bool) string {
		if isTTY {
			return "(macOS 26.0.4; arm64; terminal)"
		}
		return "(macOS 26.0.4; arm64; non-terminal)"
	}

	tests := []struct {
		name    string
		isTTY   bool
		surface string
	}{
		{name: "terminal", isTTY: true, surface: "terminal"},
		{name: "non terminal", isTTY: false, surface: "non-terminal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := SourceValue + "/" + build.Version + " (macOS 26.0.4; arm64; " + tt.surface + ")"
			if got := UserAgentValue(true, tt.isTTY); got != want {
				t.Fatalf("UserAgentValue(true, %t) = %q, want %q", tt.isTTY, got, want)
			}
		})
	}
}

func TestBaseSecurityHeaders_AllRequiredHeaders(t *testing.T) {
	h := BaseSecurityHeaders(true, true)
	for _, key := range []string{
		HeaderSource,
		HeaderVersion,
		HeaderBuild,
		HeaderUserAgent,
		HeaderAgentTerminalType,
		HeaderAgentDeviceType,
		HeaderAgentOSType,
	} {
		if h.Get(key) == "" {
			t.Errorf("BaseSecurityHeaders missing %s", key)
		}
	}
	if got := h.Get(HeaderAgentTerminalType); got != deviceinfo.TerminalTypePC {
		t.Errorf("%s = %q, want %q", HeaderAgentTerminalType, got, deviceinfo.TerminalTypePC)
	}
	if got := h.Get(HeaderAgentDeviceType); got != deviceinfo.Get() {
		t.Errorf("%s = %q, want %q", HeaderAgentDeviceType, got, deviceinfo.Get())
	}
	wantOSType := deviceinfo.GetOSType(deviceinfo.OSName())
	if got := h.Get(HeaderAgentOSType); got != wantOSType {
		t.Errorf("%s = %q, want %q", HeaderAgentOSType, got, wantOSType)
	}
}

func TestBaseSecurityHeaders_DeviceCollectionDisabled(t *testing.T) {
	originalUserAgent := buildDeviceUserAgent
	originalModel := collectDeviceModel
	originalOSName := collectDeviceOSName
	t.Cleanup(func() {
		buildDeviceUserAgent = originalUserAgent
		collectDeviceModel = originalModel
		collectDeviceOSName = originalOSName
	})
	probes := 0
	buildDeviceUserAgent = func(bool) string { probes++; return "unexpected-user-agent" }
	collectDeviceModel = func() string { probes++; return "unexpected-model" }
	collectDeviceOSName = func() string { probes++; return "unexpected-os" }

	h := BaseSecurityHeaders(false, false)
	if probes != 0 {
		t.Fatalf("device information probes called %d times while collection disabled", probes)
	}
	for _, key := range []string{
		HeaderAgentTerminalType,
		HeaderAgentDeviceType,
		HeaderAgentOSType,
	} {
		if got := h.Get(key); got != "" {
			t.Errorf("BaseSecurityHeaders(false)[%s] = %q, want omitted", key, got)
		}
	}
	wantUserAgent := SourceValue + "/" + build.Version
	if got := h.Get(HeaderUserAgent); got != wantUserAgent {
		t.Errorf("User-Agent = %q, want privacy fallback %q", got, wantUserAgent)
	}
	for _, key := range []string{HeaderSource, HeaderVersion, HeaderBuild} {
		if h.Get(key) == "" {
			t.Errorf("BaseSecurityHeaders(false) removed non-device header %s", key)
		}
	}
}

// ---------------------------------------------------------------------------
// HeaderAgentTrace injection (via BaseSecurityHeaders)
// ---------------------------------------------------------------------------

func TestBaseSecurityHeaders_NoAgentTraceHeaderWhenEnvUnset(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "")
	h := BaseSecurityHeaders(true, true)
	if v := h.Get(HeaderAgentTrace); v != "" {
		t.Fatalf("BaseSecurityHeaders() included %s = %q, want absent when env unset", HeaderAgentTrace, v)
	}
}

func TestBaseSecurityHeaders_IncludesAgentTraceHeaderWhenEnvSet(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "trace-xyz-789")
	h := BaseSecurityHeaders(true, true)
	if v := h.Get(HeaderAgentTrace); v != "trace-xyz-789" {
		t.Fatalf("BaseSecurityHeaders()[%s] = %q, want %q", HeaderAgentTrace, v, "trace-xyz-789")
	}
}

func TestBaseSecurityHeaders_AgentTraceTrimmedWhitespace(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "  trace-trim  ")
	h := BaseSecurityHeaders(true, true)
	if v := h.Get(HeaderAgentTrace); v != "trace-trim" {
		t.Fatalf("BaseSecurityHeaders()[%s] = %q, want %q (whitespace trimmed)", HeaderAgentTrace, v, "trace-trim")
	}
}

func TestBaseSecurityHeaders_AgentTraceOnlyWhitespace_Skipped(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "   ")
	h := BaseSecurityHeaders(true, true)
	if v := h.Get(HeaderAgentTrace); v != "" {
		t.Fatalf("BaseSecurityHeaders()[%s] = %q, want absent for whitespace-only value", HeaderAgentTrace, v)
	}
}

func TestBaseSecurityHeaders_AgentTraceRejectsCRLFInjection(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "val\r\nX-Evil: attack")
	h := BaseSecurityHeaders(true, true)
	if v := h.Get(HeaderAgentTrace); v != "" {
		t.Fatalf("BaseSecurityHeaders()[%s] = %q, want absent for CR/LF value", HeaderAgentTrace, v)
	}
}

func TestBaseSecurityHeaders_AgentTraceRejectsLFInjection(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "val\nX-Evil: attack")
	h := BaseSecurityHeaders(true, true)
	if v := h.Get(HeaderAgentTrace); v != "" {
		t.Fatalf("BaseSecurityHeaders()[%s] = %q, want absent for LF value", HeaderAgentTrace, v)
	}
}

func TestIsAgentHeaderAllowedHost(t *testing.T) {
	tests := []struct {
		host string
		want bool
	}{
		{host: "open.feishu.cn", want: true},
		{host: "accounts.feishu.cn", want: true},
		{host: "mcp.feishu.cn", want: true},
		{host: "applink.feishu.cn", want: true},
		{host: "open.larksuite.com", want: true},
		{host: "accounts.larksuite.com", want: true},
		{host: "mcp.larksuite.com", want: true},
		{host: "applink.larksuite.com", want: true},
		{host: "OPEN.LARKSUITE.COM", want: true},
		{host: "example.com"},
		{host: "feishu.cn"},
		{host: "api.open.feishu.cn"},
		{host: "open.feishu.cn.evil.example"},
		{host: "open.feishu-boe.cn"},
		{host: "open.larksuite-pre.com"},
		{host: ""},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			if got := isAgentHeaderAllowedHost(tt.host); got != tt.want {
				t.Fatalf("isAgentHeaderAllowedHost(%q) = %t, want %t", tt.host, got, tt.want)
			}
		})
	}
}
