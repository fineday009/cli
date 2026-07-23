// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"testing"
)

func newReqDetailResponse(status int, contentType, body string) *http.Response {
	header := http.Header{}
	if contentType != "" {
		header.Set("Content-Type", contentType)
	}
	return &http.Response{
		Status:     http.StatusText(status),
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestReqDetailTransport_DisabledIsSilentAndPreservesBodies(t *testing.T) {
	t.Setenv(ReqDetailEnv, "")

	var seen string
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(req.Body)
		seen = string(data)
		return newReqDetailResponse(200, "application/json", `{"ok":true}`), nil
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/x", strings.NewReader(`{"name":"a"}`))
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if seen != `{"name":"a"}` {
		t.Fatalf("downstream saw request body %q", seen)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != `{"ok":true}` {
		t.Fatalf("caller saw response body %q", body)
	}
	if out.Len() != 0 {
		t.Fatalf("expected no output when disabled, got:\n%s", out.String())
	}
}

func TestReqDetailTransport_DumpsRequestAndResponse(t *testing.T) {
	t.Setenv(ReqDetailEnv, "1")

	var seen string
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		data, _ := io.ReadAll(req.Body)
		seen = string(data)
		return newReqDetailResponse(400, "application/json", `{"code":1254045,"msg":"field not found"}`), nil
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/base/v3/apps", strings.NewReader(`{"name":"Sales app"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Tt-Env", "boe_lane")

	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	// The body must still be readable by both the transport below and the caller.
	if seen != `{"name":"Sales app"}` {
		t.Fatalf("downstream saw request body %q", seen)
	}
	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "1254045") {
		t.Fatalf("caller saw response body %q", body)
	}

	dump := out.String()
	for _, want := range []string{
		"==> POST https://open.feishu.cn/open-apis/base/v3/apps",
		"X-Tt-Env: boe_lane",
		"request body (20 bytes):",
		`{"name":"Sales app"}`,
		"<== Bad Request POST /open-apis/base/v3/apps",
		"response body",
		"field not found",
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("dump missing %q:\n%s", want, dump)
		}
	}
}

func TestReqDetailTransport_RedactsCredentials(t *testing.T) {
	t.Setenv(ReqDetailEnv, "1")

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newReqDetailResponse(200, "application/json", `{"tenant_access_token":"t-abc123","expire":7200}`), nil
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/auth?app_secret=shhh&page_size=10",
		strings.NewReader(`{"app_id":"cli_x","app_secret":"shhh"}`))
	req.Header.Set("Authorization", "Bearer u-secret-value")
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("err=%v", err)
	}

	dump := out.String()
	if strings.Contains(dump, "shhh") || strings.Contains(dump, "u-secret-value") || strings.Contains(dump, "t-abc123") {
		t.Fatalf("dump leaked a credential:\n%s", dump)
	}
	for _, want := range []string{
		"app_secret=%3Credacted%3E", // query value redacted, URL-encoded by url.Values.Encode
		"page_size=10",              // untouched
		`"app_secret":"<redacted>"`, // request body
		"Authorization: <redacted",  // header
		`"tenant_access_token":"<redacted>"`,
	} {
		if !strings.Contains(dump, want) {
			t.Fatalf("dump missing %q:\n%s", want, dump)
		}
	}
	// Non-sensitive fields must survive so the dump stays useful.
	if !strings.Contains(dump, `"app_id":"cli_x"`) {
		t.Fatalf("dump dropped a non-sensitive field:\n%s", dump)
	}
}

// The OAuth token exchange posts client_secret as a form field, so form bodies
// need their own redaction path.
func TestReqDetailTransport_RedactsFormEncodedSecrets(t *testing.T) {
	t.Setenv(ReqDetailEnv, "1")

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newReqDetailResponse(200, "application/json", `{"code":0}`), nil
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	form := "client_id=cli_x&client_secret=0RCZTMjAr26syOX6&grant_type=client_credentials"
	req, _ := http.NewRequest("POST", "https://accounts.feishu.cn/oauth/v3/token", strings.NewReader(form))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("err=%v", err)
	}

	dump := out.String()
	if strings.Contains(dump, "0RCZTMjAr26syOX6") {
		t.Fatalf("dump leaked client_secret:\n%s", dump)
	}
	for _, want := range []string{"client_secret=<redacted>", "client_id=cli_x", "grant_type=client_credentials"} {
		if !strings.Contains(dump, want) {
			t.Fatalf("dump missing %q:\n%s", want, dump)
		}
	}
}

func TestReqDetailTransport_BinaryBodyReportedBySizeOnly(t *testing.T) {
	t.Setenv(ReqDetailEnv, "1")

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newReqDetailResponse(200, "application/json", `{"ok":true}`), nil
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/upload", bytes.NewReader([]byte{0x00, 0x01, 0x02, 0x03}))
	req.Header.Set("Content-Type", "multipart/form-data; boundary=x")
	if _, err := rt.RoundTrip(req); err != nil {
		t.Fatalf("err=%v", err)
	}

	if !strings.Contains(out.String(), "request body: <binary, 4 bytes>") {
		t.Fatalf("unexpected dump:\n%s", out.String())
	}
}

func TestReqDetailTransport_TruncatesLargeBody(t *testing.T) {
	t.Setenv(ReqDetailEnv, "1")

	large := strings.Repeat("a", reqDetailBodyLimit+100)
	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return newReqDetailResponse(200, "text/plain", large), nil
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	req, _ := http.NewRequest("GET", "https://open.feishu.cn/x", nil)
	resp, err := rt.RoundTrip(req)
	if err != nil {
		t.Fatalf("err=%v", err)
	}

	dump := out.String()
	if !strings.Contains(dump, "truncated") {
		t.Fatalf("expected a truncation marker:\n%s", dump[:200])
	}
	// Truncation is display-only: the caller still gets the whole body.
	body, _ := io.ReadAll(resp.Body)
	if len(body) != len(large) {
		t.Fatalf("caller got %d bytes, want %d", len(body), len(large))
	}
}

func TestReqDetailTransport_ReportsTransportError(t *testing.T) {
	t.Setenv(ReqDetailEnv, "1")

	base := roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return nil, io.ErrUnexpectedEOF
	})

	out := &bytes.Buffer{}
	rt := &ReqDetailTransport{Base: base, Out: out}
	req, _ := http.NewRequest("GET", "https://open.feishu.cn/x", nil)
	if _, err := rt.RoundTrip(req); err == nil {
		t.Fatal("expected the transport error to propagate")
	}
	if !strings.Contains(out.String(), "transport error") {
		t.Fatalf("unexpected dump:\n%s", out.String())
	}
}

func TestReqDetailEnabled(t *testing.T) {
	cases := map[string]bool{"": false, "0": false, " ": false, "1": true, "true": true}
	for value, want := range cases {
		t.Setenv(ReqDetailEnv, value)
		if got := reqDetailEnabled(); got != want {
			t.Errorf("%q: got %v want %v", value, got, want)
		}
	}
}
