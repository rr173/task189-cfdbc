package store

import (
	"sync"
	"testing"
)

func TestTask189Bug10_ConcurrentIDGenerationIsUnique(t *testing.T) {
	const workers = 20
	gen := NewIDGen("face")
	start := make(chan struct{})
	ids := make(chan string, workers)
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			<-start
			ids <- gen.Next()
		}()
	}
	close(start)
	wg.Wait()
	close(ids)
	seen := map[string]bool{}
	for id := range ids {
		if seen[id] {
			t.Fatalf("duplicate generated id %q", id)
		}
		seen[id] = true
	}
	if len(seen) != workers {
		t.Fatalf("generated %d unique ids, want %d", len(seen), workers)
	}
}
