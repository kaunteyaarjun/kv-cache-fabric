package kvclient

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"radixtree/proto/kvblock"
)

// TierLocation indicates whether a block is in simulated GPU VRAM (Device) or CPU RAM (Host).
type TierLocation string

const (
	TierDevice TierLocation = "DEVICE"
	TierHost   TierLocation = "HOST"

	// DefaultRPCTimeout guards gRPC operations from stalling indefinitely if the daemon drops.
	DefaultRPCTimeout = 250 * time.Millisecond
)

// FormatBlockID returns the canonical string representation for a physical block ID.
// Uses fast ASCII formatting to avoid reflection/fmt allocations.
func FormatBlockID(id uint64) string {
	return "blk-" + strconv.FormatUint(id, 10)
}

// ParseBlockID extracts the numeric block ID from canonical representations:
// - "blk-123" (remote Rust daemon blocks)
// - "local-blk-123" (namespaced local fallback blocks)
// - "123" (raw numeric strings)
// Returns an error if the format is invalid.
func ParseBlockID(s string) (uint64, error) {
	if strings.HasPrefix(s, "blk-") {
		return strconv.ParseUint(s[4:], 10, 64)
	}
	if strings.HasPrefix(s, "local-blk-") {
		return strconv.ParseUint(s[10:], 10, 64)
	}
	return strconv.ParseUint(s, 10, 64)
}

type localBlock struct {
	id           uint64
	tier         TierLocation
	data         []byte
	numTokens    uint32
	refCount     int32
	lastAccessed time.Time
}

// Client wraps the gRPC connection to the Rust BlockManagerService with a resilient
// local tiered fallback modeling the 8-block Device / 32-block Host memory hierarchy.
type Client struct {
	addr          string
	conn          *grpc.ClientConn
	rpc           kvblock.BlockManagerServiceClient
	fallbackLocal uint32 // 0 = active gRPC, 1 = tripped to local fallback

	// Local tiered memory simulation (when gRPC server is offline or drops mid-stream)
	mu             sync.Mutex
	deviceCapacity int
	hostCapacity   int
	deviceBlocks   map[uint64]*localBlock
	hostBlocks     map[uint64]*localBlock
	seqCounter     uint64
}

// NewClient connects to the Rust BlockManagerService at addr (e.g. "127.0.0.1:50051").
// If the gRPC server is unreachable within 500ms, it falls back to the embedded
// tiered memory manager configured for hardware-pressure testing (8 Device / 32 Host blocks).
func NewClient(addr string) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	conn, err := grpc.DialContext(
		ctx,
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)

	c := &Client{
		addr:           addr,
		deviceCapacity: 8,
		hostCapacity:   32,
		deviceBlocks:   make(map[uint64]*localBlock),
		hostBlocks:     make(map[uint64]*localBlock),
		// Offset sequence counter to prevent namespace collision with remote 1-indexed daemon IDs
		seqCounter: 1_000_000,
	}

	if err != nil {
		log.Printf("[kvclient] notice: Rust BlockManager gRPC server at %s not reachable (%v). Using local tiered fallback (8 Device / 32 Host blocks).", addr, err)
		atomic.StoreUint32(&c.fallbackLocal, 1)
		return c, nil
	}

	log.Printf("[kvclient] connected to Rust BlockManager gRPC server at %s", addr)
	c.conn = conn
	c.rpc = kvblock.NewBlockManagerServiceClient(conn)
	return c, nil
}

// Close closes the underlying gRPC connection.
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// isFallback checks if the client is operating in local fallback mode.
func (c *Client) isFallback() bool {
	return atomic.LoadUint32(&c.fallbackLocal) == 1
}

// tripCircuitBreaker atomically switches the client to local memory fallback upon connection failure.
func (c *Client) tripCircuitBreaker(err error, op string) {
	if atomic.CompareAndSwapUint32(&c.fallbackLocal, 0, 1) {
		log.Printf("[kvclient] ⚠️ Circuit breaker tripped during %s: %v. Switched to resilient local tiered engine.", op, err)
	}
}

// rpcDeadline derives a bounded execution context to guarantee no gRPC operation hangs indefinitely.
func (c *Client) rpcDeadline(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, DefaultRPCTimeout)
}

// AllocateBlock allocates a block in the Device tier, triggering LRU eviction to Host if usage >= 90%.
func (c *Client) AllocateBlock(ctx context.Context, seqID uint64, numTokens uint32, initialData []byte) (uint64, string, error) {
	if c.rpc != nil && !c.isFallback() {
		callCtx, cancel := c.rpcDeadline(ctx)
		resp, err := c.rpc.AllocateBlock(callCtx, &kvblock.AllocateBlockRequest{
			SequenceId:  seqID,
			NumTokens:   numTokens,
			InitialData: initialData,
		})
		cancel()
		if err == nil {
			return resp.BlockId, resp.Tier, nil
		}
		c.tripCircuitBreaker(err, "AllocateBlock")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	newID := atomic.AddUint64(&c.seqCounter, 1)
	dataCopy := make([]byte, len(initialData))
	copy(dataCopy, initialData)

	// High watermark check: if device usage >= 90% (8 * 0.9 = 7.2 => >= 7 blocks)
	highWatermark := int(float64(c.deviceCapacity) * 0.9)
	if len(c.deviceBlocks) >= highWatermark {
		c.localEvictLRU()
	}

	if len(c.deviceBlocks) >= c.deviceCapacity {
		c.localEvictLRU()
		if len(c.deviceBlocks) >= c.deviceCapacity {
			// Device is completely saturated with in-flight pinned blocks.
			// Direct Host DRAM offload (Mooncake/MemoryAlloy tiered fallback):
			if len(c.hostBlocks) < c.hostCapacity {
				blk := &localBlock{
					id:           newID,
					tier:         TierHost,
					data:         dataCopy,
					numTokens:    numTokens,
					refCount:     1,
					lastAccessed: time.Now(),
				}
				c.hostBlocks[newID] = blk
				log.Printf("[kvclient] ⚡ Host DRAM Offload: allocated block blk-%d in Host DRAM (Device: %d/%d, Host: %d/%d)",
					newID, len(c.deviceBlocks), c.deviceCapacity, len(c.hostBlocks), c.hostCapacity)
				return newID, string(TierHost), nil
			}
			return 0, "", fmt.Errorf("both device and host tiers out of memory")
		}
	}

	blk := &localBlock{
		id:           newID,
		tier:         TierDevice,
		data:         dataCopy,
		numTokens:    numTokens,
		refCount:     1, // Pinned during initial allocation/prefill
		lastAccessed: time.Now(),
	}
	c.deviceBlocks[newID] = blk

	return newID, string(TierDevice), nil
}

// localEvictLRU evicts unpinned (refCount == 0) blocks from Device to Host.
func (c *Client) localEvictLRU() int {
	lowWatermark := int(float64(c.deviceCapacity) * 0.7) // 8 * 0.7 = 5 blocks
	evictedCount := 0

	for len(c.deviceBlocks) > lowWatermark {
		var oldestID uint64
		var oldestTime time.Time

		for id, blk := range c.deviceBlocks {
			// Never evict pinned blocks
			if blk.refCount > 0 {
				continue
			}
			if oldestID == 0 || blk.lastAccessed.Before(oldestTime) {
				oldestID = id
				oldestTime = blk.lastAccessed
			}
		}

		if oldestID == 0 {
			// No unpinned blocks available to evict
			break
		}

		blk := c.deviceBlocks[oldestID]
		if len(c.hostBlocks) >= c.hostCapacity {
			log.Printf("[kvclient] host tier full, cannot evict further")
			break
		}

		// Transfer across simulated PCIe bus
		delete(c.deviceBlocks, oldestID)
		blk.tier = TierHost
		c.hostBlocks[oldestID] = blk
		evictedCount++
		log.Printf("[kvclient] 🔄 PCIe DMA Eviction: moved block blk-%d from Device -> Host DRAM (Device: %d/%d, Host: %d/%d)",
			oldestID, len(c.deviceBlocks), c.deviceCapacity, len(c.hostBlocks), c.hostCapacity)
	}

	return evictedCount
}

// TouchBlock refreshes the LRU timestamp and decrements refCount if generation is complete.
func (c *Client) TouchBlock(ctx context.Context, blockID uint64) error {
	if c.rpc != nil && !c.isFallback() {
		callCtx, cancel := c.rpcDeadline(ctx)
		_, err := c.rpc.TouchBlock(callCtx, &kvblock.TouchBlockRequest{BlockId: blockID})
		cancel()
		if err == nil {
			return nil
		}
		c.tripCircuitBreaker(err, "TouchBlock")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if blk, ok := c.deviceBlocks[blockID]; ok {
		blk.lastAccessed = time.Now()
		// Mark unpinned so it becomes candidate for LRU eviction once idle
		blk.refCount = 0
		return nil
	}
	if blk, ok := c.hostBlocks[blockID]; ok {
		blk.lastAccessed = time.Now()
		blk.refCount = 0
		return nil
	}
	return nil
}

// WriteBlockData stores tensor bytes in the target block.
func (c *Client) WriteBlockData(ctx context.Context, blockID uint64, offset uint32, data []byte) error {
	if c.rpc != nil && !c.isFallback() {
		callCtx, cancel := c.rpcDeadline(ctx)
		resp, err := c.rpc.WriteBlockData(callCtx, &kvblock.WriteBlockDataRequest{
			BlockId: blockID,
			Offset:  offset,
			Data:    data,
		})
		cancel()
		if err == nil && resp.Success {
			return nil
		}
		if err != nil {
			c.tripCircuitBreaker(err, "WriteBlockData")
		}
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	var blk *localBlock
	if b, ok := c.deviceBlocks[blockID]; ok {
		blk = b
	} else if b, ok := c.hostBlocks[blockID]; ok {
		blk = b
	}

	if blk != nil {
		if int(offset)+len(data) > len(blk.data) {
			newBuf := make([]byte, int(offset)+len(data))
			copy(newBuf, blk.data)
			blk.data = newBuf
		}
		copy(blk.data[offset:], data)
		blk.lastAccessed = time.Now()
	}
	return nil
}

// ReadBlockData retrieves tensor bytes from the target block.
func (c *Client) ReadBlockData(ctx context.Context, blockID uint64, offset, length uint32) ([]byte, string, error) {
	if c.rpc != nil && !c.isFallback() {
		callCtx, cancel := c.rpcDeadline(ctx)
		resp, err := c.rpc.ReadBlockData(callCtx, &kvblock.ReadBlockDataRequest{
			BlockId: blockID,
			Offset:  offset,
			Length:  length,
		})
		cancel()
		if err == nil {
			return resp.Data, resp.Tier, nil
		}
		c.tripCircuitBreaker(err, "ReadBlockData")
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if blk, ok := c.deviceBlocks[blockID]; ok {
		return blk.data, string(blk.tier), nil
	}
	if blk, ok := c.hostBlocks[blockID]; ok {
		return blk.data, string(blk.tier), nil
	}
	return nil, "", fmt.Errorf("block %d not found", blockID)
}

// Stats returns the current block counts in Device and Host tiers.
func (c *Client) Stats() (deviceCount, hostCount int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.deviceBlocks), len(c.hostBlocks)
}
