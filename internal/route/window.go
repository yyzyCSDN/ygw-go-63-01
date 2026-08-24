package route

// GrayWindow describes the fraction of a request sequence that should be
// routed to the gray (new) version. The window is expressed as a ratio in
// the range [0, 1].
type GrayWindow struct {
	Ratio float64
}

// Split divides a request sequence of total requests into the number of
// gray requests and the number of base requests. The gray count never
// exceeds total, so the released ratio can never overshoot the configured
// value.
func (w GrayWindow) Split(total int) (int, int) {
	if total <= 0 {
		return 0, 0
	}
	if w.Ratio <= 0 {
		return 0, total
	}
	if w.Ratio >= 1 {
		return total, 0
	}
	gray := int(float64(total) * w.Ratio)
	if gray > total {
		gray = total
	}
	return gray, total - gray
}

// Contains reports whether the request with the given sequence number falls
// inside the gray window of a total-sized sequence.
func (w GrayWindow) Contains(seq, total int) bool {
	if seq < 0 {
		return false
	}
	gray, _ := w.Split(total)
	return seq < gray
}

// Validate checks that the window configuration is sane.
func (w GrayWindow) Validate() error {
	if w.Ratio < 0 || w.Ratio > 1 {
		return errInvalidRatio(w.Ratio)
	}
	return nil
}
