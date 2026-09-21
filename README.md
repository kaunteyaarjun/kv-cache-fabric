# Disaggregated KV-Cache Fabric

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](go.mod)
[![Rust Version](https://img.shields.io/badge/Rust-2021-DEA584?style=flat&logo=rust)](Cargo.toml)
[![Research Paper](https://img.shields.io/badge/Paper-Research%20Paper%20(PDF%2FMD)-8A2BE2)](RESEARCH_PAPER.md)
[![Documentation](https://img.shields.io/badge/Docs-Full%20Documentation-green)](docs/FULL_DOCUMENTATION.md)
[![Architecture](https://img.shields.io/badge/Architecture-Prefill--Decode%20Disaggregation-FF6F00)](#architecture)

> **Hardware-agnostic, disaggregated KV-cache memory fabric with RAM-tiering, prefix-sharing, and PCIe DMA eviction for autonomous LLM agent swarms.**  
> *Engineered by **Somya Prasad Sethy** ([@kaunteyaarjun](https://github.com/kaunteyaarjun)).*  
> 📄 **Read the Research Paper:** [`RESEARCH_PAPER.md`](RESEARCH_PAPER.md) | 📚 **Full Documentation & Post-Mortem:** [`docs/FULL_DOCUMENTATION.md`](docs/FULL_DOCUMENTATION.md)

---

## The Problem: The KV-Cache VRAM Bottleneck

When a modern Large Language Model (e.g., LLaMA-3 70B, DeepSeek-R1) generates text, it retains attention key-value states in a data structure known as the **KV Cache**. While model weights consume a static amount of VRAM, the KV Cache grows linearly with context length and concurrent users:

$$\text{KV Cache Size} = 2 \times n_{\text{layers}} \times n_{\text{kv\_heads}} \times d_{\text{head}} \times \text{tokens} \times \text{batch\_size} \times \text{sizeof(dtype)}$$

For a 70B parameter model serving 32 concurrent agents across long conversations, the KV Cache requires **over 80GB of VRAM**. GPUs frequently hit Out-Of-Memory (OOM) not due to parameter weights, but because the attention cache exhausts GPU high-bandwidth memory.

---

## The Solution: RAM-Tiering & Disaggregation

Inspired by the architectures pioneered by **Mooncake** and **Crusoe (MemoryAlloy)**, this project builds a hardware-agnostic, high-performance KV-cache memory fabric:

1. **Prefill-Decode Disaggregation**: Decouples compute-bound prompt evaluation (prefill) from memory-bound token generation (decode).
2. **Global Prefix Sharing (Concurrent Radix Tree)**: Deduplicates attention tensors for shared system prompts, tool schemas, and multi-turn conversations using a lock-coupled prefix tree.
3. **PCIe Memory Tiering Daemon (Rust)**: Manages physical GPU VRAM (Device) and CPU DRAM (Host) memory regions. When GPU VRAM hits a 90% watermark, unpinned idle blocks are evicted across the PCIe bus to host memory via asynchronous transfers, streaming back on-demand.
4. **Inter-Process Communication (gRPC & Protobuf)**: Low-latency RPC bridge connecting Go routing engines to the Rust memory-tiering daemon.
5. **Agent Swarm Ready (OpenAI API)**: Native `/v1/chat/completions` endpoint for drop-in use with **CrewAI**, **LangGraph**, and **AutoGen**.

---

## Architecture Overview

```
                          ┌────────────────────────────────────────────────────────┐
                          │         Autonomous Agent Swarm (CrewAI / LangGraph)    │
                          └───────────────────────────┬────────────────────────────┘
                                                      │ HTTP / SSE / OpenAI API
                                                      ▼
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       GO DISAGGREGATED CONDUCTOR                                        │
│                                                                                                         │
│    ┌───────────────────────────────────┐               ┌───────────────────────────────────────────┐    │
│    │     Fine-Grained Radix Tree       │               │            Worker Pools                   │    │
│    │  (Per-Node Mutex, Lock Coupling)  │               │                                           │    │
│    │                                   │               │  ┌─────────────────┐ ┌─────────────────┐  │    │
│    │   [System Prompt: blk-1..blk-35]  │               │  │ Prefill Pool    │ │ Decode Pool     │  │    │
│    │         /               \         │               │  │ (Least Loaded)  │ │ (Least Loaded)  │  │    │
│    │   [Task A: blk-36]  [Task B: blk-37]              │  └────────┬────────┘ └────────▲────────┘  │    │
│    └─────────────────┬─────────────────┘               └───────────┼───────────────────┼───────────┘    │
│                      │ 100% Hit -> Skip Prefill                    │                   │                │
│                      │ Partial  -> Prefill Suffix                  │ Allocate/Write    │ Touch / Stream │
└──────────────────────┼─────────────────────────────────────────────┼───────────────────┼────────────────┘
                       │                                             │                   │
                       │                       gRPC IPC (Port 50051) │                   │
                       └─────────────────────────────────────────────▼───────────────────┘
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                       RUST LOCAL MEMORY DAEMON                                          │
│                                                                                                         │
│  ┌──────────────────────────────────────────────────┐  PCIe DMA Transfer  ┌──────────────────────────┐  │
│  │            Device Tier (GPU VRAM)                │ ══════════════════> │    Host Tier (CPU DRAM)  │  │
│  │  Capacity: 8 Blocks (Contiguous Vec<u8> Buffers) │  Async evict_lru    │    Capacity: 32 Blocks   │  │
│  │  High Watermark: 90% | Low Watermark: 70%        │ <══════════════════ │    Cold Block Storage    │  │
│  └──────────────────────────────────────────────────┘  Touch / Stream Back └──────────────────────────┘  │
│                                                                                                         │
│   Race-Free Eviction: Re-verifies (ref_count == 0) after write-lock reacquisition before removal        │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

---

## Core Systems Implementation

### 1. Rust Memory Manager (`main.rs`)
- **Contiguous Memory Buffers**: `PhysicalBlock` allocates real `Vec<u8>` tensor buffers (`BLOCK_BYTES = BLOCK_SIZE * BYTES_PER_TOKEN = 2048 bytes`).
- **Race-Free Eviction Protocol**: Fixes the lock-release race condition during asynchronous DMA latency simulation. Re-verifies `curr.ref_count == 0` upon acquiring `device_tier.write().await` before evicting.
- **Tonic gRPC Server**: Exposes `AllocateBlock`, `TouchBlock`, `GetBlockLocation`, `EvictLRU`, `WriteBlockData`, and `ReadBlockData`.

### 2. Fine-Grained Concurrent Radix Tree (`pkg/radixtree/`)
- **Fine-Grained Node Locking**: Replaces global mutexes with per-node `sync.RWMutex`.
- **Lock Coupling (Hand-over-Hand Traversal)**: Acquires child read-lock before releasing parent read-lock, eliminating tree-level read bottlenecks.
- **Block-Aligned Prefix Commits**: Commits block IDs at 16-token boundaries (`tokens[:16*k]`), enabling arbitrary prefix matches to recover exact physical blocks.
- **Deterministic Tokenization**: Stable subword hashing mapping prompts to token integer slices.

### 3. Disaggregated Routing Conductor (`conductor.go`)
- **Prefix Cache Matching**:
  - **100% Cache Hit**: Completely **bypasses the prefill worker pool** (`prefill_skipped: true`) and forwards directly to decode.
  - **Partial Cache Hit**: Routes only remaining suffix tokens to the prefill worker. Concatenates cached and newly prefilled block IDs.
  - **Cache Miss**: Computes full prefill, commits block IDs to the Radix Tree.
- **LRU Access Touching**: Decode workers touch referenced blocks during active generation, preventing PCIe eviction while queries are running.

### 4. Protocol Buffers & IPC (`proto/kvblock/`, `pkg/kvclient/`)
- `proto/kvblock/kvblock.proto`: Protobuf schema for tiered memory management.
- `pkg/kvclient/client.go`: High-performance Go client dialing `127.0.0.1:50051` with local tiered fallback support.

---

## Verification & Benchmarks

### 1. Multi-Agent Workload Simulator (`benchmark_agents.py`)
Autonomous agent swarms operate with long shared system prompts and branching context trees. We tested a 4-agent swarm (Planner, Coder, Auditor, Reviewer) against `http://localhost:8080/generate`:

```text
--- RUN 1: Cold System Prompt ---
[Planner]  TTFT: 27.58ms | Total: 122.58ms | Prefill Skipped: False | Cached Tokens: 0/564

--- RUN 2: Concurrent Branching Swarm ---
[Coder]    TTFT: 12.87ms | Total: 106.98ms | Prefill Skipped: False | Cached Tokens: 560/564
[Reviewer] TTFT: 14.42ms | Total: 109.49ms | Prefill Skipped: True  | Cached Tokens: 564/564
[Auditor]  TTFT: 14.82ms | Total: 109.95ms | Prefill Skipped: False | Cached Tokens: 561/566
```

#### Results:
- **53% Drop in TTFT**: Time-to-first-token fell from **27.58ms** to **12.87ms** on partial cache hits (Coder & Auditor).
- **100% Prefill Bypass**: Exact duplicates (Reviewer) completely skipped the prefill engine.
- **99.3% Prefix Cache Reuse**: 560 out of 564 tokens were served directly from the Radix Tree cache.

### 2. Hardware-Pressure Eviction Thrash Test
Configured extreme capacity constraints (8 Device blocks / 32 Host blocks) and fired **16 concurrent agent requests** requesting 32 blocks simultaneously:

```text
=== RUN   TestEvictionThrashUnderHardwarePressure
[kvclient] 🔄 PCIe DMA Eviction: moved block blk-1 from Device -> Host DRAM (Device: 5/8, Host: 1/32)
[kvclient] 🔄 PCIe DMA Eviction: moved block blk-2 from Device -> Host DRAM (Device: 5/8, Host: 2/32)
...
[kvclient] ⚡ Host DRAM Offload: allocated block blk-17 in Host DRAM (Device: 8/8, Host: 9/32)
Post-Thrash Memory Stats: Device Blocks = 8/8 | Host DRAM Blocks = 32/32
--- PASS: TestEvictionThrashUnderHardwarePressure (0.54s)
```
- **Zero Deadlocks**: Concurrent worker tasks acquired read/write locks cleanly.
- **Zero OOM**: Saturated device slots safely offloaded into Host DRAM.
- **Pinned Block Protection**: Active blocks (`ref_count > 0`) were never evicted.

---

## Quickstart

You can run the fabric either with **Docker Compose** (recommended for 1-command deployment) or **natively**.

### Option A: Docker Compose (1-Command Cluster)

Spin up the entire architecture (Rust Memory Daemon on `:50051` + Go Conductor on `:8080`):

```bash
docker compose up -d
```

Run the multi-agent swarm benchmark inside Docker:
```bash
docker compose --profile benchmark run --rm benchmark
```

View cluster logs:
```bash
docker compose logs -f
```

---

### Option B: Native Execution

#### Prerequisites
- **Go 1.22+**
- **Python 3.10+** (with `httpx`)
- **Rust 2021** (optional, for standalone Rust daemon)

#### 1. Run the Disaggregated Conductor
```bash
go run .
```
The server will start listening on port `:8080`:
- `POST http://localhost:8080/generate` (Server-Sent Events)
- `POST http://localhost:8080/v1/chat/completions` (OpenAI compatible)

#### 2. Run the Multi-Agent Benchmark
In a separate terminal:
```bash
python benchmark_agents.py
```

#### 3. Run Automated Tests
```bash
go test -v ./...
```

---

## Integration with Agent Frameworks

You can point **CrewAI**, **LangGraph**, or **AutoGen** directly at the fabric by configuring the OpenAI base URL:

```python
import os
from openai import OpenAI

# Point client to the Disaggregated KV-Cache Conductor
client = OpenAI(
    base_url="http://localhost:8080/v1",
    api_key="none",
)

response = client.chat.completions.create(
    model="disaggregated-kv-fabric",
    messages=[
        {
            "role": "system",
            "content": "System: You are an autonomous software agent equipped with tools...",
        },
        {"role": "user", "content": "Plan the microservices architecture."},
    ],
)

print(response.choices[0].message.content)
print("Cached Tokens Reused:", response.usage.prompt_tokens_details.cached_tokens)
```

---

## Project Structure

```
kv-cache-fabric/
├── conductor.go               # Disaggregated Conductor HTTP/SSE & OpenAI Gateway
├── conductor_test.go          # Multi-agent tree, thrash, and OpenAI test suite
├── benchmark_agents.py        # Autonomous agent swarm traffic simulator (Python)
├── main.rs                    # Rust tiered memory manager & Tonic gRPC service
├── Cargo.toml                 # Rust dependencies (tokio, tonic, prost)
├── build.rs                   # Rust build script compiling kvblock.proto
├── go.mod                     # Go module definitions
├── LICENSE                    # MIT License
├── pkg/
│   ├── kvclient/
│   │   └── client.go          # Go gRPC client with tiered memory simulation
│   └── radixtree/
│       ├── radix_tree.go      # Fine-grained concurrent Radix Tree (per-node RWMutex)
│       └── radix_tree_test.go # Edge-splitting and concurrent lock tests
└── proto/
    └── kvblock/
        ├── kvblock.proto      # gRPC service definition
        ├── kvblock.pb.go      # Go protobuf structures
        └── kvblock_grpc.pb.go # Go gRPC client & server stubs
```

---

## License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

**Author**: **Somya Prasad Sethy**  
**GitHub / Legacy Handle**: [@kaunteyaarjun](https://github.com/kaunteyaarjun)
