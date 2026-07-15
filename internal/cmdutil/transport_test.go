// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	exttransport "github.com/larksuite/cli/extension/transport"
	internalauth "github.com/larksuite/cli/internal/auth"
	"github.com/larksuite/cli/internal/envvars"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// ---------------------------------------------------------------------------
// RetryTransport
// ---------------------------------------------------------------------------

func TestRetryTransport_NoRetry(t *testing.T) {
	calls := 0
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	rt := &RetryTransport{Base: base, MaxRetries: 0}
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call, got %d", calls)
	}
}

func TestRetryTransport_RetryOn500(t *testing.T) {
	calls := 0
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("error"))}, nil
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader("ok"))}, nil
	})
	rt := &RetryTransport{Base: base, MaxRetries: 3, Delay: 1 * time.Millisecond}
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("expected 200 after retries, got %d", resp.StatusCode)
	}
	if calls != 3 {
		t.Errorf("expected 3 calls, got %d", calls)
	}
}

func TestRetryTransport_DefaultNoRetry(t *testing.T) {
	calls := 0
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 500, Body: io.NopCloser(strings.NewReader("error"))}, nil
	})
	rt := &RetryTransport{Base: base} // default MaxRetries=0
	req, _ := http.NewRequest("GET", "http://example.com/test", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != 500 {
		t.Errorf("expected 500 with no retries, got %d", resp.StatusCode)
	}
	if calls != 1 {
		t.Errorf("expected 1 call with default config, got %d", calls)
	}
}

// ---------------------------------------------------------------------------
// buildSDKTransport chain composition
// ---------------------------------------------------------------------------

func TestBuildSDKTransport_IncludesRetryTransport(t *testing.T) {
	transport := buildSDKTransport(true, true)

	// Chain: SecurityPolicy → AgentHeaderPolicy → BuildHeader → UserAgent → Retry → Base
	sec, ok := transport.(*internalauth.SecurityPolicyTransport)
	if !ok {
		t.Fatalf("outer transport type = %T, want *auth.SecurityPolicyTransport", transport)
	}
	agentPolicy, ok := sec.Base.(*AgentHeaderPolicyTransport)
	if !ok {
		t.Fatalf("layer after SecurityPolicy = %T, want *AgentHeaderPolicyTransport", sec.Base)
	}
	bh, ok := agentPolicy.Base.(*BuildHeaderTransport)
	if !ok {
		t.Fatalf("layer after AgentHeaderPolicy has type %T, want *BuildHeaderTransport", agentPolicy.Base)
	}
	ua, ok := bh.Base.(*UserAgentTransport)
	if !ok {
		t.Fatalf("layer after BuildHeader = %T, want *UserAgentTransport", bh.Base)
	}
	if !ua.DeviceInfoCollection {
		t.Fatal("UserAgentTransport device collection = false, want enabled")
	}
	if !ua.IsTTY {
		t.Fatal("UserAgentTransport IsTTY = false, want true")
	}
	if _, ok := ua.Base.(*RetryTransport); !ok {
		t.Fatalf("inner transport type = %T, want *RetryTransport", ua.Base)
	}
}

func TestBuildSDKTransport_PropagatesDisabledDeviceCollection(t *testing.T) {
	transport := buildSDKTransport(false, false)
	sec := transport.(*internalauth.SecurityPolicyTransport)
	agentPolicy := sec.Base.(*AgentHeaderPolicyTransport)
	buildHeader := agentPolicy.Base.(*BuildHeaderTransport)
	userAgent := buildHeader.Base.(*UserAgentTransport)
	if userAgent.DeviceInfoCollection {
		t.Fatal("UserAgentTransport device collection = true, want false")
	}
	if userAgent.IsTTY {
		t.Fatal("UserAgentTransport IsTTY = true, want false")
	}
}

func TestBuildSDKTransport_WithExtension(t *testing.T) {
	exttransport.Register(&stubTransportProvider{})
	t.Cleanup(func() { exttransport.Register(nil) })

	transport := buildSDKTransport(true, true)

	// Chain: extensionMiddleware → SecurityPolicy → AgentHeaderPolicy → BuildHeader → UserAgent → Retry → Base
	mid, ok := transport.(*extensionMiddleware)
	if !ok {
		t.Fatalf("outer transport type = %T, want *extensionMiddleware", transport)
	}
	sec, ok := mid.Base.(*internalauth.SecurityPolicyTransport)
	if !ok {
		t.Fatalf("transport type = %T, want *auth.SecurityPolicyTransport", mid.Base)
	}
	agentPolicy, ok := sec.Base.(*AgentHeaderPolicyTransport)
	if !ok {
		t.Fatalf("layer after SecurityPolicy = %T, want *AgentHeaderPolicyTransport", sec.Base)
	}
	bh, ok := agentPolicy.Base.(*BuildHeaderTransport)
	if !ok {
		t.Fatalf("layer after AgentHeaderPolicy has type %T, want *BuildHeaderTransport", agentPolicy.Base)
	}
	ua, ok := bh.Base.(*UserAgentTransport)
	if !ok {
		t.Fatalf("layer after BuildHeader = %T, want *UserAgentTransport", bh.Base)
	}
	if _, ok := ua.Base.(*RetryTransport); !ok {
		t.Fatalf("innermost transport type = %T, want *RetryTransport", ua.Base)
	}
}

func TestBuildSDKTransport_WithoutExtension(t *testing.T) {
	exttransport.Register(nil)

	transport := buildSDKTransport(true, true)

	// Chain: SecurityPolicy → AgentHeaderPolicy → BuildHeader → UserAgent → Retry → Base
	sec, ok := transport.(*internalauth.SecurityPolicyTransport)
	if !ok {
		t.Fatalf("outer transport type = %T, want *auth.SecurityPolicyTransport", transport)
	}
	agentPolicy, ok := sec.Base.(*AgentHeaderPolicyTransport)
	if !ok {
		t.Fatalf("layer after SecurityPolicy = %T, want *AgentHeaderPolicyTransport", sec.Base)
	}
	bh, ok := agentPolicy.Base.(*BuildHeaderTransport)
	if !ok {
		t.Fatalf("layer after AgentHeaderPolicy has type %T, want *BuildHeaderTransport", agentPolicy.Base)
	}
	ua, ok := bh.Base.(*UserAgentTransport)
	if !ok {
		t.Fatalf("layer after BuildHeader = %T, want *UserAgentTransport", bh.Base)
	}
	if _, ok := ua.Base.(*RetryTransport); !ok {
		t.Fatalf("inner transport type = %T, want *RetryTransport", ua.Base)
	}
}

func TestUserAgentTransport_RestrictsDeviceSuffixByRequestHost(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "Lark endpoint keeps enhanced user agent",
			url:  "https://open.larksuite.com/open-apis/test",
			want: UserAgentValue(true, true),
		},
		{
			name: "external endpoint gets product user agent",
			url:  "https://example.com/resource",
			want: UserAgentValue(false, false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received string
			base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				received = req.Header.Get(HeaderUserAgent)
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			})
			req, err := http.NewRequest(http.MethodGet, tt.url, nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}

			resp, err := (&UserAgentTransport{
				Base:                 base,
				DeviceInfoCollection: true,
				IsTTY:                true,
			}).RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}
			resp.Body.Close()

			if received != tt.want {
				t.Fatalf("User-Agent = %q, want %q", received, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// extensionMiddleware — legacy Interceptor path
// ---------------------------------------------------------------------------

type stubTransportProvider struct {
	interceptor exttransport.Interceptor
}

func (s *stubTransportProvider) Name() string { return "stub" }
func (s *stubTransportProvider) ResolveInterceptor(context.Context) exttransport.Interceptor {
	if s.interceptor != nil {
		return s.interceptor
	}
	return &stubTransportImpl{}
}

type stubTransportImpl struct{}

func (s *stubTransportImpl) PreRoundTrip(req *http.Request) func(*http.Response, error) {
	return nil
}

// headerCapturingInterceptor sets custom headers in PreRoundTrip and records
// whether PostRoundTrip was called, to verify execution order.
type headerCapturingInterceptor struct {
	preCalled  bool
	postCalled bool
}

func (h *headerCapturingInterceptor) PreRoundTrip(req *http.Request) func(*http.Response, error) {
	h.preCalled = true
	// Set a custom header that should survive (no built-in override)
	req.Header.Set("X-Custom-Trace", "ext-trace-123")
	// Try to override a security header — should be overwritten by SecurityHeaderTransport
	req.Header.Set(HeaderSource, "ext-tampered")
	return func(resp *http.Response, err error) {
		h.postCalled = true
	}
}

func TestExtensionInterceptor_ExecutionOrder(t *testing.T) {
	var receivedHeaders http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	ic := &headerCapturingInterceptor{}
	exttransport.Register(&stubTransportProvider{interceptor: ic})
	t.Cleanup(func() { exttransport.Register(nil) })

	// Use HTTP transport chain (has SecurityHeaderTransport)
	var base http.RoundTripper = http.DefaultTransport
	base = &RetryTransport{Base: base}
	base = &SecurityHeaderTransport{Base: base, DeviceInfoCollection: true}
	transport := wrapWithExtension(base)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// PreRoundTrip was called
	if !ic.preCalled {
		t.Fatal("PreRoundTrip was not called")
	}
	// PostRoundTrip (closure) was called
	if !ic.postCalled {
		t.Fatal("PostRoundTrip closure was not called")
	}
	// Custom header set by extension survives (no built-in override)
	if got := receivedHeaders.Get("X-Custom-Trace"); got != "ext-trace-123" {
		t.Fatalf("X-Custom-Trace = %q, want %q", got, "ext-trace-123")
	}
	// Security header overridden by extension is restored by SecurityHeaderTransport
	if got := receivedHeaders.Get(HeaderSource); got != SourceValue {
		t.Fatalf("%s = %q, want %q (built-in should override extension)", HeaderSource, got, SourceValue)
	}
}

// buildTamperingInterceptor tries to delete and spoof X-Cli-Build via
// PreRoundTrip. The SDK chain's BuildHeaderTransport must restore the real
// value before the request leaves the process.
type buildTamperingInterceptor struct{}

func (buildTamperingInterceptor) PreRoundTrip(req *http.Request) func(*http.Response, error) {
	req.Header.Del(HeaderBuild)
	req.Header.Set(HeaderBuild, "ext-tampered-build")
	return nil
}

// TestBuildHeaderTransport_SDKChain_OverridesTamperedHeader verifies that the
// X-Cli-Build header is force-written by BuildHeaderTransport in the SDK
// transport chain, even when an extension tries to delete or spoof it. This
// closes the gap where the SDK chain had no equivalent of
// SecurityHeaderTransport (see design doc §3.3.3).
func TestBuildHeaderTransport_SDKChain_OverridesTamperedHeader(t *testing.T) {
	var receivedBuild string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBuild = r.Header.Get(HeaderBuild)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	exttransport.Register(&stubTransportProvider{interceptor: buildTamperingInterceptor{}})
	t.Cleanup(func() { exttransport.Register(nil) })

	// Replicate the SDK chain layering used by buildSDKTransport.
	var base http.RoundTripper = http.DefaultTransport
	base = &RetryTransport{Base: base}
	base = &UserAgentTransport{Base: base, DeviceInfoCollection: true}
	base = &BuildHeaderTransport{Base: base}
	transport := wrapWithExtension(base)
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedBuild == "ext-tampered-build" {
		t.Fatalf("%s = %q, extension tampering leaked to network", HeaderBuild, receivedBuild)
	}
	want := DetectBuildKind()
	if receivedBuild != want {
		t.Fatalf("%s = %q, want %q", HeaderBuild, receivedBuild, want)
	}
}

// TestBuildHeaderTransport_OverridesEvenWithoutTamper verifies that even if
// no extension is registered, BuildHeaderTransport writes X-Cli-Build.
func TestBuildHeaderTransport_OverridesEvenWithoutTamper(t *testing.T) {
	var receivedBuild string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBuild = r.Header.Get(HeaderBuild)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &BuildHeaderTransport{Base: http.DefaultTransport}
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	if receivedBuild == "" {
		t.Fatalf("%s header missing, BuildHeaderTransport did not inject", HeaderBuild)
	}
	want := DetectBuildKind()
	if receivedBuild != want {
		t.Fatalf("%s = %q, want %q", HeaderBuild, receivedBuild, want)
	}
}

// TestBuildHeaderTransport_NilBase_UsesFallback verifies that when Base is nil,
// the transport still sets X-Cli-Build and routes the request through
// transport.Fallback rather than panicking. This covers the fallback
// branch in RoundTrip that is otherwise unreachable with a non-nil Base.
func TestBuildHeaderTransport_NilBase_UsesFallback(t *testing.T) {
	var receivedBuild string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedBuild = r.Header.Get(HeaderBuild)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	transport := &BuildHeaderTransport{Base: nil}
	client := &http.Client{Transport: transport}

	req, _ := http.NewRequest("GET", srv.URL, nil)
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request via nil-Base transport failed: %v", err)
	}
	resp.Body.Close()

	want := DetectBuildKind()
	if receivedBuild != want {
		t.Fatalf("%s = %q, want %q (header must be set even on nil-Base path)",
			HeaderBuild, receivedBuild, want)
	}
}

func TestAgentHeaderPolicyTransport_StripsRestrictedAgentHeadersOffDomain(t *testing.T) {
	var received http.Header
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		received = req.Header.Clone()
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})
	req, err := http.NewRequest(http.MethodGet, "https://example.com/test", nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	for _, header := range domainRestrictedAgentHeaders {
		req.Header.Set(header, "sensitive-value")
	}
	req.Header.Set(HeaderAgentTrace, "trace-kept")
	req.Header.Set(HeaderUserAgent, UserAgentValue(true, true))

	resp, err := (&AgentHeaderPolicyTransport{Base: base}).RoundTrip(req)
	if err != nil {
		t.Fatalf("RoundTrip() error = %v", err)
	}
	resp.Body.Close()

	for _, header := range domainRestrictedAgentHeaders {
		if got := received.Get(header); got != "" {
			t.Errorf("%s leaked to disallowed host: %q", header, got)
		}
	}
	if got := received.Get(HeaderAgentTrace); got != "trace-kept" {
		t.Errorf("%s = %q, want %q", HeaderAgentTrace, got, "trace-kept")
	}
	if got := received.Get(HeaderUserAgent); got != UserAgentValue(false, false) {
		t.Errorf("%s = %q, want product-only value %q", HeaderUserAgent, got, UserAgentValue(false, false))
	}
}

func TestAgentHeadersRestrictedByRequestHost(t *testing.T) {
	t.Setenv(envvars.CliAgentTrace, "trace-allowed-everywhere")

	tests := []struct {
		name    string
		host    string
		allowed bool
	}{
		{name: "Feishu endpoint", host: "open.feishu.cn:443", allowed: true},
		{name: "Lark endpoint", host: "mcp.larksuite.com", allowed: true},
		{name: "external endpoint", host: "example.com", allowed: false},
		{name: "lookalike endpoint", host: "open.feishu.cn.evil.example", allowed: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var received http.Header
			base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
				received = req.Header.Clone()
				return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
			})

			var rt http.RoundTripper = base
			rt = &RetryTransport{Base: rt}
			rt = &AgentHeaderPolicyTransport{Base: rt}
			rt = &SecurityHeaderTransport{Base: rt, DeviceInfoCollection: true}

			req, err := http.NewRequest(http.MethodGet, "https://"+tt.host+"/test", nil)
			if err != nil {
				t.Fatalf("new request: %v", err)
			}
			req.Header.Set("Authorization", "Bearer t-real-tat")
			for _, header := range domainRestrictedAgentHeaders {
				req.Header.Set(header, "preset-sensitive-value")
			}
			// A non-canonical key must not bypass the off-domain scrubber.
			req.Header["x-agent-device-type"] = []string{"lowercase-sensitive-value"}

			resp, err := rt.RoundTrip(req)
			if err != nil {
				t.Fatalf("RoundTrip() error = %v", err)
			}
			resp.Body.Close()

			if got := received.Get(HeaderAgentTrace); got != "trace-allowed-everywhere" {
				t.Errorf("%s = %q, want trace on every domain", HeaderAgentTrace, got)
			}
			if tt.allowed {
				if got := received.Get(HeaderUserAgent); got != UserAgentValue(true, false) {
					t.Errorf("%s = %q, want enhanced value %q", HeaderUserAgent, got, UserAgentValue(true, false))
				}
				for _, header := range []string{HeaderAgentTerminalType, HeaderAgentDeviceType, HeaderAgentOSType} {
					if got := received.Get(header); got == "" {
						t.Errorf("%s missing on allowed host", header)
					}
				}
				return
			}

			for _, header := range domainRestrictedAgentHeaders {
				if got := received.Get(header); got != "" {
					t.Errorf("%s leaked to disallowed host: %q", header, got)
				}
			}
			if got := received.Get(HeaderUserAgent); got != UserAgentValue(false, false) {
				t.Errorf("%s = %q, want product-only value %q", HeaderUserAgent, got, UserAgentValue(false, false))
			}
			for header := range received {
				if isDomainRestrictedAgentHeader(header) {
					t.Errorf("non-canonical restricted header leaked to disallowed host: %s", header)
				}
			}
		})
	}
}

// interceptorFunc adapts a function to exttransport.Interceptor.
type interceptorFunc func(*http.Request) func(*http.Response, error)

func (f interceptorFunc) PreRoundTrip(req *http.Request) func(*http.Response, error) { return f(req) }

func TestExtensionInterceptor_ContextTamperPrevented(t *testing.T) {
	type ctxKeyType string
	const testKey ctxKeyType = "original"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var ctxValue any

	// Use a custom transport that captures the context value seen by the built-in chain
	capturer := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		ctxValue = req.Context().Value(testKey)
		return http.DefaultTransport.RoundTrip(req)
	})

	// Interceptor that tries to tamper with context
	tamperIC := interceptorFunc(func(req *http.Request) func(*http.Response, error) {
		// Try to replace context with a new one
		*req = *req.WithContext(context.WithValue(req.Context(), testKey, "tampered"))
		return nil
	})

	mid := &extensionMiddleware{Base: capturer, Ext: tamperIC}

	origCtx := context.WithValue(context.Background(), testKey, "original")
	req, _ := http.NewRequestWithContext(origCtx, "GET", srv.URL, nil)
	resp, err := mid.RoundTrip(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	resp.Body.Close()

	// Built-in chain should see original context, not tampered
	if ctxValue != "original" {
		t.Fatalf("built-in chain saw context value %q, want %q", ctxValue, "original")
	}
}

// ---------------------------------------------------------------------------
// extensionMiddleware — PreRoundTripE abort path
// ---------------------------------------------------------------------------

// abortingInterceptor implements exttransport.AbortableInterceptor and
// records invocation of the pre and post hooks. These middleware tests only
// assert middleware-level integration; pure *AbortError behavior
// (Error/Unwrap/Is/As) is covered in extension/transport/errors_test.go.
type abortingInterceptor struct {
	reason     error // if non-nil, PreRoundTripE returns this to abort
	nilPost    bool  // if true, PreRoundTripE returns a nil post func
	preECalled bool
	postCalled bool
	postResp   *http.Response
	postErr    error
}

// PreRoundTrip is a no-op that satisfies the legacy Interceptor method; the
// middleware never calls it when PreRoundTripE is present.
func (*abortingInterceptor) PreRoundTrip(*http.Request) func(*http.Response, error) {
	return nil
}

func (a *abortingInterceptor) PreRoundTripE(req *http.Request) (func(*http.Response, error), error) {
	a.preECalled = true
	if a.nilPost {
		return nil, a.reason
	}
	return func(resp *http.Response, err error) {
		a.postCalled = true
		a.postResp = resp
		a.postErr = err
	}, a.reason
}

func TestExtensionMiddleware_PreRoundTripEAbort(t *testing.T) {
	innerErr := errors.New("denied by policy")

	t.Run("skips base and wires AbortError fields", func(t *testing.T) {
		ic := &abortingInterceptor{reason: innerErr}
		baseCalls := 0
		base := roundTripFunc(func(*http.Request) (*http.Response, error) {
			baseCalls++
			return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
		})

		mid := &extensionMiddleware{Base: base, Ext: ic, ExtName: "stub"}
		req, _ := http.NewRequest("GET", "http://example.invalid/", nil)
		resp, err := mid.RoundTrip(req)

		if resp != nil {
			t.Fatalf("resp = %v, want nil on abort", resp)
		}
		if baseCalls != 0 {
			t.Fatalf("base RoundTrip called %d times on abort, want 0", baseCalls)
		}
		if !ic.preECalled {
			t.Fatal("PreRoundTripE was not called")
		}

		var aErr *exttransport.AbortError
		if !errors.As(err, &aErr) {
			t.Fatalf("errors.As(*AbortError) = false, err = %v (%T)", err, err)
		}
		if aErr.Extension != "stub" || aErr.Reason != innerErr {
			t.Fatalf("AbortError = %+v, want {Extension:stub Reason:%v}", aErr, innerErr)
		}

		// Post must see the original inner err, not the *AbortError wrapper.
		if !ic.postCalled {
			t.Fatal("post hook was not called on abort")
		}
		if ic.postResp != nil {
			t.Fatalf("post resp = %v, want nil", ic.postResp)
		}
		if ic.postErr != innerErr {
			t.Fatalf("post err = %v, want original inner err %v", ic.postErr, innerErr)
		}
	})

	t.Run("nil post still returns AbortError", func(t *testing.T) {
		ic := &abortingInterceptor{reason: innerErr, nilPost: true}
		base := roundTripFunc(func(*http.Request) (*http.Response, error) {
			t.Fatal("base must not be called on abort")
			return nil, nil
		})

		mid := &extensionMiddleware{Base: base, Ext: ic, ExtName: "stub"}
		req, _ := http.NewRequest("GET", "http://example.invalid/", nil)
		_, err := mid.RoundTrip(req)

		var aErr *exttransport.AbortError
		if !errors.As(err, &aErr) {
			t.Fatalf("errors.As(*AbortError) = false, err = %v", err)
		}
	})
}

func TestExtensionMiddleware_PreRoundTripEHappyPath(t *testing.T) {
	ic := &abortingInterceptor{} // reason == nil → no abort
	baseCalls := 0
	base := roundTripFunc(func(*http.Request) (*http.Response, error) {
		baseCalls++
		return &http.Response{StatusCode: http.StatusOK, Body: http.NoBody}, nil
	})

	mid := &extensionMiddleware{Base: base, Ext: ic, ExtName: "stub"}
	req, _ := http.NewRequest("GET", "http://example.invalid/", nil)
	resp, err := mid.RoundTrip(req)
	if err != nil {
		t.Fatalf("happy path returned err: %v", err)
	}
	if resp == nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("resp = %v, want 200", resp)
	}
	if baseCalls != 1 {
		t.Fatalf("base RoundTrip called %d times, want 1", baseCalls)
	}
	if !ic.preECalled {
		t.Fatal("PreRoundTripE was not called")
	}
	if !ic.postCalled || ic.postErr != nil {
		t.Fatalf("post hook not called or err != nil: called=%v err=%v", ic.postCalled, ic.postErr)
	}
}
