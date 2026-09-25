import asyncio
import httpx
import json
import numpy as np
import os
import time

CONDUCTOR_URL = os.environ.get("CONDUCTOR_URL", "http://localhost:8080/generate")

# System prompt (~560 tokens)
SHARED_SYSTEM_PROMPT = "System: You are an autonomous software agent. " * 80

async def single_request(client: httpx.AsyncClient, prompt: str, max_tokens: int = 10):
    start = time.perf_counter()
    ttft = None
    tokens = []
    meta = {}

    payload = {"prompt": prompt, "max_tokens": max_tokens}
    async with client.stream("POST", CONDUCTOR_URL, json=payload, timeout=60.0) as resp:
        async for line in resp.aiter_lines():
            if line.startswith("data: ") and not meta:
                try:
                    meta = json.loads(line[6:])
                except:
                    pass
            elif line.startswith("data: ") and meta:
                if ttft is None:
                    ttft = time.perf_counter() - start
                tok = line[6:]
                if tok != "[DONE]" and tok != "{}":
                    tokens.append(tok)

    total_time = time.perf_counter() - start
    ttft_ms = (ttft * 1000) if ttft else (total_time * 1000)
    tok_per_sec = len(tokens) / (total_time - ttft) if (ttft and total_time > ttft) else 0.0

    return {
        "ttft_ms": ttft_ms,
        "total_ms": total_time * 1000,
        "tokens_gen": len(tokens),
        "tok_per_sec": tok_per_sec,
        "cached_tokens": meta.get("cached_tokens", 0),
        "num_tokens": meta.get("num_tokens", 0),
        "prefill_skipped": meta.get("prefill_skipped", False),
    }

async def run_benchmark(n_trials: int = 15):
    print(f"================================================================")
    print(f"  Disaggregated KV-Cache Fabric: Live Empirical Evaluation")
    print(f"  Target: {CONDUCTOR_URL} | Backend: smollm2-1.7b (Intel CPU / Iris Xe)")
    print(f"  Trials per Condition: N = {n_trials}")
    print(f"================================================================\n")

    cold_ttfts = []
    branch_ttfts = []
    duplicate_ttfts = []
    throughputs = []

    async with httpx.AsyncClient() as client:
        for i in range(n_trials):
            # Unique session prompt to guarantee cold start on each trial
            unique_prefix = SHARED_SYSTEM_PROMPT + f"SessionId: {time.time_ns()}_{i}. "
            
            # 1. Cold Request
            prompt_cold = unique_prefix + "Task: Plan microservices architecture."
            res_cold = await single_request(client, prompt_cold, max_tokens=10)
            cold_ttfts.append(res_cold["ttft_ms"])
            if res_cold["tok_per_sec"] > 0:
                throughputs.append(res_cold["tok_per_sec"])

            # 2. Branching Request (Partial Cache Hit)
            prompt_branch = unique_prefix + "Task: Implement Rust DMA buffer manager."
            res_branch = await single_request(client, prompt_branch, max_tokens=10)
            branch_ttfts.append(res_branch["ttft_ms"])
            if res_branch["tok_per_sec"] > 0:
                throughputs.append(res_branch["tok_per_sec"])

            # 3. Exact Duplicate (100% Cache Hit)
            res_dup = await single_request(client, prompt_cold, max_tokens=10)
            duplicate_ttfts.append(res_dup["ttft_ms"])
            if res_dup["tok_per_sec"] > 0:
                throughputs.append(res_dup["tok_per_sec"])

            print(f"Trial {i+1:2d}/{n_trials:2d} -> Cold: {res_cold['ttft_ms']:7.2f}ms | "
                  f"Branch: {res_branch['ttft_ms']:7.2f}ms (-{(1 - res_branch['ttft_ms']/res_cold['ttft_ms'])*100:4.1f}%) | "
                  f"Duplicate: {res_dup['ttft_ms']:7.2f}ms (-{(1 - res_dup['ttft_ms']/res_cold['ttft_ms'])*100:4.1f}%) | "
                  f"Tokens/s: {res_branch['tok_per_sec']:4.1f}")

    print("\n" + "="*64)
    print("  EMPIRICAL TELEMETRY SUMMARY")
    print("="*64)

    def stats(arr):
        return {
            "mean": np.mean(arr),
            "std": np.std(arr),
            "p50": np.percentile(arr, 50),
            "p95": np.percentile(arr, 95),
            "p99": np.percentile(arr, 99),
        }

    cold_s = stats(cold_ttfts)
    branch_s = stats(branch_ttfts)
    dup_s = stats(duplicate_ttfts)
    tps_s = stats(throughputs)

    print(f"Cold Start TTFT:    Mean: {cold_s['mean']:7.2f}ms | Std: {cold_s['std']:6.2f}ms | P50: {cold_s['p50']:7.2f}ms | P99: {cold_s['p99']:7.2f}ms")
    print(f"Branch (99% Hit):   Mean: {branch_s['mean']:7.2f}ms | Std: {branch_s['std']:6.2f}ms | P50: {branch_s['p50']:7.2f}ms | P99: {branch_s['p99']:7.2f}ms")
    print(f"Duplicate (100%):   Mean: {dup_s['mean']:7.2f}ms | Std: {dup_s['std']:6.2f}ms | P50: {dup_s['p50']:7.2f}ms | P99: {dup_s['p99']:7.2f}ms")
    print(f"Decode Throughput:  Mean: {tps_s['mean']:7.2f} tok/s | Std: {tps_s['std']:4.2f} tok/s")
    
    speedup_branch = (1.0 - (branch_s['mean'] / cold_s['mean'])) * 100.0
    speedup_dup = (1.0 - (dup_s['mean'] / cold_s['mean'])) * 100.0
    print(f"\nEmpirical Latency Reduction (Branching):  {speedup_branch:.1f}%")
    print(f"Empirical Latency Reduction (Duplicate):  {speedup_dup:.1f}%")
    print("="*64)

if __name__ == "__main__":
    asyncio.run(run_benchmark(n_trials=10))
