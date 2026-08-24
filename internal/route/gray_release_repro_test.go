package route

import (
	"fmt"
	"sort"
	"testing"
)

// TestGrayReleaseRepro reproduces the reported gray-release drift: configured
// 10% but measured ~20%, with the excess clustering at each window boundary.
// It drives several full periods and asserts: (1) Split yields the configured
// fraction exactly, (2) every position 0..period-1 is visited exactly once per
// cycle (no overlap/gap at the boundary), (3) the gray set is the same
// contiguous range every cycle, (4) the steady-state ratio equals the config.
func TestGrayReleaseRepro(t *testing.T) {
	const period = 100
	const ratio = 0.1
	g := NewGrayManager(period)
	if err := g.SetWindow("m", ratio); err != nil {
		t.Fatalf("set window: %v", err)
	}

	// Split must yield the configured fraction, not double it.
	graySplit, baseSplit := GrayWindow{Ratio: ratio}.Split(period)
	t.Logf("Split(%d) = gray=%d base=%d", period, graySplit, baseSplit)
	if graySplit != 10 || baseSplit != 90 {
		t.Fatalf("Split drift: gray=%d base=%d, want 10/90", graySplit, baseSplit)
	}

	const cycles = 5
	type sample struct {
		gray bool
	}
	var samples []sample
	graySeqsByCycle := make(map[int]map[int]bool)
	seenSeqsByCycle := make(map[int]map[int]bool)
	for c := 0; c < cycles; c++ {
		graySeqsByCycle[c] = make(map[int]bool)
		seenSeqsByCycle[c] = make(map[int]bool)
		for i := 0; i < period; i++ {
			seq := g.Sequence("m") // the seq about to be decided
			isGray := g.Decide("m")
			samples = append(samples, sample{gray: isGray})
			seenSeqsByCycle[c][seq] = true
			if isGray {
				graySeqsByCycle[c][seq] = true
			}
		}
	}

	// Each cycle must visit every position 0..period-1 exactly once (no
	// overlap at the boundary, no skipped positions).
	for c := 0; c < cycles; c++ {
		seen := seenSeqsByCycle[c]
		if len(seen) != period {
			t.Fatalf("cycle %d visited %d distinct seq positions, want %d (overlap/gap at boundary)", c, len(seen), period)
		}
		for p := 0; p < period; p++ {
			if !seen[p] {
				t.Fatalf("cycle %d never visited seq %d (boundary gap)", c, p)
			}
		}
		// The gray set must be exactly {0..graySplit-1}, identical every cycle.
		got := make([]int, 0, len(graySeqsByCycle[c]))
		for s := range graySeqsByCycle[c] {
			got = append(got, s)
		}
		sort.Ints(got)
		want := make([]int, 0, graySplit)
		for i := 0; i < graySplit; i++ {
			want = append(want, i)
		}
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Fatalf("cycle %d gray seqs=%v want=%v (boundary overlap/shift)", c, got, want)
		}
	}

	// Overall ratio must equal the configured ratio strictly.
	totalGray := 0
	for _, s := range samples {
		if s.gray {
			totalGray++
		}
	}
	gotRatio := float64(totalGray) / float64(len(samples))
	if gotRatio != ratio {
		t.Fatalf("steady-state ratio=%v want=%v (gray=%d total=%d)", gotRatio, ratio, totalGray, len(samples))
	}
}
