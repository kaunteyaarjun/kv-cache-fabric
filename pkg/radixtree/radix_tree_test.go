package radixtree

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
)

func TestRadixTreeBasicMatching(t *testing.T) {
	tree := NewTree()

	sharedSys := []int{101, 102, 103, 104}
	userA := append(append([]int{}, sharedSys...), 201, 202)
	userB := append(append([]int{}, sharedSys...), 301, 302, 303)

	tree.Insert(sharedSys, CacheMetadata{BlockID: "blk-sys-1", ServerIP: "10.0.0.1:8080"})
	tree.Insert(userA, CacheMetadata{BlockID: "blk-userA-1", ServerIP: "10.0.0.2:8080"})

	// Query with userB: should match sharedSys prefix
	res := tree.MatchPrefix(userB)
	if !reflect.DeepEqual(res.MatchedTokens, sharedSys) {
		t.Fatalf("expected matched tokens %v, got %v", sharedSys, res.MatchedTokens)
	}
	if len(res.BlockIDs) != 1 || res.BlockIDs[0] != "blk-sys-1" {
		t.Fatalf("expected block blk-sys-1, got %v", res.BlockIDs)
	}
	expectedRemaining := []int{301, 302, 303}
	if !reflect.DeepEqual(res.RemainingTokens, expectedRemaining) {
		t.Fatalf("expected remaining tokens %v, got %v", expectedRemaining, res.RemainingTokens)
	}

	// Query with 100% exact match of userA
	resA := tree.MatchPrefix(userA)
	if !reflect.DeepEqual(resA.MatchedTokens, userA) {
		t.Fatalf("expected 100%% match for userA, got %v", resA.MatchedTokens)
	}
	if len(resA.RemainingTokens) != 0 {
		t.Fatalf("expected 0 remaining tokens for userA, got %v", resA.RemainingTokens)
	}
}

func TestRadixTreeEdgeSplitting(t *testing.T) {
	tree := NewTree()

	// Insert "ABCDE" -> [1, 2, 3, 4, 5]
	tree.Insert([]int{1, 2, 3, 4, 5}, CacheMetadata{BlockID: "blk-full", ServerIP: "10.0.0.1"})

	// Split edge by inserting "ABXYZ" -> [1, 2, 9, 8, 7]
	tree.Insert([]int{1, 2, 9, 8, 7}, CacheMetadata{BlockID: "blk-split", ServerIP: "10.0.0.2"})

	// Query [1, 2, 3, 4, 5]
	res1 := tree.MatchPrefix([]int{1, 2, 3, 4, 5})
	if len(res1.RemainingTokens) != 0 || len(res1.BlockIDs) != 1 || res1.BlockIDs[0] != "blk-full" {
		t.Fatalf("failed match after split: %+v", res1)
	}

	// Query [1, 2, 9, 8, 7]
	res2 := tree.MatchPrefix([]int{1, 2, 9, 8, 7})
	if len(res2.RemainingTokens) != 0 || len(res2.BlockIDs) != 1 || res2.BlockIDs[0] != "blk-split" {
		t.Fatalf("failed match on split branch: %+v", res2)
	}

	// Query common prefix only [1, 2]
	res3 := tree.MatchPrefix([]int{1, 2})
	if !reflect.DeepEqual(res3.MatchedTokens, []int{1, 2}) {
		t.Fatalf("failed common prefix match: %+v", res3)
	}
}

func TestConcurrentFineGrainedLocking(t *testing.T) {
	tree := NewTree()
	var wg sync.WaitGroup

	numGoroutines := 30
	opsPerGoroutine := 50

	// Concurrent writers across distinct and overlapping branches
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for op := 0; op < opsPerGoroutine; op++ {
				tokens := []int{100, gid, op, op + 1}
				meta := CacheMetadata{
					BlockID:  fmt.Sprintf("blk-%d-%d", gid, op),
					ServerIP: "10.0.0.1",
				}
				tree.Insert(tokens, meta)
			}
		}(g)
	}

	// Concurrent readers
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(gid int) {
			defer wg.Done()
			for op := 0; op < opsPerGoroutine; op++ {
				tokens := []int{100, gid, op, op + 1}
				_ = tree.MatchPrefix(tokens)
			}
		}(g)
	}

	wg.Wait()

	// Verify all entries can be retrieved
	for g := 0; g < numGoroutines; g++ {
		tokens := []int{100, g, opsPerGoroutine - 1, opsPerGoroutine}
		res := tree.MatchPrefix(tokens)
		if len(res.RemainingTokens) != 0 {
			t.Fatalf("missing token sequence for goroutine %d: %+v", g, res)
		}
	}
}

func TestDeterministicTokenizer(t *testing.T) {
	p1 := "System: You are an intelligent LLM assistant."
	p2 := "System: You are an intelligent LLM assistant. Please write code."

	t1 := Tokenize(p1)
	t2 := Tokenize(p2)

	if len(t1) >= len(t2) {
		t.Fatalf("expected t1 (%d) < t2 (%d)", len(t1), len(t2))
	}

	// Check prefix equality
	for i := range t1 {
		if t1[i] != t2[i] {
			t.Fatalf("token mismatch at index %d: %d vs %d", i, t1[i], t2[i])
		}
	}
}
