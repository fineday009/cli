// Copyright (c) 2026 Lark Technologies Pte. Ltd.
// SPDX-License-Identifier: MIT

package cmdutil

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/larksuite/cli/internal/transport"
)

// ReqDetailEnv, when set to a truthy value (any non-empty non-"0" string),
// makes ReqDetailTransport dump the full request and response — request line,
// headers and body, then response status, headers and body — to stderr.
//
// LARK_CLI_DEBUG_HEADERS only prints outbound headers; this one adds both
// bodies, which is what you need when the backend rejects a payload and the
// error message alone does not say why. Credentials in headers, query strings
// and JSON bodies are redacted.
const ReqDetailEnv = "LARK_CLI_SHOW_REQ_DETAIL"

// reqDetailBodyLimit caps how much of each body is printed. Bodies larger than
// this are truncated with a trailing marker; the full byte count is reported.
const reqDetailBodyLimit = 64 * 1024

// sensitiveQueryParams are redacted in the dumped URL. Matching is a substring
// check on the lowercased parameter name, so "app_secret" and "access_token"
// are both covered.
var sensitiveQueryParams = []string{"secret", "token", "password", "code"}

// sensitiveBodyKeys are JSON keys whose string values are redacted in dumped
// bodies.
var sensitiveBodyKeys = []string{
	"app_secret", "appsecret", "access_token", "refresh_token",
	"tenant_access_token", "app_access_token", "user_access_token",
	"password", "secret", "client_secret",
}

// jsonSecretPattern matches "key": "value" for the keys above, capturing the
// key so the replacement can keep it.
var jsonSecretPattern = regexp.MustCompile(
	`(?i)"(` + strings.Join(sensitiveBodyKeys, "|") + `)"\s*:\s*"[^"]*"`)

// ReqDetailTransport dumps full request/response details when ReqDetailEnv is set.
// It sits at the innermost position of the transport chain, so the headers it
// prints are the ones that actually go on the wire.
type ReqDetailTransport struct {
	Base http.RoundTripper
	Out  io.Writer // defaults to os.Stderr
}

func (t *ReqDetailTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return transport.Fallback()
}

func (t *ReqDetailTransport) out() io.Writer {
	if t.Out != nil {
		return t.Out
	}
	return os.Stderr
}

func reqDetailEnabled() bool {
	v := strings.TrimSpace(os.Getenv(ReqDetailEnv))
	return v != "" && v != "0"
}

// RoundTrip implements http.RoundTripper.
func (t *ReqDetailTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if !reqDetailEnabled() {
		return t.base().RoundTrip(req)
	}

	reqBody, req := drainRequestBody(req)

	var b strings.Builder
	fmt.Fprintf(&b, "[lark-cli] ==> %s %s\n", req.Method, redactURL(req.URL))
	writeDumpHeaders(&b, req.Header)
	writeDumpBody(&b, "request body", req.Header.Get("Content-Type"), reqBody)
	fmt.Fprint(t.out(), b.String())

	resp, err := t.base().RoundTrip(req)
	if err != nil {
		fmt.Fprintf(t.out(), "[lark-cli] <== transport error: %v\n", err)
		return resp, err
	}
	if resp == nil {
		return resp, err
	}

	respBody, resp := drainResponseBody(resp)

	var rb strings.Builder
	fmt.Fprintf(&rb, "[lark-cli] <== %s %s %s\n", resp.Status, req.Method, req.URL.Path)
	writeDumpHeaders(&rb, resp.Header)
	writeDumpBody(&rb, "response body", resp.Header.Get("Content-Type"), respBody)
	fmt.Fprint(t.out(), rb.String())

	return resp, nil
}

// drainRequestBody reads the request body and returns a request whose body can
// still be consumed downstream. The original request is left untouched.
func drainRequestBody(req *http.Request) ([]byte, *http.Request) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, req
	}
	data, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, req
	}
	clone := req.Clone(req.Context())
	clone.Body = io.NopCloser(bytes.NewReader(data))
	clone.GetBody = func() (io.ReadCloser, error) {
		return io.NopCloser(bytes.NewReader(data)), nil
	}
	clone.ContentLength = int64(len(data))
	return data, clone
}

// drainResponseBody reads the response body and puts an equivalent reader back,
// so callers downstream still see an unread body.
func drainResponseBody(resp *http.Response) ([]byte, *http.Response) {
	if resp.Body == nil || resp.Body == http.NoBody {
		return nil, resp
	}
	data, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		resp.Body = io.NopCloser(bytes.NewReader(data))
		return nil, resp
	}
	resp.Body = io.NopCloser(bytes.NewReader(data))
	return data, resp
}

func writeDumpHeaders(b *strings.Builder, header http.Header) {
	keys := make([]string, 0, len(header))
	for k := range header {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		val := strings.Join(header[k], ", ")
		if sensitiveHeaders[strings.ToLower(k)] {
			val = fmt.Sprintf("<redacted, %d chars>", len(val))
		}
		fmt.Fprintf(b, "[lark-cli]     %s: %s\n", k, val)
	}
}

func writeDumpBody(b *strings.Builder, label, contentType string, body []byte) {
	if len(body) == 0 {
		return
	}
	if !printableBody(contentType, body) {
		fmt.Fprintf(b, "[lark-cli]     %s: <binary, %d bytes>\n", label, len(body))
		return
	}
	text := redactBody(contentType, string(body))
	suffix := ""
	if len(body) > reqDetailBodyLimit {
		text = text[:reqDetailBodyLimit]
		suffix = fmt.Sprintf(" ... <truncated, %d bytes total>", len(body))
	}
	fmt.Fprintf(b, "[lark-cli]     %s (%d bytes):\n", label, len(body))
	for _, line := range strings.Split(text, "\n") {
		fmt.Fprintf(b, "[lark-cli]       %s\n", line)
	}
	if suffix != "" {
		fmt.Fprintf(b, "[lark-cli]      %s\n", suffix)
	}
}

// printableBody reports whether a body is safe to print as text. Multipart and
// octet-stream uploads are large and binary, so they are reported by size only;
// anything containing a NUL byte is treated the same way.
func printableBody(contentType string, body []byte) bool {
	ct := strings.ToLower(contentType)
	switch {
	case strings.Contains(ct, "multipart/"),
		strings.Contains(ct, "octet-stream"),
		strings.HasPrefix(ct, "image/"),
		strings.HasPrefix(ct, "audio/"),
		strings.HasPrefix(ct, "video/"):
		return false
	}
	probe := body
	if len(probe) > 1024 {
		probe = probe[:1024]
	}
	return !bytes.ContainsRune(probe, 0)
}

// redactBody masks credential-bearing values. Form-encoded bodies (the OAuth
// token exchange posts client_secret this way) are redacted by parameter name;
// everything else is treated as JSON.
func redactBody(contentType, body string) string {
	if strings.Contains(strings.ToLower(contentType), "x-www-form-urlencoded") {
		return redactFormBody(body)
	}
	return jsonSecretPattern.ReplaceAllString(body, `"$1":"<redacted>"`)
}

// redactFormBody masks sensitive parameters in a form-encoded body, preserving
// parameter order so the dump still lines up with what was sent.
func redactFormBody(body string) string {
	parts := strings.Split(body, "&")
	for i, part := range parts {
		eq := strings.Index(part, "=")
		if eq < 0 {
			continue
		}
		key := part[:eq]
		decoded, err := url.QueryUnescape(key)
		if err != nil {
			decoded = key
		}
		if isSensitiveParamName(decoded) {
			parts[i] = key + "=<redacted>"
		}
	}
	return strings.Join(parts, "&")
}

// isSensitiveParamName reports whether a query or form parameter name looks
// credential-bearing.
func isSensitiveParamName(name string) bool {
	lower := strings.ToLower(name)
	for _, needle := range sensitiveQueryParams {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return false
}

// redactURL masks credential-bearing query parameters, keeping everything else
// byte-for-byte so the printed URL still matches what was sent.
func redactURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if u.RawQuery == "" {
		return u.String()
	}
	values, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return u.String()
	}
	redacted := false
	for key := range values {
		if isSensitiveParamName(key) {
			values.Set(key, "<redacted>")
			redacted = true
		}
	}
	if !redacted {
		return u.String()
	}
	clone := *u
	clone.RawQuery = values.Encode()
	return clone.String()
}
