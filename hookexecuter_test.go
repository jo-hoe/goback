package gohook

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewHookExecutor_NilClient_UsesConfigFlags(t *testing.T) {
	t.Run("InsecureSkipVerify allows TLS server", func(t *testing.T) {
		ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			_, _ = w.Write([]byte("ok"))
		}))
		defer ts.Close()

		exec, err := NewHookExecutor(Config{
			URL:                ts.URL,
			InsecureSkipVerify: true,
		}, nil)
		if err != nil {
			t.Fatalf("NewHookExecutor error: %v", err)
		}
		_, _, err = exec.Execute(context.Background(), TemplateData{})
		if err != nil {
			t.Fatalf("Execute error with insecure TLS: %v", err)
		}
	})

	t.Run("Timeout enforced on default client", func(t *testing.T) {
		rs := newRecorderServer([]int{200}, 50*time.Millisecond)
		defer rs.Close()

		exec, err := NewHookExecutor(Config{
			URL:     rs.URL(),
			Timeout: "10ms",
		}, nil)
		if err != nil {
			t.Fatalf("NewHookExecutor error: %v", err)
		}
		_, _, err = exec.Execute(context.Background(), TemplateData{})
		if err == nil || !strings.Contains(err.Error(), "Client.Timeout") {
			t.Fatalf("expected client timeout error, got %v", err)
		}
	})
}

func TestNewHookExecutor_WithCustomClient_IgnoresConfigInsecureSkipVerify(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer ts.Close()

	custom := &http.Client{} // no InsecureSkipVerify
	exec, err := NewHookExecutor(Config{
		URL:                ts.URL,
		InsecureSkipVerify: true, // should be ignored because custom client is provided
	}, custom)
	if err != nil {
		t.Fatalf("NewHookExecutor error: %v", err)
	}
	_, _, err = exec.Execute(context.Background(), TemplateData{})
	if err == nil {
		t.Fatalf("expected x509 unknown authority error with custom client")
	}
	if !strings.Contains(err.Error(), "x509") {
		t.Fatalf("expected x509 error, got %v", err)
	}
}

func TestHookExecutor_Execute_MapsTemplateDataAndDelegates(t *testing.T) {
	rs := newRecorderServer([]int{200}, 0)
	defer rs.Close()

	cfg := Config{
		URL:         rs.URL() + "/echo/{{ .id }}?q={{ .query | urlencode }}",
		Headers:     map[string]string{"X-Name": "{{ .name }}"},
		ContentType: "text/plain",
		Body:        "{{ .message }}",
		// Method intentionally left empty to rely on default (POST because Body non-empty)
	}
	exec, err := NewHookExecutor(cfg, nil)
	if err != nil {
		t.Fatalf("NewHookExecutor error: %v", err)
	}

	data := TemplateData{Values: map[string]string{
		"id":      "42",
		"query":   "hello world",
		"name":    "alice",
		"message": "hi",
	}}
	_, _, err = exec.Execute(context.Background(), data)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	reqs := rs.Requests()
	if len(reqs) != 1 {
		t.Fatalf("expected 1 request, got %d", len(reqs))
	}
	r := reqs[0]
	if r.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", r.Method)
	}
	if r.URL.Path != "/echo/42" {
		t.Fatalf("expected path /echo/42, got %s", r.URL.Path)
	}
	if q := r.URL.Query().Get("q"); q != "hello+world" && q != "hello world" {
		// Depending on server parsing, but httptest server will show '+' in raw query decode as 'hello world'
		if q != "hello world" {
			t.Fatalf("unexpected query q: %q", q)
		}
	}
	if got := r.Header.Get("X-Name"); got != "alice" {
		t.Fatalf("header X-Name mismatch: %q", got)
	}
	if got := r.Header.Get("Content-Type"); got != "text/plain" {
		t.Fatalf("Content-Type mismatch: %q", got)
	}
	if r.Body != "hi" {
		t.Fatalf("body mismatch: want %q, got %q", "hi", r.Body)
	}
}

func TestHookExecutor_Execute_StrictTemplatesMissingKeyErrors(t *testing.T) {
	exec, err := NewHookExecutor(Config{
		URL:             "http://example/{{ .missing }}",
		StrictTemplates: true,
	}, nil)
	if err != nil {
		t.Fatalf("NewHookExecutor error: %v", err)
	}
	_, _, err = exec.Execute(context.Background(), TemplateData{Values: map[string]string{}})
	if err == nil || !strings.Contains(err.Error(), "render URL") {
		t.Fatalf("expected render URL error due to missing key, got %v", err)
	}
}