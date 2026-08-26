package route

import (
	"fmt"
	"sync"
)

// GrayManager tracks per-model request sequences and decides whether each
// request belongs to the gray window.
type GrayManager struct {
	mu      sync.Mutex
	windows map[string]GrayWindow
	period  int
	seq     map[string]int
}

// NewGrayManager creates a gray manager with a fixed window period.
func NewGrayManager(period int) *GrayManager {
	if period <= 0 {
		period = 100
	}
	return &GrayManager{
		windows: make(map[string]GrayWindow),
		period:  period,
		seq:     make(map[string]int),
	}
}

// SetWindow configures the gray ratio for a model.
func (g *GrayManager) SetWindow(modelName string, ratio float64) error {
	window := GrayWindow{Ratio: ratio}
	if err := window.Validate(); err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.windows[modelName] = window
	return nil
}

// Ratio returns the configured gray ratio of a model.
func (g *GrayManager) Ratio(modelName string) float64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.windows[modelName].Ratio
}

// Window returns the configured gray window of a model.
func (g *GrayManager) Window(modelName string) GrayWindow {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.windows[modelName]
}

// Period returns the configured window period.
func (g *GrayManager) Period() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.period
}

func errInvalidRatio(ratio float64) error {
	return fmt.Errorf("gray ratio %v out of range [0,1]", ratio)
}
