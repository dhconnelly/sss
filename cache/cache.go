package cache

import (
	"container/heap"
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
	data    map[string]CachedData
	lfu     *frequencyHeap
	size    int
	maxSize int
}

func (c *Cache) Size() int {
	return c.size
}

func (c *Cache) Len() int {
	return len(c.data)
}

func New(size int) *Cache {
	return &Cache{
		data:    make(map[string]CachedData),
		lfu:     &frequencyHeap{},
		size:    0,
		maxSize: size,
	}
}

func (c *Cache) Get(path string) (CachedData, bool) {
	data, ok := c.data[path]
	if !ok {
		return CachedData{}, false
	}
	c.lfu.incr(path)
	return data, true
}

func (c *Cache) pop() {
	last := heap.Pop(c.lfu).(elem)
	delete(c.data, last.path)
	c.size -= last.size
}

func (c *Cache) push(path string, data CachedData) {
	last := elem{path: path, size: data.Len(), count: 0}
	heap.Push(c.lfu, last)
	c.data[path] = data
	c.size += last.size
}

func (c *Cache) Put(path string, data CachedData) {
	if len(data.Data) > c.maxSize {
		return
	}
	for c.size+len(data.Data) > c.maxSize {
		c.pop()
	}
	c.push(path, data)
}
