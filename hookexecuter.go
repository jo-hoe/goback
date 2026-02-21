package gohook

import (
	"context"
	"net/http"
)

// TemplateData provides a strongly-typed wrapper for selector values
// that are made available to gohook templates as {{ .<Key> }}.
type TemplateData struct {
	Values map[string]string
}

// HookExecutor defines a simple, strongly-typed interface to execute a webhook
// using gohook under the hood.
type HookExecutor interface {
	// Execute evaluates templates against the provided data and performs the HTTP request.
	// It returns the underlying http.Response (if any), the response body bytes, and an error if execution fails.
	Execute(ctx context.Context, data TemplateData) (*http.Response, []byte, error)
}

type hookExecutor struct {
	h *Hook
}

// NewHookExecutor constructs a HookExecutor from a Config.
// If client is non-nil, it will be used via WithHTTPClient; otherwise gohook will create its own client
// using TimeoutSeconds and InsecureSkipVerify from the config.
func NewHookExecutor(cfg Config, client *http.Client) (HookExecutor, error) {
	var (
		h   *Hook
		err error
	)
	if client != nil {
		h, err = New(cfg, WithHTTPClient(client))
	} else {
		h, err = New(cfg)
	}
	if err != nil {
		return nil, err
	}
	return &hookExecutor{h: h}, nil
}

func (e *hookExecutor) Execute(ctx context.Context, data TemplateData) (*http.Response, []byte, error) {
	payload := make(map[string]any, len(data.Values))
	for k, v := range data.Values {
		payload[k] = v
	}
	return e.h.Execute(ctx, payload)
}
