package cache

import (
	"slices"
	"testing"
)

func constant(size int, value byte) []byte {
	b := make([]byte, size)
	for i := range b {
		b[i] = value
	}
	return b
}

func equalData(left, right CachedData) bool {
	return left.ContentType == right.ContentType && slices.Equal(left.Data, right.Data)
}

func getOrFail(t *testing.T, cache *Cache, path string) CachedData {
	t.Helper()
	data, ok := cache.Get(path)
	if !ok {
		t.Fatalf("data not present for path: %s", path)
		return CachedData{}
	}
	return data
}

func has(t *testing.T, cache *Cache, path string) bool {
	t.Helper()
	_, ok := cache.Get(path)
	return ok
}

func TestCache(t *testing.T) {
	maxSize := 2048
	cache := New(maxSize)

	data1 := CachedData{Data: constant(1024, 1), ContentType: "text/plain"}
	data2 := CachedData{Data: constant(1024, 2), ContentType: "text/plain"}
	data3 := CachedData{Data: constant(1024, 3), ContentType: "text/plain"}

	// cache should be empty
	if cache.size != 0 || len(cache.data) != 0 || cache.Len() != 0 {
		t.Fatalf("cache should be empty!")
	}

	// add two items, they should fit
	cache.Put("data1", data1)
	cache.Put("data2", data2)
	if cache.size != data1.Len()+data2.Len() ||
		!equalData(getOrFail(t, cache, "data1"), data1) ||
		!equalData(getOrFail(t, cache, "data2"), data2) {
		t.Fatalf("cache should contain added elements")
	}

	// ping both of them a couple of times
	getOrFail(t, cache, "data1") // 2
	getOrFail(t, cache, "data1") // 3
	getOrFail(t, cache, "data1") // 4
	getOrFail(t, cache, "data2") // 1
	getOrFail(t, cache, "data2") // 2

	// add a new element, should evict the previous with lower count
	cache.Put("data3", data3)
	if cache.size != data1.Len()+data3.Len() ||
		!equalData(getOrFail(t, cache, "data1"), data1) ||
		!equalData(getOrFail(t, cache, "data3"), data3) ||
		has(t, cache, "data2") {
		t.Fatalf("cache should have evicted data2")
	}

	// add the evicted element, should evict the newer element
	cache.Put("data2", data2)
	if cache.size != data1.Len()+data2.Len() ||
		!equalData(getOrFail(t, cache, "data1"), data1) ||
		!equalData(getOrFail(t, cache, "data2"), data2) ||
		has(t, cache, "data3") {
		t.Fatalf("cache should have evicted data3")
	}

	// ping the lower count element until it's higher
	getOrFail(t, cache, "data2") // 2
	getOrFail(t, cache, "data2") // 3
	getOrFail(t, cache, "data2") // 4
	getOrFail(t, cache, "data2") // 5
	getOrFail(t, cache, "data2") // 6
	getOrFail(t, cache, "data2") // 7

	// add again, should evict first element
	cache.Put("data3", data3)
	if cache.size != data2.Len()+data3.Len() ||
		!equalData(getOrFail(t, cache, "data2"), data2) ||
		!equalData(getOrFail(t, cache, "data3"), data3) ||
		has(t, cache, "data1") {
		t.Fatalf("cache should have evicted data1")
	}
}
