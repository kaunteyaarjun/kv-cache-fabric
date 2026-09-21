// Package radixtree implements a concurrent, fine-grained locked radix (compressed prefix) tree
// over token sequences for global KV-cache prefix sharing.
package radixtree

import (
	"hash/fnv"
	"strings"
	"sync"
)

// CacheMetadata identifies where a cached KV block lives in the fabric.
type CacheMetadata struct {
	BlockID  string `json:"block_id"`
	ServerIP string `json:"server_ip"`
}

// Node represents one edge and its subtree with fine-grained per-node locking.
// Tokens is the edge label leading into this node from its parent.
type Node struct {
	mu       sync.RWMutex
	Tokens   []int
	Metadata *CacheMetadata
	Children map[int]*Node
}

func NewNode(tokens []int, meta *CacheMetadata) *Node {
	return &Node{
		Tokens:   tokens,
		Metadata: meta,
		Children: make(map[int]*Node),
	}
}

// Tree is a concurrent radix tree using fine-grained, per-node locking.
// Instead of a single global mutex bottleneck, reads use lock-coupling
// (hand-over-hand RLock) and writes lock only the specific parent and child
// nodes undergoing structural split or modification.
type Tree struct {
	root *Node
}

// NewTree constructs an empty radix tree with a root node.
func NewTree() *Tree {
	return &Tree{root: NewNode(nil, nil)}
}

// MatchResult contains the outcome of querying the prefix tree with an incoming token sequence.
type MatchResult struct {
	// MatchedTokens is the longest prefix of query tokens found in the tree.
	MatchedTokens []int
	// BlockIDs are the cached block identifiers covering MatchedTokens in root-to-leaf order.
	BlockIDs []string
	// RemainingTokens is the suffix that is NOT cached and requires prefill computation.
	RemainingTokens []int
}

// MatchPrefix traverses the radix tree using hand-over-hand read-locking (lock coupling)
// to find the longest matching prefix without holding any global lock.
func (t *Tree) MatchPrefix(tokens []int) MatchResult {
	if len(tokens) == 0 {
		return MatchResult{}
	}

	var matched []int
	var blockIDs []string
	remaining := tokens

	curr := t.root
	curr.mu.RLock()

	for len(remaining) > 0 {
		first := remaining[0]
		child, ok := curr.Children[first]
		if !ok {
			curr.mu.RUnlock()
			break
		}

		// Hand-over-hand read locking: acquire child lock before releasing parent
		child.mu.RLock()
		curr.mu.RUnlock()
		curr = child

		common := commonPrefixLen(curr.Tokens, remaining)

		if common < len(curr.Tokens) {
			// Diverges mid-edge: the block cannot be fetched whole.
			// Stop at the last committed block boundary so RemainingTokens
			// can be cleanly computed by the prefill worker.
			curr.mu.RUnlock()
			break
		}

		// Full edge matched
		matched = append(matched, curr.Tokens...)
		if curr.Metadata != nil {
			blockIDs = append(blockIDs, curr.Metadata.BlockID)
		}
		remaining = remaining[common:]

		// If remaining is empty, release lock and return
		if len(remaining) == 0 {
			curr.mu.RUnlock()
			break
		}
	}

	return MatchResult{
		MatchedTokens:   matched,
		BlockIDs:        blockIDs,
		RemainingTokens: remaining,
	}
}

// Insert adds a token prefix with associated cache metadata into the tree.
// It uses localized two-level locking (parent + child) during descent and edge splitting,
// ensuring concurrent inserts to disjoint branches never block each other.
func (t *Tree) Insert(tokens []int, meta CacheMetadata) {
	if len(tokens) == 0 {
		return
	}

	curr := t.root
	curr.mu.Lock()
	remaining := tokens

	for {
		first := remaining[0]
		child, ok := curr.Children[first]
		if !ok {
			// No branch exists: append new leaf node directly
			curr.Children[first] = NewNode(cloneInts(remaining), &meta)
			curr.mu.Unlock()
			return
		}

		// Lock child while still holding parent lock
		child.mu.Lock()

		common := commonPrefixLen(child.Tokens, remaining)

		if common == len(child.Tokens) {
			if common == len(remaining) {
				// Exact match: update metadata
				child.Metadata = &meta
				child.mu.Unlock()
				curr.mu.Unlock()
				return
			}
			// Consumed the entire edge: advance down the tree
			remaining = remaining[common:]
			curr.mu.Unlock()
			curr = child // curr remains locked for the next loop iteration
			continue
		}

		// Partial match: split child's edge at `common`
		splitNode := NewNode(cloneInts(child.Tokens[:common]), nil)
		child.Tokens = child.Tokens[common:]
		splitNode.Children[child.Tokens[0]] = child
		curr.Children[first] = splitNode

		if common == len(remaining) {
			splitNode.Metadata = &meta
		} else {
			rem := remaining[common:]
			splitNode.Children[rem[0]] = NewNode(cloneInts(rem), &meta)
		}

		child.mu.Unlock()
		curr.mu.Unlock()
		return
	}
}

func commonPrefixLen(a, b []int) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

func cloneInts(src []int) []int {
	dst := make([]int, len(src))
	copy(dst, src)
	return dst
}

// Tokenize deterministically converts a prompt string into a slice of integer token IDs.
// It splits by whitespace and punctuation words, hashing each word into a positive integer ID.
func Tokenize(prompt string) []int {
	words := strings.Fields(prompt)
	if len(words) == 0 {
		return nil
	}

	tokens := make([]int, len(words))
	for i, w := range words {
		h := fnv.New32a()
		h.Write([]byte(strings.ToLower(w)))
		// Map hash into positive token space [1, 1000000]
		tokens[i] = int(h.Sum32()%1000000) + 1
	}
	return tokens
}
