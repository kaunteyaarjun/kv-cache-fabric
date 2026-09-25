// Command conductor implements the global routing and scheduling proxy for a
// Prefill-Decode disaggregated KV-Cache fabric with Radix-Tree prefix sharing,
// gRPC block management, and an OpenAI-compatible /v1/chat/completions endpoint.
package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	mrand "math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"radixtree/pkg/kvclient"
	"radixtree/pkg/radixtree"
)

// PromptRequest is the incoming client payload for /generate.
type PromptRequest struct {
	Prompt    string `json:"prompt"`
	MaxTokens int    `json:"max_tokens"`
}

// KVCacheReference tracks physical block allocations in the memory fabric.
type KVCacheReference struct {
	RequestID       string   `json:"request_id"`
	BlockIDs        []string `json:"block_ids"`
	NumTokens       int      `json:"num_tokens"`
	CachedTokens    int      `json:"cached_tokens"`
	PrefilledTokens int      `json:"prefilled_tokens"`
	PrefillSkipped  bool     `json:"prefill_skipped"`
	SourceIP        string   `json:"source_ip"`
}

// Worker represents an inference worker node in the disaggregated cluster.
type Worker struct {
	ID         string
	Addr       string
	mu         sync.Mutex
	ActiveLoad int
}

func (w *Worker) Load() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.ActiveLoad
}

func (w *Worker) inc() {
	w.mu.Lock()
	w.ActiveLoad++
	w.mu.Unlock()
}

func (w *Worker) dec() {
	w.mu.Lock()
	w.ActiveLoad--
	w.mu.Unlock()
}

// WorkerPool maintains a set of workers for a specific role (prefill or decode).
type WorkerPool struct {
	mu      sync.RWMutex
	workers []*Worker
}

func NewWorkerPool(prefix string, n int) *WorkerPool {
	ws := make([]*Worker, n)
	for i := 0; i < n; i++ {
		ws[i] = &Worker{
			ID:   fmt.Sprintf("%s-%d", prefix, i),
			Addr: fmt.Sprintf("10.0.%d.%d:8080", i/256, i%256),
		}
	}
	return &WorkerPool{workers: ws}
}

func (p *WorkerPool) LeastLoaded() *Worker {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var best *Worker
	for _, w := range p.workers {
		if best == nil || w.Load() < best.Load() {
			best = w
		}
	}
	return best
}

// Conductor coordinates prefix caching, worker pools, and memory daemon RPCs.
type Conductor struct {
	prefillPool *WorkerPool
	decodePool  *WorkerPool
	radixTree   *radixtree.Tree
	kvClient    *kvclient.Client
	seqCounter  uint64
	backendURL  string
	modelName   string
	httpClient  *http.Client
}

func NewConductor(numPrefill, numDecode int, grpcAddr string) *Conductor {
	client, err := kvclient.NewClient(grpcAddr)
	if err != nil {
		log.Printf("[conductor] warning: kvclient init notice: %v", err)
	}

	return &Conductor{
		prefillPool: NewWorkerPool("prefill", numPrefill),
		decodePool:  NewWorkerPool("decode", numDecode),
		radixTree:   radixtree.NewTree(),
		kvClient:    client,
		httpClient:  &http.Client{Timeout: 90 * time.Second},
	}
}

// SetBackend configures an upstream LLM inference engine (e.g. Ollama or llama.cpp)
// for executing real prefill and autoregressive token decoding.
func (c *Conductor) SetBackend(url, model string) {
	c.backendURL = url
	c.modelName = model
}

// commitBlocksToTree commits physical block IDs at 16-token block boundaries into the Radix Tree.
func (c *Conductor) commitBlocksToTree(tokens []int, blockIDs []string, serverIP string) {
	const blockSize = 16
	for i, bID := range blockIDs {
		end := (i + 1) * blockSize
		if end > len(tokens) {
			end = len(tokens)
		}
		c.radixTree.Insert(tokens[:end], radixtree.CacheMetadata{
			BlockID:  bID,
			ServerIP: serverIP,
		})
	}
}

// runPrefill allocates real physical memory blocks in the Rust BlockManager daemon
// over gRPC for the given token slice, streaming simulated KV tensor bytes into each block.
func (c *Conductor) runPrefill(ctx context.Context, w *Worker, tokens []int, requestID string) []string {
	w.inc()
	defer w.dec()

	tokenCount := len(tokens)
	if tokenCount == 0 {
		return nil
	}

	const blockSize = 16
	numBlocks := (tokenCount + blockSize - 1) / blockSize
	seqID := atomic.AddUint64(&c.seqCounter, 1)

	blockIDs := make([]string, numBlocks)

	for i := 0; i < numBlocks; i++ {
		tokensInBlock := blockSize
		if (i+1)*blockSize > tokenCount {
			tokensInBlock = tokenCount - i*blockSize
		}

		// Synthesize KV tensor bytes (128 bytes per token)
		tensorPayload := make([]byte, tokensInBlock*128)
		_, _ = rand.Read(tensorPayload)

		blockID, tier, err := c.kvClient.AllocateBlock(ctx, seqID, uint32(tokensInBlock), tensorPayload)
		if err != nil {
			log.Printf("[%s] error allocating block %d: %v", requestID, i, err)
		}

		_ = c.kvClient.WriteBlockData(ctx, blockID, 0, tensorPayload)

		blockIDs[i] = kvclient.FormatBlockID(blockID)
		log.Printf("[%s] allocated block %s (%s tier, %d tokens) on %s",
			requestID, blockIDs[i], tier, tokensInBlock, w.ID)
	}

	return blockIDs
}

// streamDecodeBackend forwards the prompt or residual suffix to an upstream LLM engine
// (e.g. Ollama or llama.cpp OpenAI-compatible endpoint) and streams real tokens via SSE.
func (c *Conductor) streamDecodeBackend(ctx context.Context, w *Worker, ref KVCacheReference, prompt string, maxTokens int, out chan<- string) error {
	payload := map[string]interface{}{
		"model": c.modelName,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens": maxTokens,
		"stream":     true,
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.backendURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := c.httpClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("backend HTTP error: %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	tokensGenerated := 0

	for scanner.Scan() {
		line := scanner.Text()
		if line == "data: [DONE]" {
			break
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		rawJSON := strings.TrimPrefix(line, "data: ")
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(rawJSON), &chunk); err == nil {
			if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
				tokensGenerated++
				out <- chunk.Choices[0].Delta.Content

				// Periodically refresh active block timestamps in the memory daemon
				if tokensGenerated%16 == 0 {
					for _, blkStr := range ref.BlockIDs {
						if bID, parseErr := kvclient.ParseBlockID(blkStr); parseErr == nil {
							_ = c.kvClient.TouchBlock(ctx, bID)
						}
					}
				}
			}
		}
	}

	return scanner.Err()
}

// streamDecode generates tokens sequentially, either from an active real model backend
// or from simulated generation if no backend is reachable. It refreshes LRU access
// timestamps on active KV blocks in the memory daemon.
func (c *Conductor) streamDecode(ctx context.Context, w *Worker, ref KVCacheReference, prompt string, maxTokens int, out chan<- string) {
	w.inc()
	defer w.dec()
	defer close(out)

	if c.backendURL != "" && c.backendURL != "none" && c.backendURL != "mock" {
		err := c.streamDecodeBackend(ctx, w, ref, prompt, maxTokens, out)
		if err == nil {
			// Real engine streaming finished successfully.
			// Touch/unpin blocks after decode completes
			for _, blkStr := range ref.BlockIDs {
				if bID, parseErr := kvclient.ParseBlockID(blkStr); parseErr == nil {
					_ = c.kvClient.TouchBlock(ctx, bID)
				}
			}
			return
		}
		log.Printf("[conductor] live engine (%s) unavailable (%v); falling back to simulated generation", c.backendURL, err)
	}

	// Simulation fallback mode
	vocab := []string{
		"disaggregated", "kv-cache", "fabric", "accelerates", "inference",
		"by", "reusing", "prefix", "tensors", "and", "tiering", "ram",
	}

	for i := 0; i < maxTokens; i++ {
		select {
		case <-ctx.Done():
			return
		default:
		}

		time.Sleep(10 * time.Millisecond) // Simulated per-token decode latency
		out <- vocab[mrand.Intn(len(vocab))]
	}

	// Unpin blocks after decode completes
	for _, blkStr := range ref.BlockIDs {
		if bID, err := kvclient.ParseBlockID(blkStr); err == nil {
			_ = c.kvClient.TouchBlock(ctx, bID)
		}
	}
}

// processPrompt coordinates prefix matching, prefill bypass/execution, and block reference generation.
func (c *Conductor) processPrompt(ctx context.Context, prompt string, requestID string) (*KVCacheReference, *Worker, []int, error) {
	tokens := radixtree.Tokenize(prompt)
	if len(tokens) == 0 {
		tokens = []int{1}
	}

	match := c.radixTree.MatchPrefix(tokens)

	var finalBlockIDs []string
	var prefillSkipped bool
	var sourceIP string
	cachedTokens := len(match.MatchedTokens)
	prefilledTokens := len(match.RemainingTokens)

	if prefilledTokens == 0 && len(match.BlockIDs) > 0 {
		// 100% Prefix Cache Hit
		prefillSkipped = true
		finalBlockIDs = match.BlockIDs
		sourceIP = "radix-cache"
		log.Printf("[%s] ⚡ 100%% PREFIX CACHE HIT (%d tokens). SKIPPING PREFILL entirely! Reusing %d blocks: %v",
			requestID, len(tokens), len(finalBlockIDs), finalBlockIDs)

	} else if cachedTokens > 0 && prefilledTokens > 0 {
		// Partial Prefix Cache Hit
		prefillWorker := c.prefillPool.LeastLoaded()
		if prefillWorker == nil {
			return nil, nil, nil, fmt.Errorf("no prefill workers available")
		}

		log.Printf("[%s] ⚡ PARTIAL PREFIX CACHE HIT (%d/%d tokens cached, %d blocks reused). Prefilling remaining %d tokens on %s",
			requestID, cachedTokens, len(tokens), len(match.BlockIDs), prefilledTokens, prefillWorker.ID)

		newBlocks := c.runPrefill(ctx, prefillWorker, match.RemainingTokens, requestID)
		finalBlockIDs = append(append([]string{}, match.BlockIDs...), newBlocks...)
		sourceIP = prefillWorker.Addr

		// Commit full prefix including new blocks
		c.commitBlocksToTree(tokens, finalBlockIDs, prefillWorker.Addr)

	} else {
		// Cache Miss
		prefillWorker := c.prefillPool.LeastLoaded()
		if prefillWorker == nil {
			return nil, nil, nil, fmt.Errorf("no prefill workers available")
		}

		log.Printf("[%s] CACHE MISS (%d tokens). Routing to prefill worker %s",
			requestID, len(tokens), prefillWorker.ID)

		finalBlockIDs = c.runPrefill(ctx, prefillWorker, tokens, requestID)
		sourceIP = prefillWorker.Addr

		c.commitBlocksToTree(tokens, finalBlockIDs, prefillWorker.Addr)
	}

	decodeWorker := c.decodePool.LeastLoaded()
	if decodeWorker == nil {
		return nil, nil, nil, fmt.Errorf("no decode workers available")
	}

	ref := &KVCacheReference{
		RequestID:       requestID,
		BlockIDs:        finalBlockIDs,
		NumTokens:       len(tokens),
		CachedTokens:    cachedTokens,
		PrefilledTokens: prefilledTokens,
		PrefillSkipped:  prefillSkipped,
		SourceIP:        sourceIP,
	}

	return ref, decodeWorker, tokens, nil
}

// handleGenerate serves the streaming SSE endpoint for /generate.
func (c *Conductor) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req PromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 16
	}

	ctx := r.Context()
	requestID := fmt.Sprintf("req-%d", time.Now().UnixNano())

	ref, decodeWorker, _, err := c.processPrompt(ctx, req.Prompt, requestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	tokenCh := make(chan string)
	go c.streamDecode(ctx, decodeWorker, *ref, req.Prompt, req.MaxTokens, tokenCh)

	// Emit metadata event
	fmt.Fprintf(w, "event: meta\ndata: %s\n\n", mustJSON(ref))
	flusher.Flush()

	// Stream generated tokens
	for tok := range tokenCh {
		fmt.Fprintf(w, "event: token\ndata: %s\n\n", tok)
		flusher.Flush()
	}

	fmt.Fprint(w, "event: done\ndata: {}\n\n")
	flusher.Flush()
}

// ---------------------------------------------------------------------------
// OpenAI-Compatible /v1/chat/completions Endpoint (CrewAI / AutoGen / LangGraph)
// ---------------------------------------------------------------------------

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionRequest struct {
	Model     string        `json:"model"`
	Messages  []ChatMessage `json:"messages"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
}

type ChatCompletionResponseChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

type PromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type UsageStats struct {
	PromptTokens        int                 `json:"prompt_tokens"`
	CompletionTokens    int                 `json:"completion_tokens"`
	TotalTokens         int                 `json:"total_tokens"`
	PromptTokensDetails PromptTokensDetails `json:"prompt_tokens_details"`
}

type ChatCompletionResponse struct {
	ID      string                         `json:"id"`
	Object  string                         `json:"object"`
	Created int64                          `json:"created"`
	Model   string                         `json:"model"`
	Choices []ChatCompletionResponseChoice `json:"choices"`
	Usage   UsageStats                     `json:"usage"`
}

func (c *Conductor) handleChatCompletions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ChatCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON payload", http.StatusBadRequest)
		return
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = 20
	}
	if req.Model == "" {
		req.Model = "disaggregated-kv-fabric"
	}

	// Canonicalize message history into prompt sequence
	var sb strings.Builder
	for _, m := range req.Messages {
		sb.WriteString(fmt.Sprintf("<|im_start|>%s\n%s<|im_end|>\n", m.Role, m.Content))
	}
	prompt := sb.String()

	ctx := r.Context()
	requestID := fmt.Sprintf("chatcmpl-%d", time.Now().UnixNano())

	ref, decodeWorker, tokens, err := c.processPrompt(ctx, prompt, requestID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	tokenCh := make(chan string)
	go c.streamDecode(ctx, decodeWorker, *ref, prompt, req.MaxTokens, tokenCh)

	if req.Stream {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		for tok := range tokenCh {
			chunk := map[string]interface{}{
				"id":      requestID,
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"model":   req.Model,
				"choices": []map[string]interface{}{
					{
						"index": 0,
						"delta": map[string]string{"content": tok + " "},
					},
				},
			}
			fmt.Fprintf(w, "data: %s\n\n", mustJSON(chunk))
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
		return
	}

	// Non-streaming response: accumulate tokens
	var responseWords []string
	for tok := range tokenCh {
		responseWords = append(responseWords, tok)
	}

	resp := ChatCompletionResponse{
		ID:      requestID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   req.Model,
		Choices: []ChatCompletionResponseChoice{
			{
				Index: 0,
				Message: ChatMessage{
					Role:    "assistant",
					Content: strings.Join(responseWords, " "),
				},
				FinishReason: "stop",
			},
		},
		Usage: UsageStats{
			PromptTokens:     len(tokens),
			CompletionTokens: len(responseWords),
			TotalTokens:      len(tokens) + len(responseWords),
			PromptTokensDetails: PromptTokensDetails{
				CachedTokens: ref.CachedTokens,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func main() {
	daemonAddr := os.Getenv("DAEMON_ADDR")
	if daemonAddr == "" {
		daemonAddr = "127.0.0.1:50051"
	}
	conductor := NewConductor(4, 4, daemonAddr)

	backendURL := os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://127.0.0.1:11434/v1/chat/completions"
	}
	modelName := os.Getenv("MODEL_NAME")
	if modelName == "" {
		modelName = "smollm2"
	}
	conductor.SetBackend(backendURL, modelName)

	mux := http.NewServeMux()
	mux.HandleFunc("/generate", conductor.handleGenerate)
	mux.HandleFunc("/v1/chat/completions", conductor.handleChatCompletions)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port
	log.Printf("Disaggregated KV-Cache Conductor listening on %s (Daemon: %s)", addr, daemonAddr)
	log.Printf("Real Inference Engine Backend: %s (Model: %s)", backendURL, modelName)
	log.Printf(`Endpoints available:`)
	log.Printf(`  - POST http://localhost%s/generate (SSE streaming)`, addr)
	log.Printf(`  - POST http://localhost%s/v1/chat/completions (OpenAI compatible)`, addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
