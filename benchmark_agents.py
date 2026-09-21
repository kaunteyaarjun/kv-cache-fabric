import asyncio
import httpx
import json
import os
import time

CONDUCTOR_URL = os.environ.get("CONDUCTOR_URL", "http://localhost:8080/generate")

# Shared static context representing agent rules and tool definitions (~560 tokens)
SHARED_SYSTEM_PROMPT = "System: You are an autonomous software agent. " * 80

AGENTS = [
    {"name": "Planner", "prompt": SHARED_SYSTEM_PROMPT + "Task: Plan database schema."},
    {"name": "Coder",   "prompt": SHARED_SYSTEM_PROMPT + "Task: Write Rust migrations."},
    {"name": "Auditor", "prompt": SHARED_SYSTEM_PROMPT + "Task: Audit locks and memory leaks."},
    {"name": "Reviewer","prompt": SHARED_SYSTEM_PROMPT + "Task: Plan database schema."}, # Exact duplicate of Planner
]

async def run_agent(client: httpx.AsyncClient, agent: dict):
    start_time = time.perf_counter()
    ttft = None
    meta = None
    tokens = []

    payload = {"prompt": agent["prompt"], "max_tokens": 10}

    async with client.stream("POST", CONDUCTOR_URL, json=payload, timeout=60.0) as response:
        async for line in response.aiter_lines():
            if line.startswith("event: meta"):
                continue
            elif line.startswith("data: ") and not meta:
                meta = json.loads(line.replace("data: ", ""))
            elif line.startswith("event: token"):
                continue
            elif line.startswith("data: ") and meta:
                if ttft is None:
                    ttft = time.perf_counter() - start_time
                tokens.append(line.replace("data: ", ""))

    total_time = time.perf_counter() - start_time
    ttft_ms = ttft * 1000 if ttft is not None else 0.0
    print(f"[{agent['name']}] TTFT: {ttft_ms:.2f}ms | Total: {total_time*1000:.2f}ms | "
          f"Prefill Skipped: {meta.get('prefill_skipped')} | "
          f"Cached Tokens: {meta.get('cached_tokens')}/{meta.get('num_tokens')}")

async def main():
    async with httpx.AsyncClient() as client:
        # Step 1: Cold Run
        print("--- RUN 1: Cold System Prompt ---")
        await run_agent(client, AGENTS[0])

        # Step 2: Concurrent Branching Swarm
        print("\n--- RUN 2: Concurrent Branching Swarm ---")
        await asyncio.gather(
            run_agent(client, AGENTS[1]),
            run_agent(client, AGENTS[2]),
            run_agent(client, AGENTS[3]),
        )

if __name__ == "__main__":
    asyncio.run(main())
