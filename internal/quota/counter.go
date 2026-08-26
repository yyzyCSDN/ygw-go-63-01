package quota

// SlotCounter issues the monotonically increasing ownership tags that the
// limiter stamps onto its slots. Tags are intentionally bounded so the
// wrap-around behaviour is exercised close to the boundary; the limiter
// refuses to issue new tags once the counter approaches that boundary,
// which prevents a wrapped tag from being mistaken for a free slot.
type SlotCounter struct {
	value  uint8
	wrapAt uint8
}

// NewSlotCounter creates a counter that stops issuing tags at wrapAt.
func NewSlotCounter(wrapAt uint8) *SlotCounter {
	if wrapAt == 0 {
		wrapAt = 250
	}
	return &SlotCounter{wrapAt: wrapAt}
}

// Next returns the next tag, or wrapAt when the counter is exhausted.
func (c *SlotCounter) Next() uint8 {
	if c.value >= c.wrapAt {
		return c.wrapAt
	}
	c.value++
	return c.value
}

// NearWrap reports whether the counter is at or beyond the boundary where a
// wrapped tag would collide with the free marker.
func (c *SlotCounter) NearWrap() bool {
	return c.value >= c.wrapAt
}

// Value returns the current counter value.
func (c *SlotCounter) Value() uint8 {
	return c.value
}
