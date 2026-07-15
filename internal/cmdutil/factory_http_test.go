// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"errors"
	"io"
	"testing"

	exttransport "github.com/larksuite/cli/extension/transport"
	internalauth "github.com/larksuite/cli/internal/auth"
	"github.com/larksuite/cli/internal/deviceinfo"
)

func TestCachedHttpClientFunc_ReturnsSameInstance(t *testing.T) {
	fn := cachedHttpClientFunc(&Factory{IOStreams: &IOStreams{ErrOut: io.Discard}})

	c1, err := fn()
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	if c1 == nil {
		t.Fatal("first call returned nil")
	}

	c2, err := fn()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if c1 != c2 {
		t.Error("expected same *http.Client instance on second call (cache hit)")
	}
}

func TestCachedHttpClientFunc_PropagatesDisabledDeviceCollection(t *testing.T) {
	f := &Factory{
		IOStreams: &IOStreams{ErrOut: io.Discard},
		DeviceInfoCollection: func() (deviceinfo.DeviceInfoCollectionDecision, error) {
			return deviceinfo.DeviceInfoCollectionDecision{Enabled: false, Source: deviceinfo.DeviceInfoCollectionSourceConfig}, nil
		},
	}
	client, err := cachedHttpClientFunc(f)()
	if err != nil {
		t.Fatal(err)
	}
	policy := client.Transport.(*internalauth.SecurityPolicyTransport)
	security := policy.Base.(*SecurityHeaderTransport)
	if security.DeviceInfoCollection {
		t.Fatal("SecurityHeaderTransport device collection = true, want false")
	}
}

func TestCachedHttpClientFunc_PropagatesTTYState(t *testing.T) {
	tests := []struct {
		name  string
		isTTY bool
	}{
		{name: "terminal", isTTY: true},
		{name: "non terminal", isTTY: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &Factory{IOStreams: &IOStreams{ErrOut: io.Discard, IsTerminal: tt.isTTY}}
			client, err := cachedHttpClientFunc(f)()
			if err != nil {
				t.Fatal(err)
			}
			policy := client.Transport.(*internalauth.SecurityPolicyTransport)
			security := policy.Base.(*SecurityHeaderTransport)
			if security.IsTTY != tt.isTTY {
				t.Fatalf("SecurityHeaderTransport IsTTY = %t, want %t", security.IsTTY, tt.isTTY)
			}
		})
	}
}

func TestCachedHttpClientFunc_DeviceCollectionErrorStopsConstruction(t *testing.T) {
	sentinel := errors.New("invalid device collection setting")
	f := &Factory{
		IOStreams: &IOStreams{ErrOut: io.Discard},
		DeviceInfoCollection: func() (deviceinfo.DeviceInfoCollectionDecision, error) {
			return deviceinfo.DeviceInfoCollectionDecision{}, sentinel
		},
	}
	client, err := cachedHttpClientFunc(f)()
	if client != nil || !errors.Is(err, sentinel) {
		t.Fatalf("cachedHttpClientFunc() = %v, %v; want nil client and sentinel", client, err)
	}
}

func TestCachedHttpClientFunc_HasTimeout(t *testing.T) {
	fn := cachedHttpClientFunc(&Factory{IOStreams: &IOStreams{ErrOut: io.Discard}})
	c, _ := fn()
	if c.Timeout == 0 {
		t.Error("expected non-zero timeout")
	}
}

func TestCachedHttpClientFunc_HasRedirectPolicy(t *testing.T) {
	fn := cachedHttpClientFunc(&Factory{IOStreams: &IOStreams{ErrOut: io.Discard}})
	c, _ := fn()
	if c.CheckRedirect == nil {
		t.Error("expected CheckRedirect to be set (safeRedirectPolicy)")
	}
}

func TestCachedHttpClientFunc_AgentHeaderTransportOrder(t *testing.T) {
	exttransport.Register(nil)
	t.Cleanup(func() { exttransport.Register(nil) })

	fn := cachedHttpClientFunc(&Factory{IOStreams: &IOStreams{ErrOut: io.Discard}})
	client, err := fn()
	if err != nil {
		t.Fatalf("cachedHttpClientFunc() error = %v", err)
	}
	policy, ok := client.Transport.(*internalauth.SecurityPolicyTransport)
	if !ok {
		t.Fatalf("outer transport = %T, want *auth.SecurityPolicyTransport", client.Transport)
	}
	security, ok := policy.Base.(*SecurityHeaderTransport)
	if !ok {
		t.Fatalf("layer after SecurityPolicy = %T, want *SecurityHeaderTransport", policy.Base)
	}
	if !security.DeviceInfoCollection {
		t.Fatal("default SecurityHeaderTransport device collection = false, want enabled")
	}
	agentPolicy, ok := security.Base.(*AgentHeaderPolicyTransport)
	if !ok {
		t.Fatalf("layer after SecurityHeader = %T, want *AgentHeaderPolicyTransport", security.Base)
	}
	if _, ok := agentPolicy.Base.(*RetryTransport); !ok {
		t.Fatalf("layer after AgentHeaderPolicy has type %T, want *RetryTransport", agentPolicy.Base)
	}
}
