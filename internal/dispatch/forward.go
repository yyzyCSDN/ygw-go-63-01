package dispatch

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"modelrouter/internal/model"
)

const forwardPath = "/v1/infer"

// buildForwardRequest constructs the upstream HTTP request for a target.
func buildForwardRequest(ctx context.Context, target model.Target, req *model.InferenceRequest) (*http.Request, error) {
	url := "http://" + target.Addr + forwardPath
	body := bytes.NewReader(req.Payload)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, fmt.Errorf("dispatch: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/octet-stream")
	httpReq.Header.Set("X-Model", target.Model)
	httpReq.Header.Set("X-Version", target.Version)
	httpReq.Header.Set("X-Instance", target.InstanceID)
	httpReq.Header.Set("X-Request-ID", req.RequestID)
	return httpReq, nil
}
