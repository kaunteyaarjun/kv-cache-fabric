package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func parseSSEResponse(body string) (*KVCacheReference, []string) {
	var meta KVCacheReference
	var tokens []string

	scanner := bufio.NewScanner(strings.NewReader(body))
	var currentEvent string

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "event: ") {
			currentEvent = strings.TrimPrefix(line, "event: ")
		} else if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			switch currentEvent {
			case "meta":
				_ = json.Unmarshal([]byte(data), &meta)
			case "token":
				tokens = append(tokens, data)
			}
		}
	}
	return &meta, tokens
}

func TestMultiAgentPrefixSharingLifecycle(t *testing.T) {
	conductor := NewConductor(4, 4, "127.0.0.1:50051")
	handler := http.HandlerFunc(conductor.handleGenerate)

	// Base shared system prompt (simulating long agent rules)
	sharedPrefix := strings.Repeat("System: You are an autonomous software agent with deep reasoning. ", 40)

	promptPlanner := sharedPrefix + "Task: Plan database schema."
	promptCoder := sharedPrefix + "Task: Write Rust migrations."
	promptAuditor := sharedPrefix + "Task: Audit locks and memory leaks."
	promptReviewer := sharedPrefix + "Task: Plan database schema." // Exact duplicate of Planner

	// -----------------------------------------------------------------
	// 1. Agent 1 (Planner): Cold Request -> Cache Miss
	// -----------------------------------------------------------------
	req1Body, _ := json.Marshal(PromptRequest{Prompt: promptPlanner, MaxTokens: 5})
	req1 := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(req1Body))
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("planner request failed: %d", rec1.Code)
	}
	meta1, _ := parseSSEResponse(rec1.Body.String())
	if meta1.PrefillSkipped {
		t.Fatalf("expected Planner (cold) to NOT skip prefill")
	}
	if len(meta1.BlockIDs) == 0 {
		t.Fatalf("expected allocated blocks for Planner")
	}
	t.Logf("[Planner] Cold Run: Allocated %d blocks for %d tokens (Cached: %d)",
		len(meta1.BlockIDs), meta1.NumTokens, meta1.CachedTokens)

	// -----------------------------------------------------------------
	// 2. Agent 2 (Coder) & Agent 3 (Auditor): Partial Cache Hit
	// -----------------------------------------------------------------
	reqCoderBody, _ := json.Marshal(PromptRequest{Prompt: promptCoder, MaxTokens: 5})
	reqCoder := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(reqCoderBody))
	recCoder := httptest.NewRecorder()
	handler.ServeHTTP(recCoder, reqCoder)

	metaCoder, _ := parseSSEResponse(recCoder.Body.String())
	if metaCoder.PrefillSkipped {
		t.Fatalf("expected Coder to NOT skip prefill completely")
	}
	if metaCoder.CachedTokens == 0 {
		t.Fatalf("expected Coder to hit shared system prompt prefix cache, got 0 cached tokens")
	}
	t.Logf("[Coder] Partial Hit: Reused %d cached tokens, prefilled only %d suffix tokens",
		metaCoder.CachedTokens, metaCoder.PrefilledTokens)

	reqAuditorBody, _ := json.Marshal(PromptRequest{Prompt: promptAuditor, MaxTokens: 5})
	reqAuditor := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(reqAuditorBody))
	recAuditor := httptest.NewRecorder()
	handler.ServeHTTP(recAuditor, reqAuditor)

	metaAuditor, _ := parseSSEResponse(recAuditor.Body.String())
	if metaAuditor.CachedTokens == 0 {
		t.Fatalf("expected Auditor to hit prefix cache")
	}
	t.Logf("[Auditor] Partial Hit: Reused %d cached tokens, prefilled only %d suffix tokens",
		metaAuditor.CachedTokens, metaAuditor.PrefilledTokens)

	// -----------------------------------------------------------------
	// 3. Agent 4 (Reviewer): Exact Duplicate -> 100% Cache Hit
	// -----------------------------------------------------------------
	reqRevBody, _ := json.Marshal(PromptRequest{Prompt: promptReviewer, MaxTokens: 5})
	reqRev := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(reqRevBody))
	recRev := httptest.NewRecorder()
	handler.ServeHTTP(recRev, reqRev)

	metaRev, _ := parseSSEResponse(recRev.Body.String())
	if !metaRev.PrefillSkipped {
		t.Fatalf("expected Reviewer to completely SKIP prefill on exact duplicate")
	}
	if metaRev.CachedTokens != metaRev.NumTokens {
		t.Fatalf("expected 100%% cached tokens (%d), got %d", metaRev.NumTokens, metaRev.CachedTokens)
	}
	t.Logf("[Reviewer] 100%% Cache Hit: Reused all %d blocks. Prefill Skipped = true!",
		len(metaRev.BlockIDs))
}

func TestOpenAIChatCompletionsEndpoint(t *testing.T) {
	conductor := NewConductor(2, 2, "127.0.0.1:50051")
	handler := http.HandlerFunc(conductor.handleChatCompletions)

	chatReq := ChatCompletionRequest{
		Model: "gpt-4-turbo",
		Messages: []ChatMessage{
			{Role: "system", Content: "You are an autonomous systems agent."},
			{Role: "user", Content: "Explain memory-tiering for KV caches."},
		},
		MaxTokens: 5,
		Stream:    false,
	}

	body, _ := json.Marshal(chatReq)
	req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("chat completions failed with code %d: %s", rec.Code, rec.Body.String())
	}

	var resp ChatCompletionResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode chat completion response: %v", err)
	}

	if len(resp.Choices) == 0 || resp.Choices[0].Message.Content == "" {
		t.Fatalf("expected generated assistant message, got empty choices")
	}
	if resp.Usage.PromptTokens == 0 || resp.Usage.CompletionTokens == 0 {
		t.Fatalf("expected usage tokens recorded, got: %+v", resp.Usage)
	}

	t.Logf("OpenAI Response: %+v | Content: %q | Usage: %+v",
		resp.ID, resp.Choices[0].Message.Content, resp.Usage)
}

func TestEvictionThrashUnderHardwarePressure(t *testing.T) {
	// Configure conductor
	conductor := NewConductor(4, 4, "127.0.0.1:50051")
	handler := http.HandlerFunc(conductor.handleGenerate)

	// Fire 16 concurrent requests, each asking for 32 tokens (2 blocks).
	// Total requested = 16 * 2 = 32 blocks on an 8-block device tier.
	// This forces immediate LRU evictions to the host tier without deadlocks or OOM.
	numConcurrent := 16
	var wg sync.WaitGroup
	errCh := make(chan error, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()
			prompt := fmt.Sprintf("System: Agent %d task definition. %s User query %d.",
				agentID, strings.Repeat("Arbitrary unique context token padding ", 6), agentID)

			reqBody, _ := json.Marshal(PromptRequest{Prompt: prompt, MaxTokens: 4})
			req := httptest.NewRequest(http.MethodPost, "/generate", bytes.NewReader(reqBody))
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				errCh <- fmt.Errorf("agent %d failed with code %d: %s", agentID, rec.Code, rec.Body.String())
				return
			}

			meta, tokens := parseSSEResponse(rec.Body.String())
			if len(tokens) == 0 {
				errCh <- fmt.Errorf("agent %d generated 0 tokens", agentID)
				return
			}
			if len(meta.BlockIDs) == 0 {
				errCh <- fmt.Errorf("agent %d received 0 blocks", agentID)
				return
			}
		}(i)
	}

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Fatalf("concurrent thrash failure: %v", err)
	}

	devCount, hostCount := conductor.kvClient.Stats()
	t.Logf("Post-Thrash Memory Stats: Device Blocks = %d/8 | Host DRAM Blocks = %d/32",
		devCount, hostCount)

	if hostCount == 0 && devCount <= 8 {
		t.Logf("Notice: Hardware pressure successfully managed within available tiers")
	}
}
