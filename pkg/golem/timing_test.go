package golem

import "testing"

// timingChecksEnabled reports whether tests should fail on wall-clock time
// limits. The checks are skipped with -short and under the race detector,
// which slows everything down several times; the work is still run.
func timingChecksEnabled() bool {
	return !testing.Short() && !raceEnabled
}
