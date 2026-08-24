package store

import (
	"sync"
	"testing"
)

// TestIDGenNextConcurrentUnique 验证并发调用 Next 时返回的 ID 全部唯一。
// 修复前 Next 未持锁，多 goroutine 同时自增 seq 会读到相同值，
// 产生重复 ID；修复后必须 0 重复。
func TestIDGenNextConcurrentUnique(t *testing.T) {
	const goroutines = 64
	const perG = 200
	g := NewIDGen("reg")

	seen := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			local := make([]string, 0, perG)
			for j := 0; j < perG; j++ {
				local = append(local, g.Next())
			}
			mu.Lock()
			for _, id := range local {
				seen[id] = struct{}{}
			}
			mu.Unlock()
		}()
	}
	wg.Wait()

	want := goroutines * perG
	if len(seen) != want {
		t.Fatalf("got %d unique IDs, want %d (duplicates present)", len(seen), want)
	}
}

// TestIDGenNextSequence 确保持锁后单线程递增序列仍正确。
func TestIDGenNextSequence(t *testing.T) {
	g := NewIDGen("bc")
	for i, want := range []string{"bc-1", "bc-2", "bc-3"} {
		if got := g.Next(); got != want {
			t.Fatalf("call %d: got %q want %q", i, got, want)
		}
	}
}
