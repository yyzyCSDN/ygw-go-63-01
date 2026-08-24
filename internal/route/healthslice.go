package route

import "modelrouter/internal/model"

// chunkTargets splits targets into chunks of at most chunkSize. Every
// target appears in exactly one chunk, including the final partial chunk.
func chunkTargets(targets []model.Target, chunkSize int) [][]model.Target {
	if chunkSize <= 0 || len(targets) == 0 {
		return nil
	}
	chunkCount := (len(targets) + chunkSize - 1) / chunkSize
	chunks := make([][]model.Target, 0, chunkCount)
	for i := 0; i < len(targets); i += chunkSize {
		end := i + chunkSize
		if end > len(targets) {
			end = len(targets)
		}
		chunks = append(chunks, targets[i:end])
	}
	return chunks
}
