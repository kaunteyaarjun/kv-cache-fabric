package kvclient

import (
	"context"
	"fmt"
	"log"
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
)

type localBlock struct {
	id           uint64
	tier         TierLocation
	data         []byte
	numTokens    uint32
	refCount     int32
	lastAccessed time.Time
}

// Client wraps the gRPC connection to the Rust BlockManagerService with a high-fidelity
// local tiered fallback that models the 8-block Device / 32-block Host memory hierarchy.
type Client struct {
	addr          string
	conn          *grpc.ClientConn
	rpc           kvblock.BlockManagerServiceClient
	fallbackLocal bool

	// Local tiered memory simulation (when gRPC server is offline)
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
	}

	if err != nil {
		log.Printf("[kvclient] notice: Rust BlockManager gRPC server at %s not reachable (%v). Using local tiered fallback (8 Device / 32 Host blocks).", addr, err)
		c.fallbackLocal = true
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

// AllocateBlock allocates a block in the Device tier, triggering LRU eviction to Host if usage >= 90%.
func (c *Client) AllocateBlock(ctx context.Context, seqID uint64, numTokens uint32, initialData []byte) (uint64, string, error) {
	if c.rpc != nil && !c.fallbackLocal {
		resp, err := c.rpc.AllocateBlock(ctx, &kvblock.AllocateBlockRequest{
			SequenceId:  seqID,
			NumTokens:   numTokens,
			InitialData: initialData,
		})
		if err == nil {
			return resp.BlockId, resp.Tier, nil
		}
		log.Printf("[kvclient] gRPC AllocateBlock failed: %v; falling back", err)
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
	if c.rpc != nil && !c.fallbackLocal {
		_, err := c.rpc.TouchBlock(ctx, &kvblock.TouchBlockRequest{BlockId: blockID})
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if blk, ok := c.deviceBlocks[blockID]; ok {
		blk.lastAccessed = time.Now();
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
	if c.rpc != nil && !c.fallbackLocal {
		resp, err := c.rpc.WriteBlockData(ctx, &kvblock.WriteBlockDataRequest{
			BlockId: blockID,
			Offset:  offset,
			Data:    data,
		})
		if err != nil {
			return err
		}
		if !resp.Success {
			return fmt.Errorf("write to block %d failed", blockID)
		}
		return nil
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
	if c.rpc != nil && !c.fallbackLocal {
		resp, err := c.rpc.ReadBlockData(ctx, &kvblock.ReadBlockDataRequest{
			BlockId: blockID,
			Offset:  offset,
			Length:  length,
		})
		if err != nil {
			return nil, "", err
		}
		return resp.Data, resp.Tier, nil
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
