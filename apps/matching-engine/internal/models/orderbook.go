package models


type OrderBook struct {
	Ticker string
	Bids BidHeap
	Asks AskHeap
	Orders map[string]Order
}

type BidHeap []Order

type AskHeap []Order 


// BidHeap methods

func (h BidHeap) Less(i, j int) bool {
	return h[i].Price > h[j].Price
}

func (h BidHeap) Len() int {
	return len(h)
}

func (h BidHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *BidHeap) Push(x interface{}) {
	*h = append(*h, x.(Order))
}

func (h *BidHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// AskHeap methods

func (h AskHeap) Len() int {
	return len(h)
}

func (h AskHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h AskHeap) Less(i, j int) bool {
	return h[i].Price < h[j].Price
}

func (h *AskHeap) Push(x interface{}) {
	*h = append(*h, x.(Order))
}

func (h *AskHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}