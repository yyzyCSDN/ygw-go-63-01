package route

import (
	"hash/maphash"
	"sync"

	"modelrouter/internal/metric"
	"modelrouter/internal/model"

	"github.com/cespare/xxhash/v2"
)

// Selector chooses a concrete target from the healthy chunks using a
// stable hash of the model name and request id.
type Selector struct {
	rec    metric.Recorder
	seed   maphash.Seed
	seedMu sync.Mutex
}

// NewSelector creates a selector with a random hash seed.
func NewSelector(rec metric.Recorder) *Selector {
	return &Selector{rec: rec, seed: maphash.MakeSeed()}
}

// ChunkSize returns the number of targets per chunk for a healthy list.
func (s *Selector) ChunkSize(total int) int {
	switch {
	case total <= 0:
		return 1
	case total <= 8:
		return 2
	case total <= 32:
		return 4
	default:
		return 8
	}
}

// Pick deterministically selects one target across the chunks.
func (s *Selector) Pick(chunks [][]model.Target, modelName, requestID string) model.Target {
	if len(chunks) == 0 {
		return model.Target{}
	}
	s.seedMu.Lock()
	seed := s.seed
	s.seedMu.Unlock()
	key := modelName + ":" + requestID
	hash := xxhash.Sum64String(key)
	scramble := maphash.String(seed, key)
	index := int((hash ^ scramble) % uint64(len(chunks)))
	chunk := chunks[index]
	within := int((hash >> 32) % uint64(len(chunk)))
	s.rec.Inc(metric.NameRoute)
	return chunk[within]
}
