package route

import (
	"testing"
)

func TestGrayWindowRespectsRatio(t *testing.T) {
	gray := NewGrayManager(100)
	if err := gray.SetWindow("resnet", 0.1); err != nil {
		t.Fatalf("set window: %v", err)
	}
	grayCount := 0
	for i := 0; i < 100; i++ {
		if gray.Decide("resnet") {
			grayCount++
		}
	}
	if grayCount != 10 {
		t.Fatalf("gray traffic = %d of 100, want exactly 10 (10%%)", grayCount)
	}
}
