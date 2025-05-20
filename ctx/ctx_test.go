package ctx

import (
	"bytes"
	"context"
	"testing"

	"github.com/go-gost/core/logger"
	"github.com/stretchr/testify/assert"
)

// mockLogger is a simple logger mock for testing.
type mockLogger struct {
	logger.Logger
}

func TestClientAddr(t *testing.T) {
	ctx := context.Background()

	// Test getting when not set
	addrNotSet := ClientAddrFromContext(ctx)
	assert.Equal(t, ClientAddr(""), addrNotSet, "Expected empty string for unset ClientAddr")

	// Test setting and getting
	expectedAddr := ClientAddr("127.0.0.1:12345")
	ctxWithAddr := ContextWithClientAddr(ctx, expectedAddr)

	assert.NotSame(t, ctx, ctxWithAddr, "ContextWithClientAddr should return a new context")

	actualAddr := ClientAddrFromContext(ctxWithAddr)
	assert.Equal(t, expectedAddr, actualAddr, "Retrieved ClientAddr does not match expected")

	// Test that original context is unaffected
	addrFromOriginalCtx := ClientAddrFromContext(ctx)
	assert.Equal(t, ClientAddr(""), addrFromOriginalCtx, "Original context should not be affected by WithValue")
}

func TestSid(t *testing.T) {
	ctx := context.Background()

	// Test getting when not set
	sidNotSet := SidFromContext(ctx)
	assert.Equal(t, Sid(""), sidNotSet, "Expected empty string for unset Sid")

	// Test setting and getting
	expectedSid := Sid("test-session-id-123")
	ctxWithSid := ContextWithSid(ctx, expectedSid)

	assert.NotSame(t, ctx, ctxWithSid, "ContextWithSid should return a new context")

	actualSid := SidFromContext(ctxWithSid)
	assert.Equal(t, expectedSid, actualSid, "Retrieved Sid does not match expected")

	// Test that original context is unaffected
	sidFromOriginalCtx := SidFromContext(ctx)
	assert.Equal(t, Sid(""), sidFromOriginalCtx, "Original context should not be affected")
}

func TestHash(t *testing.T) {
	ctx := context.Background()

	// Test getting when not set
	hashNotSet := HashFromContext(ctx)
	assert.Nil(t, hashNotSet, "Expected nil for unset Hash")

	// Test setting and getting
	expectedHash := &Hash{Source: "client_ip"}
	ctxWithHash := ContextWithHash(ctx, expectedHash)

	assert.NotSame(t, ctx, ctxWithHash, "ContextWithHash should return a new context")

	actualHash := HashFromContext(ctxWithHash)
	assert.Equal(t, expectedHash, actualHash, "Retrieved Hash does not match expected")
	assert.Same(t, expectedHash, actualHash, "Retrieved Hash pointer should be the same as set")


	// Test that original context is unaffected
	hashFromOriginalCtx := HashFromContext(ctx)
	assert.Nil(t, hashFromOriginalCtx, "Original context should not be affected")

	// Test with nil hash
	ctxWithNilHash := ContextWithHash(ctx, nil)
	nilHashRetrieved := HashFromContext(ctxWithNilHash)
	assert.Nil(t, nilHashRetrieved, "Expected nil when a nil Hash is set")
}

func TestClientID(t *testing.T) {
	ctx := context.Background()

	// Test getting when not set
	idNotSet := ClientIDFromContext(ctx)
	assert.Equal(t, ClientID(""), idNotSet, "Expected empty string for unset ClientID")

	// Test setting and getting
	expectedID := ClientID("client-abc-789")
	ctxWithID := ContextWithClientID(ctx, expectedID)

	assert.NotSame(t, ctx, ctxWithID, "ContextWithClientID should return a new context")

	actualID := ClientIDFromContext(ctxWithID)
	assert.Equal(t, expectedID, actualID, "Retrieved ClientID does not match expected")

	// Test that original context is unaffected
	idFromOriginalCtx := ClientIDFromContext(ctx)
	assert.Equal(t, ClientID(""), idFromOriginalCtx, "Original context should not be affected")
}

func TestBuffer(t *testing.T) {
	ctx := context.Background()

	// Test getting when not set
	bufferNotSet := BufferFromContext(ctx)
	assert.Nil(t, bufferNotSet, "Expected nil for unset Buffer")

	// Test setting and getting
	expectedBuffer := new(bytes.Buffer)
	expectedBuffer.WriteString("test data")
	ctxWithBuffer := ContextWithBuffer(ctx, expectedBuffer)

	assert.NotSame(t, ctx, ctxWithBuffer, "ContextWithBuffer should return a new context")

	actualBuffer := BufferFromContext(ctxWithBuffer)
	assert.Equal(t, expectedBuffer, actualBuffer, "Retrieved Buffer does not match expected")
	assert.Same(t, expectedBuffer, actualBuffer, "Retrieved Buffer pointer should be the same as set")
	if actualBuffer != nil {
		assert.Equal(t, "test data", actualBuffer.String(), "Buffer content mismatch")
	}

	// Test that original context is unaffected
	bufferFromOriginalCtx := BufferFromContext(ctx)
	assert.Nil(t, bufferFromOriginalCtx, "Original context should not be affected")

	// Test with nil buffer
	ctxWithNilBuffer := ContextWithBuffer(ctx, nil)
	nilBufferRetrieved := BufferFromContext(ctxWithNilBuffer)
	assert.Nil(t, nilBufferRetrieved, "Expected nil when a nil Buffer is set")
}

func TestLogger(t *testing.T) {
	ctx := context.Background()

	// Test getting when not set
	loggerNotSet := LoggerFromContext(ctx)
	assert.Nil(t, loggerNotSet, "Expected nil for unset Logger")

	// Test setting and getting
	expectedLogger := &mockLogger{} // Using a simple mock
	ctxWithLogger := ContextWithLogger(ctx, expectedLogger)

	assert.NotSame(t, ctx, ctxWithLogger, "ContextWithLogger should return a new context")

	actualLogger := LoggerFromContext(ctxWithLogger)
	assert.Equal(t, expectedLogger, actualLogger, "Retrieved Logger does not match expected")
	assert.Same(t, expectedLogger, actualLogger, "Retrieved Logger instance should be the same as set")

	// Test that original context is unaffected
	loggerFromOriginalCtx := LoggerFromContext(ctx)
	assert.Nil(t, loggerFromOriginalCtx, "Original context should not be affected")

	// Test with nil logger
	ctxWithNilLogger := ContextWithLogger(ctx, nil)
	nilLoggerRetrieved := LoggerFromContext(ctxWithNilLogger)
	assert.Nil(t, nilLoggerRetrieved, "Expected nil when a nil Logger is set")
}

func TestContextChaining(t *testing.T) {
	baseCtx := context.Background()

	addr := ClientAddr("192.168.1.1")
	sid := Sid("session-chain")
	clientID := ClientID("client-chain")

	// Chain the context modifications
	ctx1 := ContextWithClientAddr(baseCtx, addr)
	ctx2 := ContextWithSid(ctx1, sid)
	ctx3 := ContextWithClientID(ctx2, clientID)

	// Verify all values can be retrieved from the final context
	assert.Equal(t, addr, ClientAddrFromContext(ctx3), "ClientAddr not found in chained context")
	assert.Equal(t, sid, SidFromContext(ctx3), "Sid not found in chained context")
	assert.Equal(t, clientID, ClientIDFromContext(ctx3), "ClientID not found in chained context")

	// Verify intermediate contexts have appropriate values
	assert.Equal(t, addr, ClientAddrFromContext(ctx1), "ClientAddr not found in ctx1")
	assert.Equal(t, Sid(""), SidFromContext(ctx1), "Sid should not be in ctx1")

	assert.Equal(t, addr, ClientAddrFromContext(ctx2), "ClientAddr not found in ctx2")
	assert.Equal(t, sid, SidFromContext(ctx2), "Sid not found in ctx2")
	assert.Equal(t, ClientID(""), ClientIDFromContext(ctx2), "ClientID should not be in ctx2")

	// Verify original context is still empty
	assert.Equal(t, ClientAddr(""), ClientAddrFromContext(baseCtx), "ClientAddr should not be in baseCtx")
	assert.Equal(t, Sid(""), SidFromContext(baseCtx), "Sid should not be in baseCtx")
	assert.Equal(t, ClientID(""), ClientIDFromContext(baseCtx), "ClientID should not be in baseCtx")
}

func TestOverwritingValue(t *testing.T) {
	ctx := context.Background()

	initialAddr := ClientAddr("127.0.0.1")
	ctxWithAddr := ContextWithClientAddr(ctx, initialAddr)
	retrievedAddr1 := ClientAddrFromContext(ctxWithAddr)
	assert.Equal(t, initialAddr, retrievedAddr1, "Initial ClientAddr mismatch")

	// Overwrite with a new value
	newAddr := ClientAddr("192.168.0.100")
	ctxWithNewAddr := ContextWithClientAddr(ctxWithAddr, newAddr) // Using ctxWithAddr as parent

	assert.NotSame(t, ctxWithAddr, ctxWithNewAddr, "ContextWithClientAddr should return a new context when overwriting")

	retrievedNewAddr := ClientAddrFromContext(ctxWithNewAddr)
	assert.Equal(t, newAddr, retrievedNewAddr, "New ClientAddr mismatch after overwriting")

	// Ensure the "previous" context (ctxWithAddr) still holds the old value
	// This is standard context.WithValue behavior: it creates a new context,
	// the old one is immutable.
	retrievedOldAddr := ClientAddrFromContext(ctxWithAddr)
	assert.Equal(t, initialAddr, retrievedOldAddr, "ClientAddr in the 'previous' context was unexpectedly changed")
}
