package kvclient

import (
	"context"
	"testing"
)

func TestFormatAndParseBlockID(t *testing.T) {
	testCases := []struct {
		id       uint64
		expected string
	}{
		{1, "blk-1"},
		{42, "blk-42"},
		{1000001, "blk-1000001"},
		{18446744073709551615, "blk-18446744073709551615"},
	}

	for _, tc := range testCases {
		formatted := FormatBlockID(tc.id)
		if formatted != tc.expected {
			t.Errorf("FormatBlockID(%d) = %s; want %s", tc.id, formatted, tc.expected)
		}

		parsed, err := ParseBlockID(formatted)
		if err != nil {
			t.Fatalf("ParseBlockID(%s) failed: %v", formatted, err)
		}
		if parsed != tc.id {
			t.Errorf("ParseBlockID(%s) = %d; want %d", formatted, parsed, tc.id)
		}
	}

	// Test namespaced and fallback formats
	namespaced := "local-blk-999"
	parsed, err := ParseBlockID(namespaced)
	if err != nil || parsed != 999 {
		t.Errorf("ParseBlockID(%s) = %d, %v; want 999, nil", namespaced, parsed, err)
	}

	rawNumeric := "12345"
	parsed, err = ParseBlockID(rawNumeric)
	if err != nil || parsed != 12345 {
		t.Errorf("ParseBlockID(%s) = %d, %v; want 12345, nil", rawNumeric, parsed, err)
	}

	// Invalid strings
	if _, err := ParseBlockID("not-a-block"); err == nil {
		t.Errorf("ParseBlockID('not-a-block') should have failed")
	}
}

func TestLocalFallbackResilienceAndOffsetNamespace(t *testing.T) {
	// Attempt connection to non-existent gRPC daemon port
	client, err := NewClient("127.0.0.1:59999")
	if err != nil {
		t.Fatalf("NewClient should not error on connection failure, should return fallback: %v", err)
	}
	defer client.Close()

	if !client.isFallback() {
		t.Fatalf("client should be in fallback mode when gRPC is unreachable")
	}

	ctx := context.Background()

	// Allocate a block and verify offset ID starts above 1,000,000
	payload := []byte("TENSOR_DATA_TEST_12345")
	blockID, tier, err := client.AllocateBlock(ctx, 1, 16, payload)
	if err != nil {
		t.Fatalf("AllocateBlock failed on fallback client: %v", err)
	}
	if blockID <= 1_000_000 {
		t.Errorf("expected blockID to be offset >= 1000000, got %d", blockID)
	}
	if tier != string(TierDevice) {
		t.Errorf("expected tier %s, got %s", TierDevice, tier)
	}

	// Test writing and reading data
	err = client.WriteBlockData(ctx, blockID, 0, []byte("MODIFIED_DATA"))
	if err != nil {
		t.Fatalf("WriteBlockData failed: %v", err)
	}

	readData, readTier, err := client.ReadBlockData(ctx, blockID, 0, uint32(len("MODIFIED_DATA")))
	if err != nil {
		t.Fatalf("ReadBlockData failed: %v", err)
	}
	if string(readData[:len("MODIFIED_DATA")]) != "MODIFIED_DATA" {
		t.Errorf("ReadBlockData returned %s; want MODIFIED_DATA", string(readData))
	}
	if readTier != string(TierDevice) {
		t.Errorf("readTier = %s; want %s", readTier, TierDevice)
	}

	// Touch block
	if err := client.TouchBlock(ctx, blockID); err != nil {
		t.Fatalf("TouchBlock failed: %v", err)
	}
}
