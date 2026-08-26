package model

import "time"

// InferenceRequest is a routed inference call that enters the gateway.
type InferenceRequest struct {
	Model     string
	RequestID string
	Payload   []byte
	Deadline  time.Duration
}

// Target identifies one concrete serving endpoint selected by the route
// table.
type Target struct {
	Model      string
	Version    string
	InstanceID string
	Addr       string
	Gray       bool
}

// Response is the result of forwarding a request to a target.
type Response struct {
	Status int
	Body   []byte
	Target Target
}

// Success reports whether the backend answered with a 2xx status.
func (r Response) Success() bool {
	return r.Status >= 200 && r.Status < 300
}

// RouteResult is what the route table produces for one request.
type RouteResult struct {
	Model     string
	Version   string
	Target    Target
	Fallbacks []Target
	Gray      bool
}

// Candidates returns the primary target followed by fallbacks.
func (r RouteResult) Candidates() []Target {
	out := make([]Target, 0, 1+len(r.Fallbacks))
	out = append(out, r.Target)
	out = append(out, r.Fallbacks...)
	return out
}
