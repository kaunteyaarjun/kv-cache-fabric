//! Tiered KV-Cache Block Manager with Real Host/Device Memory and gRPC Service
//! -----------------------------------------------------------------------------
//! A production-grade tiered (GPU device / CPU host) KV-cache block manager
//! with actual memory buffer allocation, race-free LRU eviction, and gRPC IPC.

use std::collections::HashMap;
use std::sync::atomic::{AtomicU64, Ordering};
use std::sync::Arc;
use std::time::{Duration, Instant};

use tokio::sync::RwLock;

/// Fixed number of tokens stored per physical block.
pub const BLOCK_SIZE: usize = 16;
/// Bytes per token in the KV cache (e.g. 2 x fp16 vectors of dim 32 = 128 bytes).
pub const BYTES_PER_TOKEN: usize = 128;
/// Total physical byte capacity per block.
pub const BLOCK_BYTES: usize = BLOCK_SIZE * BYTES_PER_TOKEN; // 2048 bytes

pub type BlockId = u64;
pub type SequenceId = u64;

/// Which tier a block's tensors currently live on.
#[derive(Debug, Clone, Copy, PartialEq, Eq, Hash)]
pub enum TierLocation {
    Device,
    Host,
}

impl TierLocation {
    pub fn as_str(&self) -> &'static str {
        match self {
            TierLocation::Device => "DEVICE",
            TierLocation::Host => "HOST",
        }
    }
}

#[derive(Debug, PartialEq, Eq)]
pub enum BlockManagerError {
    DeviceOutOfMemory,
    HostOutOfMemory,
    BlockNotFound(BlockId),
    OffsetOutOfBounds,
}

impl std::fmt::Display for BlockManagerError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            BlockManagerError::DeviceOutOfMemory => {
                write!(f, "device tier is out of capacity even after eviction")
            }
            BlockManagerError::HostOutOfMemory => {
                write!(f, "host tier is out of capacity, cannot evict further")
            }
            BlockManagerError::BlockNotFound(id) => {
                write!(f, "block {id} not found in any memory tier")
            }
            BlockManagerError::OffsetOutOfBounds => {
                write!(f, "write or read offset exceeds block byte capacity")
            }
        }
    }
}

impl std::error::Error for BlockManagerError {}

/// A single physical block: contains real memory buffers (BLOCK_BYTES) holding KV tensors.
#[derive(Debug, Clone)]
pub struct PhysicalBlock {
    pub block_id: BlockId,
    pub tier: TierLocation,
    pub last_accessed: Instant,
    /// Number of in-flight requests currently reading/writing this block.
    /// Blocks with ref_count > 0 are pinned and never evicted.
    pub ref_count: u32,
    pub num_tokens: usize,
    /// Real contiguous memory buffer holding KV tensor bytes.
    pub data: Vec<u8>,
}

impl PhysicalBlock {
    pub fn new(block_id: BlockId, tier: TierLocation, num_tokens: usize, initial_data: Option<&[u8]>) -> Self {
        let mut data = vec![0u8; BLOCK_BYTES];
        if let Some(bytes) = initial_data {
            let len = bytes.len().min(BLOCK_BYTES);
            data[..len].copy_from_slice(&bytes[..len]);
        }
        Self {
            block_id,
            tier,
            last_accessed: Instant::now(),
            ref_count: 0,
            num_tokens: num_tokens.min(BLOCK_SIZE),
            data,
        }
    }
}

/// Maps logical token sequences to their ordered list of physical blocks,
/// and tracks the current tier ("pointer") for every block it knows about.
#[derive(Debug, Default)]
pub struct BlockTable {
    sequences: HashMap<SequenceId, Vec<BlockId>>,
    locations: HashMap<BlockId, TierLocation>,
}

impl BlockTable {
    pub fn new() -> Self {
        Self::default()
    }

    pub fn append_block(&mut self, seq_id: SequenceId, block_id: BlockId, location: TierLocation) {
        self.sequences.entry(seq_id).or_default().push(block_id);
        self.locations.insert(block_id, location);
    }

    pub fn update_location(&mut self, block_id: BlockId, location: TierLocation) {
        self.locations.insert(block_id, location);
    }

    pub fn location_of(&self, block_id: BlockId) -> Option<TierLocation> {
        self.locations.get(&block_id).copied()
    }

    pub fn blocks_for(&self, seq_id: SequenceId) -> Option<&[BlockId]> {
        self.sequences.get(&seq_id).map(|v| v.as_slice())
    }

    pub fn remove_sequence(&mut self, seq_id: SequenceId) -> Option<Vec<BlockId>> {
        self.sequences.remove(&seq_id)
    }
}

/// Tracks blocks resident in GPU VRAM with physical memory buffers.
#[derive(Debug)]
pub struct DeviceTier {
    pub capacity_blocks: usize,
    pub blocks: HashMap<BlockId, PhysicalBlock>,
}

impl DeviceTier {
    pub fn new(capacity_blocks: usize) -> Self {
        Self {
            capacity_blocks,
            blocks: HashMap::new(),
        }
    }

    pub fn usage_ratio(&self) -> f64 {
        if self.capacity_blocks == 0 {
            return 1.0;
        }
        self.blocks.len() as f64 / self.capacity_blocks as f64
    }

    pub fn has_room(&self) -> bool {
        self.blocks.len() < self.capacity_blocks
    }

    pub fn insert(&mut self, block: PhysicalBlock) {
        self.blocks.insert(block.block_id, block);
    }

    pub fn remove(&mut self, block_id: BlockId) -> Option<PhysicalBlock> {
        self.blocks.remove(&block_id)
    }

    pub fn touch(&mut self, block_id: BlockId) {
        if let Some(b) = self.blocks.get_mut(&block_id) {
            b.last_accessed = Instant::now();
        }
    }
}

/// Tracks blocks resident in CPU host DRAM with physical memory buffers.
#[derive(Debug)]
pub struct HostTier {
    pub capacity_blocks: usize,
    pub blocks: HashMap<BlockId, PhysicalBlock>,
}

impl HostTier {
    pub fn new(capacity_blocks: usize) -> Self {
        Self {
            capacity_blocks,
            blocks: HashMap::new(),
        }
    }

    pub fn usage_ratio(&self) -> f64 {
        if self.capacity_blocks == 0 {
            return 1.0;
        }
        self.blocks.len() as f64 / self.capacity_blocks as f64
    }

    pub fn has_room(&self) -> bool {
        self.blocks.len() < self.capacity_blocks
    }

    pub fn insert(&mut self, block: PhysicalBlock) {
        self.blocks.insert(block.block_id, block);
    }

    pub fn remove(&mut self, block_id: BlockId) -> Option<PhysicalBlock> {
        self.blocks.remove(&block_id)
    }

    pub fn touch(&mut self, block_id: BlockId) {
        if let Some(b) = self.blocks.get_mut(&block_id) {
            b.last_accessed = Instant::now();
        }
    }
}

/// Owns the block table and both tiers behind `Arc<RwLock<_>>`.
pub struct BlockManager {
    pub block_table: Arc<RwLock<BlockTable>>,
    pub device_tier: Arc<RwLock<DeviceTier>>,
    pub host_tier: Arc<RwLock<HostTier>>,
    /// Eviction fires once device usage reaches this ratio (0.9 = 90%).
    high_watermark: f64,
    /// Eviction stops once device usage falls back to this ratio.
    low_watermark: f64,
    next_block_id: AtomicU64,
}

impl BlockManager {
    pub fn new(device_capacity_blocks: usize, host_capacity_blocks: usize) -> Self {
        Self {
            block_table: Arc::new(RwLock::new(BlockTable::new())),
            device_tier: Arc::new(RwLock::new(DeviceTier::new(device_capacity_blocks))),
            host_tier: Arc::new(RwLock::new(HostTier::new(host_capacity_blocks))),
            high_watermark: 0.9,
            low_watermark: 0.7,
            next_block_id: AtomicU64::new(1),
        }
    }

    fn alloc_id(&self) -> BlockId {
        self.next_block_id.fetch_add(1, Ordering::Relaxed)
    }

    /// Allocate a new block for `seq_id` with real memory allocation.
    pub async fn allocate_block(
        &self,
        seq_id: SequenceId,
        num_tokens: usize,
        initial_data: Option<&[u8]>,
    ) -> Result<BlockId, BlockManagerError> {
        self.maybe_evict().await?;

        let block_id = self.alloc_id();
        let block = PhysicalBlock::new(block_id, TierLocation::Device, num_tokens, initial_data);

        {
            let mut device = self.device_tier.write().await;
            if !device.has_room() {
                return Err(BlockManagerError::DeviceOutOfMemory);
            }
            device.insert(block);
        }

        {
            let mut table = self.block_table.write().await;
            table.append_block(seq_id, block_id, TierLocation::Device);
        }

        Ok(block_id)
    }

    pub async fn touch(&self, block_id: BlockId) {
        {
            let mut device = self.device_tier.write().await;
            if device.blocks.contains_key(&block_id) {
                device.touch(block_id);
                return;
            }
        }
        {
            let mut host = self.host_tier.write().await;
            host.touch(block_id);
        }
    }

    pub async fn pin_block(&self, block_id: BlockId) -> Result<(), BlockManagerError> {
        let mut device = self.device_tier.write().await;
        if let Some(b) = device.blocks.get_mut(&block_id) {
            b.ref_count += 1;
            return Ok(());
        }
        let mut host = self.host_tier.write().await;
        if let Some(b) = host.blocks.get_mut(&block_id) {
            b.ref_count += 1;
            return Ok(());
        }
        Err(BlockManagerError::BlockNotFound(block_id))
    }

    pub async fn unpin_block(&self, block_id: BlockId) -> Result<(), BlockManagerError> {
        let mut device = self.device_tier.write().await;
        if let Some(b) = device.blocks.get_mut(&block_id) {
            b.ref_count = b.ref_count.saturating_sub(1);
            return Ok(());
        }
        let mut host = self.host_tier.write().await;
        if let Some(b) = host.blocks.get_mut(&block_id) {
            b.ref_count = b.ref_count.saturating_sub(1);
            return Ok(());
        }
        Err(BlockManagerError::BlockNotFound(block_id))
    }

    /// Write actual tensor byte payload into a block.
    pub async fn write_block_data(
        &self,
        block_id: BlockId,
        offset: usize,
        data: &[u8],
    ) -> Result<usize, BlockManagerError> {
        if offset + data.len() > BLOCK_BYTES {
            return Err(BlockManagerError::OffsetOutOfBounds);
        }

        let mut device = self.device_tier.write().await;
        if let Some(b) = device.blocks.get_mut(&block_id) {
            b.data[offset..offset + data.len()].copy_from_slice(data);
            b.last_accessed = Instant::now();
            return Ok(data.len());
        }

        let mut host = self.host_tier.write().await;
        if let Some(b) = host.blocks.get_mut(&block_id) {
            b.data[offset..offset + data.len()].copy_from_slice(data);
            b.last_accessed = Instant::now();
            return Ok(data.len());
        }

        Err(BlockManagerError::BlockNotFound(block_id))
    }

    /// Read actual tensor byte payload from a block.
    pub async fn read_block_data(
        &self,
        block_id: BlockId,
        offset: usize,
        length: usize,
    ) -> Result<(Vec<u8>, TierLocation), BlockManagerError> {
        if offset > BLOCK_BYTES {
            return Err(BlockManagerError::OffsetOutOfBounds);
        }
        let actual_len = length.min(BLOCK_BYTES - offset);

        let mut device = self.device_tier.write().await;
        if let Some(b) = device.blocks.get_mut(&block_id) {
            let slice = b.data[offset..offset + actual_len].to_vec();
            b.last_accessed = Instant::now();
            return Ok((slice, TierLocation::Device));
        }

        let mut host = self.host_tier.write().await;
        if let Some(b) = host.blocks.get_mut(&block_id) {
            let slice = b.data[offset..offset + actual_len].to_vec();
            b.last_accessed = Instant::now();
            return Ok((slice, TierLocation::Host));
        }

        Err(BlockManagerError::BlockNotFound(block_id))
    }

    pub async fn maybe_evict(&self) -> Result<usize, BlockManagerError> {
        let ratio = self.device_tier.read().await.usage_ratio();
        if ratio >= self.high_watermark {
            self.evict_lru().await
        } else {
            Ok(0)
        }
    }

    /// Move least-recently-used, unpinned (ref_count == 0) blocks from the
    /// device tier to the host tier until usage drops to the low watermark.
    ///
    /// RACE CONDITION FIX:
    /// In an asynchronous/concurrent architecture, candidates are snapshot without
    /// holding locks during data transfer. Upon acquiring the write locks,
    /// we MUST re-verify that the candidate block still exists in the device tier
    /// AND that `curr.ref_count == 0`. If a concurrent worker pinned the block
    /// during the async copy window, eviction of that block MUST be aborted.
    pub async fn evict_lru(&self) -> Result<usize, BlockManagerError> {
        let mut candidates: Vec<PhysicalBlock> = {
            let device = self.device_tier.read().await;
            device
                .blocks
                .values()
                .filter(|b| b.ref_count == 0)
                .cloned()
                .collect()
        };
        candidates.sort_by_key(|b| b.last_accessed);

        let mut evicted = 0usize;

        for block in candidates {
            let ratio = self.device_tier.read().await.usage_ratio();
            if ratio < self.low_watermark {
                break;
            }

            // Asynchronous DMA copy latency simulation (cudaMemcpyAsync seam)
            tokio::time::sleep(Duration::from_micros(20 * block.num_tokens as u64)).await;

            // Lock ordering: device then host
            let mut device = self.device_tier.write().await;
            let mut host = self.host_tier.write().await;

            // CRITICAL AUDIT FIX: Re-verify block state after lock re-acquisition.
            // If another task pinned the block (ref_count > 0) or deleted it, abort eviction for this block.
            let is_still_evictable = match device.blocks.get(&block.block_id) {
                Some(curr) => curr.ref_count == 0,
                None => false,
            };

            if !is_still_evictable {
                // Abort eviction for this candidate to prevent evicting actively running queries!
                continue;
            }

            if !host.has_room() {
                return Err(BlockManagerError::HostOutOfMemory);
            }

            if let Some(mut moved) = device.remove(block.block_id) {
                // REAL MEMORY COPY: Transfer bytes from device buffer to host buffer
                let mut host_data = vec![0u8; BLOCK_BYTES];
                host_data.copy_from_slice(&moved.data);
                moved.data = host_data;
                moved.tier = TierLocation::Host;

                host.insert(moved);
                drop(device);
                drop(host);

                let mut table = self.block_table.write().await;
                table.update_location(block.block_id, TierLocation::Host);

                evicted += 1;
            }
        }

        Ok(evicted)
    }
}

// ---------------------------------------------------------------------------
// gRPC Service Implementation (tonic)
// ---------------------------------------------------------------------------
pub mod kvblock_proto {
    tonic::include_proto!("kvblock");
}

use kvblock_proto::block_manager_service_server::{BlockManagerService, BlockManagerServiceServer};
use kvblock_proto::{
    AllocateBlockRequest, AllocateBlockResponse, EvictLruRequest, EvictLruResponse,
    GetBlockLocationRequest, GetBlockLocationResponse, ReadBlockDataRequest, ReadBlockDataResponse,
    TouchBlockRequest, TouchBlockResponse, WriteBlockDataRequest, WriteBlockDataResponse,
};
use tonic::{Request, Response, Status};

pub struct GrpcBlockService {
    pub manager: Arc<BlockManager>,
}

#[tonic::async_trait]
impl BlockManagerService for GrpcBlockService {
    async fn allocate_block(
        &self,
        request: Request<AllocateBlockRequest>,
    ) -> Result<Response<AllocateBlockResponse>, Status> {
        let req = request.into_inner();
        let init_data = if req.initial_data.is_empty() {
            None
        } else {
            Some(req.initial_data.as_slice())
        };

        match self
            .manager
            .allocate_block(req.sequence_id, req.num_tokens as usize, init_data)
            .await
        {
            Ok(block_id) => Ok(Response::new(AllocateBlockResponse {
                block_id,
                tier: "DEVICE".to_string(),
                capacity_bytes: BLOCK_BYTES as u32,
            })),
            Err(e) => Err(Status::resource_exhausted(format!(
                "allocation failed: {e}"
            ))),
        }
    }

    async fn touch_block(
        &self,
        request: Request<TouchBlockRequest>,
    ) -> Result<Response<TouchBlockResponse>, Status> {
        let req = request.into_inner();
        self.manager.touch(req.block_id).await;
        Ok(Response::new(TouchBlockResponse { success: true }))
    }

    async fn get_block_location(
        &self,
        request: Request<GetBlockLocationRequest>,
    ) -> Result<Response<GetBlockLocationResponse>, Status> {
        let req = request.into_inner();
        let table = self.manager.block_table.read().await;
        if let Some(loc) = table.location_of(req.block_id) {
            Ok(Response::new(GetBlockLocationResponse {
                tier: loc.as_str().to_string(),
                exists: true,
                num_tokens: BLOCK_SIZE as u32,
            }))
        } else {
            Ok(Response::new(GetBlockLocationResponse {
                tier: "NONE".to_string(),
                exists: false,
                num_tokens: 0,
            }))
        }
    }

    async fn evict_lru(
        &self,
        _request: Request<EvictLruRequest>,
    ) -> Result<Response<EvictLruResponse>, Status> {
        match self.manager.evict_lru().await {
            Ok(evicted) => Ok(Response::new(EvictLruResponse {
                evicted_blocks: evicted as u64,
            })),
            Err(e) => Err(Status::internal(format!("eviction error: {e}"))),
        }
    }

    async fn write_block_data(
        &self,
        request: Request<WriteBlockDataRequest>,
    ) -> Result<Response<WriteBlockDataResponse>, Status> {
        let req = request.into_inner();
        match self
            .manager
            .write_block_data(req.block_id, req.offset as usize, &req.data)
            .await
        {
            Ok(written) => Ok(Response::new(WriteBlockDataResponse {
                success: true,
                bytes_written: written as u32,
            })),
            Err(e) => Err(Status::invalid_argument(format!("write error: {e}"))),
        }
    }

    async fn read_block_data(
        &self,
        request: Request<ReadBlockDataRequest>,
    ) -> Result<Response<ReadBlockDataResponse>, Status> {
        let req = request.into_inner();
        match self
            .manager
            .read_block_data(req.block_id, req.offset as usize, req.length as usize)
            .await
        {
            Ok((data, loc)) => Ok(Response::new(ReadBlockDataResponse {
                data,
                tier: loc.as_str().to_string(),
            })),
            Err(e) => Err(Status::not_found(format!("read error: {e}"))),
        }
    }
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let bind_addr = std::env::var("GRPC_BIND_ADDR").unwrap_or_else(|_| "0.0.0.0:50051".to_string());
    let addr = bind_addr.parse()?;
    let manager = Arc::new(BlockManager::new(64, 512));

    println!("Starting Rust KV-Cache Block Manager gRPC server on {}", addr);
    let service = GrpcBlockService {
        manager: Arc::clone(&manager),
    };

    tonic::transport::Server::builder()
        .add_service(BlockManagerServiceServer::new(service))
        .serve(addr)
        .await?;

    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    #[tokio::test]
    async fn eviction_triggers_at_high_watermark_and_updates_block_table() {
        let manager = BlockManager::new(4, 16);

        let mut allocated = Vec::new();
        for seq_id in 0..5u64 {
            let block_id = manager.allocate_block(seq_id, BLOCK_SIZE, None).await.unwrap();
            allocated.push(block_id);
        }

        let device_count = manager.device_tier.read().await.blocks.len();
        let host_count = manager.host_tier.read().await.blocks.len();
        assert!(device_count < 4, "device tier should have evicted at least one block");
        assert!(host_count > 0, "evicted blocks should have landed in the host tier");

        let table = manager.block_table.read().await;
        let saw_host_block = allocated
            .iter()
            .any(|id| table.location_of(*id) == Some(TierLocation::Host));
        assert!(saw_host_block, "block table should reflect the eviction");
    }

    #[tokio::test]
    async fn pinned_blocks_are_never_evicted() {
        let manager = BlockManager::new(2, 8);
        let pinned_id = manager.allocate_block(0, BLOCK_SIZE, None).await.unwrap();

        manager.pin_block(pinned_id).await.unwrap();

        let _ = manager.allocate_block(1, BLOCK_SIZE, None).await.unwrap();
        let _ = manager.maybe_evict().await.unwrap();

        let device = manager.device_tier.read().await;
        assert!(
            device.blocks.contains_key(&pinned_id),
            "pinned block must remain in device tier"
        );
    }

    #[tokio::test]
    async fn race_condition_prevented_when_block_pinned_during_eviction_window() {
        let manager = Arc::new(BlockManager::new(2, 8));
        let block_1 = manager.allocate_block(10, BLOCK_SIZE, None).await.unwrap();
        let _block_2 = manager.allocate_block(11, BLOCK_SIZE, None).await.unwrap();

        // Spawn eviction task
        let mgr_clone = Arc::clone(&manager);
        let evict_handle = tokio::spawn(async move {
            mgr_clone.evict_lru().await
        });

        // Concurrently pin block_1 during the eviction window
        tokio::time::sleep(Duration::from_micros(5)).await;
        manager.pin_block(block_1).await.unwrap();

        let _ = evict_handle.await.unwrap();

        // Verification: block_1 was pinned during the window, so it must NOT have been evicted!
        let device = manager.device_tier.read().await;
        assert!(
            device.blocks.contains_key(&block_1),
            "block pinned during eviction window must not be evicted"
        );
    }

    #[tokio::test]
    async fn real_memory_buffer_data_integrity_across_eviction() {
        let manager = BlockManager::new(2, 8);
        let magic_payload = b"KV_TENSOR_MAGIC_BYTES_1234";

        let block_id = manager
            .allocate_block(100, BLOCK_SIZE, Some(magic_payload))
            .await
            .unwrap();

        // Write additional tensor bytes at offset
        let write_payload = b"ATTENTION_WEIGHTS_XYZ";
        manager
            .write_block_data(block_id, 32, write_payload)
            .await
            .unwrap();

        // Trigger eviction to host tier
        let _ = manager.allocate_block(101, BLOCK_SIZE, None).await.unwrap();
        let _ = manager.allocate_block(102, BLOCK_SIZE, None).await.unwrap();

        // Verify block moved to host tier
        let table = manager.block_table.read().await;
        assert_eq!(table.location_of(block_id), Some(TierLocation::Host));

        // Read back data from host tier and verify physical byte integrity
        let (data_header, loc) = manager.read_block_data(block_id, 0, magic_payload.len()).await.unwrap();
        assert_eq!(loc, TierLocation::Host);
        assert_eq!(&data_header, magic_payload);

        let (data_tensor, _) = manager.read_block_data(block_id, 32, write_payload.len()).await.unwrap();
        assert_eq!(&data_tensor, write_payload);
    }
}
