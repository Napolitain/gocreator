package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCacheService_SetAndGet(t *testing.T) {
	cache := NewCacheService(5*time.Minute, 10*time.Minute)

	// Test setting and getting a value
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)
}

func TestCacheService_GetNonExistent(t *testing.T) {
	cache := NewCacheService(5*time.Minute, 10*time.Minute)

	// Test getting a non-existent key
	_, found := cache.Get("nonexistent")
	assert.False(t, found)
}

func TestCacheService_Delete(t *testing.T) {
	cache := NewCacheService(5*time.Minute, 10*time.Minute)

	// Set a value
	cache.Set("key1", "value1")
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	// Delete the value
	cache.Delete("key1")
	_, found = cache.Get("key1")
	assert.False(t, found)
}

func TestCacheService_Clear(t *testing.T) {
	cache := NewCacheService(5*time.Minute, 10*time.Minute)

	// Set multiple values
	cache.Set("key1", "value1")
	cache.Set("key2", "value2")
	cache.Set("key3", "value3")

	// Clear all values
	cache.Clear()

	// Verify all values are gone
	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	_, found3 := cache.Get("key3")

	assert.False(t, found1)
	assert.False(t, found2)
	assert.False(t, found3)
}

func TestCacheService_DifferentTypes(t *testing.T) {
	cache := NewCacheService(5*time.Minute, 10*time.Minute)

	// Test with different types
	cache.Set("string", "value")
	cache.Set("int", 42)
	cache.Set("bool", true)
	cache.Set("slice", []string{"a", "b", "c"})

	strVal, found := cache.Get("string")
	assert.True(t, found)
	assert.Equal(t, "value", strVal)

	intVal, found := cache.Get("int")
	assert.True(t, found)
	assert.Equal(t, 42, intVal)

	boolVal, found := cache.Get("bool")
	assert.True(t, found)
	assert.Equal(t, true, boolVal)

	sliceVal, found := cache.Get("slice")
	assert.True(t, found)
	assert.Equal(t, []string{"a", "b", "c"}, sliceVal)
}

func TestCacheService_Expiration(t *testing.T) {
	// Create cache with very short expiration for testing
	cache := NewCacheService(100*time.Millisecond, 50*time.Millisecond)

	// Set a value
	cache.Set("key1", "value1")

	// Value should exist immediately
	value, found := cache.Get("key1")
	assert.True(t, found)
	assert.Equal(t, "value1", value)

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Value should be expired
	_, found = cache.Get("key1")
	assert.False(t, found)
}
