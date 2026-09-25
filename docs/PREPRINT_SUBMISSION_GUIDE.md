# Zenodo & TechRxiv Preprint Submission Guide

This guide contains the exact copy-paste metadata and step-by-step instructions to publish your paper to **Zenodo** (CERN) and **TechRxiv** (IEEE).

Both platforms provide permanent, citable **Digital Object Identifiers (DOIs)**, index directly on Google Scholar, and do **not** have arXiv's account endorsement gatekeeping.

---

## 1. Zenodo (CERN) Submission

Zenodo assigns an immediate, citable DOI (`10.5281/zenodo.XXXXXXX`) and permanently preserves your preprint in the CERN data center.

### Submission Steps:
1. Go to [Zenodo](https://zenodo.org/) and log in (or click **Sign in with GitHub**).
2. Click **New Upload** (or go to `https://zenodo.org/deposit/new`).
3. **Files**: Upload your compiled `paper.pdf` (and optionally `kv_cache_fabric_arxiv.zip` as supplementary material).
4. Fill in the metadata below:

### Copy-Paste Metadata for Zenodo:

- **Resource type**:
  - Type: `Publication`
  - Subtype: `Preprint` (or `Working paper`)
- **Title**:
  ```text
  Disaggregated KV-Cache Fabric: Hardware-Agnostic RAM-Tiering, PCIe DMA Eviction, and Fine-Grained Prefix Deduplication for Autonomous Agent Swarms
  ```
- **Authors**:
  - Family name: `Sethy`
  - Given name: `Somya Prasad`
  - Affiliation: `Independent Systems Research / Advanced Agentic Infrastructure`
- **Publication date**: (Today's date, e.g. `2026-09-26`)
- **Abstract / Description**:
  ```text
  Modern Large Language Model (LLM) serving systems face an acute memory wall dominated by Key-Value (KV) cache allocation rather than model weight storage. In multi-agent autonomous swarms—where dozens of concurrent agents share extensive system prompts, tool schemas, and branching conversational trajectories—the aggregate KV cache for 70B+ parameter models rapidly exceeds 80GB, triggering Out-Of-Memory (OOM) faults on modern high-bandwidth memory (HBM) accelerators.

  We present KV-Cache Fabric, an open-source, disaggregated reference architecture designed to investigate hierarchical memory tiering and concurrent prefix deduplication across three co-designed subsystems:
  1. A Tiered Physical Memory Manager in Rust, which backs logical blocks with contiguous host/device byte buffers, establishes a 90% watermark LRU eviction policy across simulated PCIe Direct Memory Access (DMA) seams, and eliminates the unlocked-snapshot race condition via optimistic two-phase write-lock re-verification.
  2. A Concurrent, Fine-Grained Radix Prefix Tree in Go, which eliminates global mutex contention through hand-over-hand lock coupling down the tree and commits physical block identifiers at strict 16-token boundaries.
  3. A Disaggregated Routing Conductor, which coordinates compute-bound prefill workers and memory-bound decode workers over gRPC, completely bypassing prefill on 100% prefix cache hits and prefilling only residual suffixes on partial hits.

  Evaluating the system under multi-agent swarm traffic shapes across both isolated control-plane dispatch benchmarks and live physical model inference with SmolLM2-1.7B-Instruct on an Intel CPU / Iris Xe platform demonstrates a 46.1% reduction in real Time-To-First-Token (TTFT) on branching agent tasks, a 12.4x reduction in latency variance (sigma = 11.19 ms vs 138.59 ms), a 51.8% TTFT reduction on duplicate queries, and sustained autoregressive decode throughput of 24.05 tokens/second with active background memory tiering.
  ```
- **Keywords / Subjects**:
  ```text
  Key-Value Cache, LLM Serving, Prefix Caching, Radix Tree, Disaggregated Inference, Memory Tiering, PCIe DMA, Autonomous Agent Swarms, PagedAttention, vLLM, SGLang, Mooncake, DistServe, Operating Systems, Computer Systems
  ```
- **License**: `Creative Commons Attribution 4.0 International` (CC-BY-4.0) or `MIT License`
- **Related Identifiers**:
  - Identifier: `https://github.com/kaunteyaarjun/kv-cache-fabric`
  - Relationship: `isSupplementTo` (or `isDocumentedBy`)
  - Resource type: `Software`
5. Click **Publish**. Your preprint will receive an immediate DOI!

---

### Pro-Tip: Zenodo Automated GitHub Release Sync
Zenodo can automatically archive and mint DOIs for every release of your repository:
1. Go to [Zenodo GitHub Settings](https://zenodo.org/account/settings/github/).
2. Enable the switch for `kaunteyaarjun/kv-cache-fabric`.
3. The included [`.zenodo.json`](file:///c:/Users/somya/Downloads/kv-cache-fabric/.zenodo.json) file automatically populates all title, author, abstract, and license metadata.
4. On GitHub, create a release (e.g. tag `v1.0.0`), and Zenodo will automatically issue a permanent DOI badge for your README!

---

## 2. TechRxiv (IEEE) Submission

TechRxiv is IEEE's official preprint server for Computer Science, Electrical Engineering, and AI. It is free, indexed by Google Scholar, and does not require arXiv endorsement.

### Submission Steps:
1. Go to [TechRxiv](https://www.techrxiv.org/) and log in with your IEEE account (free to create).
2. Click **Submit a Preprint**.
3. **Upload File**: Upload your compiled `paper.pdf`.
4. Fill in the metadata below:

### Copy-Paste Metadata for TechRxiv:

- **Preprint Title**:
  ```text
  Disaggregated KV-Cache Fabric: Hardware-Agnostic RAM-Tiering, PCIe DMA Eviction, and Fine-Grained Prefix Deduplication for Autonomous Agent Swarms
  ```
- **Subject Areas / Categories**:
  - Primary: `Computing and Processing` $\to$ `Computer Systems Organization` or `Operating Systems`
  - Secondary: `Information Systems` $\to$ `Artificial Intelligence` or `Distributed Computing`
- **Authors**:
  - Name: `Somya Prasad Sethy`
  - Affiliation: `Independent Systems Research / Advanced Agentic Infrastructure`
  - Email: `somya5400840@gmail.com`
- **Abstract**: (Use the same plain-text abstract above).
- **Keywords**:
  ```text
  Key-Value cache; Large language models; Prefix caching; Memory tiering; Disaggregated serving; Multi-agent swarms; Radix tree; PCIe DMA
  ```
- **Non-exclusive License**: Select `CC-BY 4.0` (standard for open access preprints).
- **Repository / Supplementary Link**:
  ```text
  https://github.com/kaunteyaarjun/kv-cache-fabric
  ```
5. Click **Submit for Moderation**.
6. TechRxiv reviews submissions (typically takes 1–3 business days). Upon approval, it issues an official IEEE preprint DOI (`10.36227/techrxiv.XXXXXXX`).
