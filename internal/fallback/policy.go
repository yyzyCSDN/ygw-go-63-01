package fallback

import "fmt"

// Policy controls how many candidates the fallback executor tries.
type Policy struct {
	MaxAttempts int
}

// DefaultPolicy returns a policy with three attempts.
func DefaultPolicy() Policy {
	return Policy{MaxAttempts: 3}
}

// Validate checks that the policy is usable.
func (p Policy) Validate() error {
	if p.MaxAttempts < 1 {
		return fmt.Errorf("fallback: MaxAttempts must be at least 1")
	}
	return nil
}
