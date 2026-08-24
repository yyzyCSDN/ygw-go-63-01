package route

import "testing"

// TestBoundaryOverlapRepro isolates the window-boundary overlap defect: when a
// period rolls over, Decide must restart the sequence at 0 (the start of a
// fresh, non-overlapping window), not carry the gray tail forward.
func TestBoundaryOverlapRepro(t *testing.T) {
	const period = 10
	const ratio = 0.2
	g := NewGrayManager(period)
	if err := g.SetWindow("m", ratio); err != nil {
		t.Fatalf("set window: %v", err)
	}

	// Run a few full periods and capture the decided sequence at the boundary.
	const cycles = 4
	var seqsAtDecision []int
	var grayAtDecision []bool
	for c := 0; c < cycles; c++ {
		for i := 0; i < period; i++ {
			seq := g.Sequence("m")
			seqsAtDecision = append(seqsAtDecision, seq)
			grayAtDecision = append(grayAtDecision, g.Decide("m"))
		}
	}

	// Within each period the decided sequence should be exactly 0..period-1
	// in order, with no position repeated across the boundary.
	for c := 0; c < cycles; c++ {
		for i := 0; i < period; i++ {
			idx := c*period + i
			want := i
			if seqsAtDecision[idx] != want {
				t.Fatalf("cycle %d pos %d: decided seq=%d want=%d (boundary overlap)",
					c, i, seqsAtDecision[idx], want)
			}
		}
	}

	// The gray decisions within a period must be exactly the first
	// graySplit positions and nothing extra at the boundary.
	graySplit, _ := GrayWindow{Ratio: ratio}.Split(period)
	for c := 0; c < cycles; c++ {
		for i := 0; i < period; i++ {
			idx := c*period + i
			wantGray := i < graySplit
			if grayAtDecision[idx] != wantGray {
				t.Fatalf("cycle %d pos %d: gray=%v want=%v (boundary drift)",
					c, i, grayAtDecision[idx], wantGray)
			}
		}
	}
}
