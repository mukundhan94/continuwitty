package chat

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func BenchmarkBuildContradictionWarnings50Paths(b *testing.B) {
	paths := benchmarkTracePaths(50)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildContradictionWarnings(paths)
	}
}

func BenchmarkBuildContradictionWarnings200Paths(b *testing.B) {
	paths := benchmarkTracePaths(200)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildContradictionWarnings(paths)
	}
}

func benchmarkTracePaths(count int) []EngramTracePath {
	paths := make([]EngramTracePath, 0, count)
	for index := 0; index < count; index++ {
		rootID := benchmarkUUID(index + 1)
		targetID := benchmarkUUID(index + 10_000)
		path := EngramTracePath{
			RootEngramID:   rootID,
			TargetEngramID: targetID,
			Depth:          (index % 3) + 1,
		}
		if index%3 == 0 {
			path.HasContradiction = true
			path.ContradictingLinkIDs = []uuid.UUID{
				benchmarkUUID(index + 20_000),
				benchmarkUUID(index + 30_000),
			}
		}
		paths = append(paths, path)
	}
	return paths
}

func benchmarkUUID(seed int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-0000-0000-%012x", seed))
}
