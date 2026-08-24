package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// newRequestID returns a random request identifier.
func newRequestID() string {
	var buffer [8]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buffer[:])
}

func timeSince(start time.Time) string {
	elapsed := time.Since(start)
	if elapsed < time.Second {
		return "0s"
	}
	return elapsed.Truncate(time.Second).String()
}
