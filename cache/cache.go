package cache

import (
	"container/heap"
	"expvar"
	"log"
	"sync"
)

var (
	cacheCount   = expvar.NewInt("cacheSizeItems")
	cacheSizeCur = expvar.NewInt("cacheSizeBytesCur")
	cacheSizeMax = expvar.NewInt("cacheSizeBytesMax")
)

type elem struct {
	path  string
	size  int
	count int
}

type frequencyHeap struct {
	elems []elem
}

func (h *frequencyHeap) Len() int {
	return len(h.elems)
}

func (h *frequencyHeap) Less(i, j int) bool {
	return h.elems[i].count < h.elems[j].count
}

func (h *frequencyHeap) Swap(i, j int) {
	h.elems[i], h.elems[j] = h.elems[j], h.elems[i]
}

func (h *frequencyHeap) Pop() any {
	last := h.elems[len(h.elems)-1]
	h.elems = h.elems[0 : len(h.elems)-1]
	return last
}

func (h *frequencyHeap) Push(x any) {
	last := x.(elem)
	h.elems = append(h.elems, last)
}

func (h *frequencyHeap) incr(path string) {
	for i := range h.elems {
		if h.elems[i].path == path {
			h.elems[i].count++
			heap.Fix(h, i)
			break
		}
	}
}

type CachedData struct {
	ContentType string
	Data        []byte
}

func (data CachedData) Len() int {
	return len(data.Data)
}

type Cache struct {
	mux     sync.Mutex
	data    map[string]CachedData
	lfu     *frequencyHeap
	size    int
	maxSize int
}

func New(size int) *Cache {
	cacheSizeMax.Set(int64(size))
	return &Cache{
		mux:     sync.Mutex{},
		data:    make(map[string]CachedData),
		lfu:     &frequencyHeap{},
		size:    0,
		maxSize: size,
	}
}

func (c *Cache) pop() {
	// UNSYNCHRONIZED
	last := heap.Pop(c.lfu).(elem)
	delete(c.data, last.path)
	c.size -= last.size
	log.Printf("cache: evicted %v, new count %d", last, len(c.data))
	cacheCount.Set(int64(len(c.data)))
	cacheSizeCur.Set(int64(c.size))
}

func (c *Cache) push(path string, data CachedData) {
	// UNSYNCHRONIZED
	last := elem{path: path, size: data.Len(), count: 0}
	heap.Push(c.lfu, last)
	c.data[path] = data
	c.size += last.size
	log.Printf("cache: added %v, new count %d", last, len(c.data))
	cacheCount.Set(int64(len(c.data)))
	cacheSizeCur.Set(int64(c.size))
}

func (c *Cache) Size() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return c.size
}

func (c *Cache) Len() int {
	c.mux.Lock()
	defer c.mux.Unlock()
	return len(c.data)
}

func (c *Cache) Get(path string) (CachedData, bool) {
	c.mux.Lock()
	defer c.mux.Unlock()
	data, ok := c.data[path]
	if !ok {
		return CachedData{}, false
	}
	c.lfu.incr(path)
	return data, true
}

func (c *Cache) Put(path string, data CachedData) {
	c.mux.Lock()
	defer c.mux.Unlock()
	if len(data.Data) > c.maxSize {
		return
	}
	for c.size+len(data.Data) > c.maxSize {
		c.pop()
	}
	c.push(path, data)
}
