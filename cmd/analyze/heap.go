package main

// entryHeap implements heap.Interface for a min-heap of dirEntry (sorted by Size)
// Since we want Top N Largest, we use a Min Heap of size N.
// When adding a new item:
// 1. If heap size < N: push
// 2. If heap size == N and item > min (root): pop min, push item
// The heap will thus maintain the largest N items.
type entryHeap []dirEntry

func (h entryHeap) Len() int           { return len(h) }
func (h entryHeap) Less(i, j int) bool { return h[i].Size < h[j].Size } // Min-heap based on Size
func (h entryHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *entryHeap) Push(x interface{}) {
	*h = append(*h, x.(dirEntry))
}

func (h *entryHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// largeFileHeap implements heap.Interface for fileEntry
type largeFileHeap []fileEntry

func (h largeFileHeap) Len() int           { return len(h) }
func (h largeFileHeap) Less(i, j int) bool { return h[i].Size < h[j].Size }
func (h largeFileHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *largeFileHeap) Push(x interface{}) {
	*h = append(*h, x.(fileEntry))
}

func (h *largeFileHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}
