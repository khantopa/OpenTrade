package engine

import (
	"testing"
	"sync"
	"github.com/khantopa/opentrade/matching-engine/internal/models"
)

// ---------------------------------------------------------------------------
// Market Buy Orders
// ---------------------------------------------------------------------------

func TestMatch_MarketBuyOrder_NoAsks_Rejected(t *testing.T) {
	matcher := NewMatcher()

	order := models.Order{
		ID:       "1",
		UserID:   "user1",
		Ticker:   "AAPL",
		Price:    150.00,
		Quantity: 10,
		Side:     models.OrderSideBuy,
		Type:     models.OrderTypeMarket,
		Status:   models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(order)

	if err == nil {
		t.Fatalf("Expected error for market buy with no asks, got nil")
	}

	if len(trades) != 0 {
		t.Errorf("Expected no trades, got %d", len(trades))
	}

	if updatedOrder.Status != models.OrderStatusRejected {
		t.Errorf("Expected status Rejected, got %s", updatedOrder.Status)
	}

	stored := matcher.OrderBooks["AAPL"].Orders["1"]
	if stored.Status != models.OrderStatusRejected {
		t.Errorf("Expected stored order status Rejected, got %s", stored.Status)
	}
}

func TestMatch_MarketBuyOrder_ExactFill(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "AAPL",
		Price: 150.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	marketBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "AAPL",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(marketBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Errorf("Expected trade quantity 10, got %d", trades[0].Quantity)
	}
	if trades[0].Price != 150.00 {
		t.Errorf("Expected trade price 150, got %f", trades[0].Price)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected status Filled, got %s", updatedOrder.Status)
	}
}

func TestMatch_MarketBuyOrder_PartialFill_Rejected(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "AAPL",
		Price: 150.00, Quantity: 5,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	marketBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "AAPL",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(marketBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 5 {
		t.Errorf("Expected trade quantity 5, got %d", trades[0].Quantity)
	}
	if updatedOrder.Quantity != 5 {
		t.Errorf("Expected remaining quantity 5, got %d", updatedOrder.Quantity)
	}
	if updatedOrder.Status != models.OrderStatusRejected {
		t.Errorf("Expected Rejected (unfilled market remainder), got %s", updatedOrder.Status)
	}
}

func TestMatch_MarketBuyOrder_MultipleAsks(t *testing.T) {
	matcher := NewMatcher()

	for _, ask := range []models.Order{
		{ID: "ask1", UserID: "s1", Ticker: "AAPL", Price: 100.00, Quantity: 3, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "ask2", UserID: "s2", Ticker: "AAPL", Price: 101.00, Quantity: 4, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "ask3", UserID: "s3", Ticker: "AAPL", Price: 102.00, Quantity: 5, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
	} {
		matcher.Match(ask)
	}

	marketBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "AAPL",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(marketBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 3 {
		t.Fatalf("Expected 3 trades, got %d", len(trades))
	}

	totalFilled := 0
	for _, tr := range trades {
		totalFilled += tr.Quantity
	}
	if totalFilled != 10 {
		t.Errorf("Expected total filled 10, got %d", totalFilled)
	}

	if trades[0].Price != 100.00 {
		t.Errorf("Expected best ask price 100 first, got %f", trades[0].Price)
	}

	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["AAPL"]
	storedAsk3 := books.Orders["ask3"]
	if storedAsk3.Quantity != 2 {
		t.Errorf("Expected ask3 remaining quantity 2, got %d", storedAsk3.Quantity)
	}
	if storedAsk3.Status != models.OrderStatusPartiallyFilled {
		t.Errorf("Expected ask3 PartiallyFilled, got %s", storedAsk3.Status)
	}
}

// ---------------------------------------------------------------------------
// Market Sell Orders
// ---------------------------------------------------------------------------

func TestMatch_MarketSellOrder_NoBids_Rejected(t *testing.T) {
	matcher := NewMatcher()

	order := models.Order{
		ID: "sell1", UserID: "user1", Ticker: "TSLA",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(order)

	if err == nil {
		t.Fatal("Expected error for market sell with no bids, got nil")
	}
	if len(trades) != 0 {
		t.Errorf("Expected no trades, got %d", len(trades))
	}
	if updatedOrder.Status != models.OrderStatusRejected {
		t.Errorf("Expected Rejected, got %s", updatedOrder.Status)
	}
}

func TestMatch_MarketSellOrder_ExactFill(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "TSLA",
		Price: 200.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	marketSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "TSLA",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(marketSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Errorf("Expected trade quantity 10, got %d", trades[0].Quantity)
	}
	if trades[0].Price != 200.00 {
		t.Errorf("Expected trade price 200, got %f", trades[0].Price)
	}
	if trades[0].BuyOrderID != "sell1" {
		t.Errorf("Expected BuyOrderID sell1, got %s", trades[0].BuyOrderID)
	}
	if trades[0].SellOrderID != "bid1" {
		t.Errorf("Expected SellOrderID bid1, got %s", trades[0].SellOrderID)
	}
	if trades[0].BuyerUserID != "seller1" {
		t.Errorf("Expected BuyerUserID seller1, got %s", trades[0].BuyerUserID)
	}
	if trades[0].SellerUserID != "buyer1" {
		t.Errorf("Expected SellerUserID buyer1, got %s", trades[0].SellerUserID)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}
}

func TestMatch_MarketSellOrder_PartialFill_Rejected(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "TSLA",
		Price: 200.00, Quantity: 3,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	marketSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "TSLA",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(marketSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 3 {
		t.Errorf("Expected trade quantity 3, got %d", trades[0].Quantity)
	}
	if updatedOrder.Quantity != 7 {
		t.Errorf("Expected remaining 7, got %d", updatedOrder.Quantity)
	}
	if updatedOrder.Status != models.OrderStatusRejected {
		t.Errorf("Expected Rejected, got %s", updatedOrder.Status)
	}
}

func TestMatch_MarketSellOrder_MultipleBids(t *testing.T) {
	matcher := NewMatcher()

	for _, bid := range []models.Order{
		{ID: "bid1", UserID: "b1", Ticker: "TSLA", Price: 202.00, Quantity: 4, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "bid2", UserID: "b2", Ticker: "TSLA", Price: 201.00, Quantity: 4, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "bid3", UserID: "b3", Ticker: "TSLA", Price: 200.00, Quantity: 4, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
	} {
		matcher.Match(bid)
	}

	marketSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "TSLA",
		Price: 0.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(marketSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 3 {
		t.Fatalf("Expected 3 trades, got %d", len(trades))
	}

	if trades[0].Price != 202.00 {
		t.Errorf("Expected best bid 202 first, got %f", trades[0].Price)
	}

	totalFilled := 0
	for _, tr := range trades {
		totalFilled += tr.Quantity
	}
	if totalFilled != 10 {
		t.Errorf("Expected total filled 10, got %d", totalFilled)
	}

	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	storedBid3 := matcher.OrderBooks["TSLA"].Orders["bid3"]
	if storedBid3.Quantity != 2 {
		t.Errorf("Expected bid3 remaining 2, got %d", storedBid3.Quantity)
	}
}

// ---------------------------------------------------------------------------
// Limit Buy Orders
// ---------------------------------------------------------------------------

func TestMatch_LimitBuyOrder_NoAsks_PushedToBids(t *testing.T) {
	matcher := NewMatcher()

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "GOOG",
		Price: 100.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("Expected 0 trades, got %d", len(trades))
	}
	if updatedOrder.Status != models.OrderStatusPending {
		t.Errorf("Expected Pending, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["GOOG"]
	if len(books.Bids) != 1 {
		t.Fatalf("Expected 1 bid in book, got %d", len(books.Bids))
	}
	if books.Bids[0].ID != "buy1" {
		t.Errorf("Expected bid ID buy1, got %s", books.Bids[0].ID)
	}
}

func TestMatch_LimitBuyOrder_PriceTooLow_NoMatch(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "GOOG",
		Price: 110.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "GOOG",
		Price: 100.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("Expected 0 trades, got %d", len(trades))
	}
	if updatedOrder.Status != models.OrderStatusPending {
		t.Errorf("Expected Pending, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["GOOG"]
	if len(books.Bids) != 1 {
		t.Errorf("Expected 1 bid in book, got %d", len(books.Bids))
	}
	if len(books.Asks) != 1 {
		t.Errorf("Expected 1 ask still in book, got %d", len(books.Asks))
	}
}

func TestMatch_LimitBuyOrder_ExactMatch(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "GOOG",
		Price: 100.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "GOOG",
		Price: 100.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Errorf("Expected quantity 10, got %d", trades[0].Quantity)
	}
	if trades[0].Price != 100.00 {
		t.Errorf("Expected price 100, got %f", trades[0].Price)
	}
	if trades[0].BuyOrderID != "ask1" {
		t.Errorf("Expected BuyOrderID ask1, got %s", trades[0].BuyOrderID)
	}
	if trades[0].SellOrderID != "buy1" {
		t.Errorf("Expected SellOrderID buy1, got %s", trades[0].SellOrderID)
	}
	if trades[0].BuyerUserID != "seller1" {
		t.Errorf("Expected BuyerUserID seller1, got %s", trades[0].BuyerUserID)
	}
	if trades[0].SellerUserID != "buyer1" {
		t.Errorf("Expected SellerUserID buyer1, got %s", trades[0].SellerUserID)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["GOOG"]
	if len(books.Asks) != 0 {
		t.Errorf("Expected asks heap empty, got %d", len(books.Asks))
	}
	if len(books.Bids) != 0 {
		t.Errorf("Expected bids heap empty, got %d", len(books.Bids))
	}
	storedAsk := books.Orders["ask1"]
	if storedAsk.Status != models.OrderStatusFilled {
		t.Errorf("Expected ask Filled, got %s", storedAsk.Status)
	}
}

func TestMatch_LimitBuyOrder_PartialFill_RemainderOnBook(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "GOOG",
		Price: 100.00, Quantity: 4,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "GOOG",
		Price: 100.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 4 {
		t.Errorf("Expected trade quantity 4, got %d", trades[0].Quantity)
	}
	if updatedOrder.Status != models.OrderStatusPartiallyFilled {
		t.Errorf("Expected PartiallyFilled, got %s", updatedOrder.Status)
	}
	if updatedOrder.Quantity != 6 {
		t.Errorf("Expected remaining 6, got %d", updatedOrder.Quantity)
	}

	books := matcher.OrderBooks["GOOG"]
	if len(books.Bids) != 1 {
		t.Fatalf("Expected remainder pushed to bids, got %d", len(books.Bids))
	}
	if books.Bids[0].Quantity != 6 {
		t.Errorf("Expected bid remainder 6, got %d", books.Bids[0].Quantity)
	}
}

func TestMatch_LimitBuyOrder_BuyPriceHigherThanAsk(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "GOOG",
		Price: 95.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "GOOG",
		Price: 100.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Price != 95.00 {
		t.Errorf("Expected trade at ask price 95, got %f", trades[0].Price)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}
}

func TestMatch_LimitBuyOrder_MultipleAskLevels(t *testing.T) {
	matcher := NewMatcher()

	for _, ask := range []models.Order{
		{ID: "ask1", UserID: "s1", Ticker: "GOOG", Price: 98.00, Quantity: 3, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "ask2", UserID: "s2", Ticker: "GOOG", Price: 99.00, Quantity: 3, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "ask3", UserID: "s3", Ticker: "GOOG", Price: 100.00, Quantity: 3, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "ask4", UserID: "s4", Ticker: "GOOG", Price: 105.00, Quantity: 3, Side: models.OrderSideSell, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
	} {
		matcher.Match(ask)
	}

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "GOOG",
		Price: 100.00, Quantity: 8,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 3 {
		t.Fatalf("Expected 3 trades (ask4 too expensive), got %d", len(trades))
	}

	if trades[0].Price != 98.00 {
		t.Errorf("Expected first trade at 98, got %f", trades[0].Price)
	}
	if trades[1].Price != 99.00 {
		t.Errorf("Expected second trade at 99, got %f", trades[1].Price)
	}
	if trades[2].Price != 100.00 {
		t.Errorf("Expected third trade at 100, got %f", trades[2].Price)
	}

	totalFilled := 0
	for _, tr := range trades {
		totalFilled += tr.Quantity
	}
	if totalFilled != 8 {
		t.Errorf("Expected total filled 8, got %d", totalFilled)
	}

	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["GOOG"]
	if len(books.Asks) != 2 {
		t.Errorf("Expected 2 asks remaining (ask3 partial + ask4 unmatched), got %d", len(books.Asks))
	}
}

// ---------------------------------------------------------------------------
// Limit Sell Orders
// ---------------------------------------------------------------------------

func TestMatch_LimitSellOrder_NoBids_PushedToAsks(t *testing.T) {
	matcher := NewMatcher()

	limitSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "MSFT",
		Price: 300.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("Expected 0 trades, got %d", len(trades))
	}
	if updatedOrder.Status != models.OrderStatusPending {
		t.Errorf("Expected Pending, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["MSFT"]
	if len(books.Asks) != 1 {
		t.Fatalf("Expected 1 ask in book, got %d", len(books.Asks))
	}
	if books.Asks[0].ID != "sell1" {
		t.Errorf("Expected ask ID sell1, got %s", books.Asks[0].ID)
	}
}

func TestMatch_LimitSellOrder_PriceTooHigh_NoMatch(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "MSFT",
		Price: 290.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	limitSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "MSFT",
		Price: 300.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 0 {
		t.Errorf("Expected 0 trades, got %d", len(trades))
	}
	if updatedOrder.Status != models.OrderStatusPending {
		t.Errorf("Expected Pending, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["MSFT"]
	if len(books.Asks) != 1 {
		t.Errorf("Expected 1 ask in book, got %d", len(books.Asks))
	}
	if len(books.Bids) != 1 {
		t.Errorf("Expected 1 bid still in book, got %d", len(books.Bids))
	}
}

func TestMatch_LimitSellOrder_ExactMatch(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "MSFT",
		Price: 300.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	limitSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "MSFT",
		Price: 300.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 10 {
		t.Errorf("Expected quantity 10, got %d", trades[0].Quantity)
	}
	if trades[0].Price != 300.00 {
		t.Errorf("Expected price 300, got %f", trades[0].Price)
	}
	if trades[0].BuyOrderID != "sell1" {
		t.Errorf("Expected BuyOrderID sell1, got %s", trades[0].BuyOrderID)
	}
	if trades[0].SellOrderID != "bid1" {
		t.Errorf("Expected SellOrderID bid1, got %s", trades[0].SellOrderID)
	}
	if trades[0].BuyerUserID != "seller1" {
		t.Errorf("Expected BuyerUserID seller1, got %s", trades[0].BuyerUserID)
	}
	if trades[0].SellerUserID != "buyer1" {
		t.Errorf("Expected SellerUserID buyer1, got %s", trades[0].SellerUserID)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["MSFT"]
	if len(books.Bids) != 0 {
		t.Errorf("Expected bids heap empty, got %d", len(books.Bids))
	}
	storedBid := books.Orders["bid1"]
	if storedBid.Status != models.OrderStatusFilled {
		t.Errorf("Expected bid Filled, got %s", storedBid.Status)
	}
}

func TestMatch_LimitSellOrder_PartialFill_RemainderOnBook(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "MSFT",
		Price: 300.00, Quantity: 4,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	limitSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "MSFT",
		Price: 300.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 4 {
		t.Errorf("Expected trade quantity 4, got %d", trades[0].Quantity)
	}
	if updatedOrder.Status != models.OrderStatusPartiallyFilled {
		t.Errorf("Expected PartiallyFilled, got %s", updatedOrder.Status)
	}
	if updatedOrder.Quantity != 6 {
		t.Errorf("Expected remaining 6, got %d", updatedOrder.Quantity)
	}

	books := matcher.OrderBooks["MSFT"]
	if len(books.Asks) != 1 {
		t.Fatalf("Expected remainder pushed to asks, got %d", len(books.Asks))
	}
	if books.Asks[0].Quantity != 6 {
		t.Errorf("Expected ask remainder 6, got %d", books.Asks[0].Quantity)
	}
}

func TestMatch_LimitSellOrder_SellPriceLowerThanBid(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "MSFT",
		Price: 310.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	limitSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "MSFT",
		Price: 300.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Price != 310.00 {
		t.Errorf("Expected trade at bid price 310, got %f", trades[0].Price)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}
}

func TestMatch_LimitSellOrder_MultipleBidLevels(t *testing.T) {
	matcher := NewMatcher()

	for _, bid := range []models.Order{
		{ID: "bid1", UserID: "b1", Ticker: "MSFT", Price: 305.00, Quantity: 3, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "bid2", UserID: "b2", Ticker: "MSFT", Price: 303.00, Quantity: 3, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "bid3", UserID: "b3", Ticker: "MSFT", Price: 301.00, Quantity: 3, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
		{ID: "bid4", UserID: "b4", Ticker: "MSFT", Price: 295.00, Quantity: 3, Side: models.OrderSideBuy, Type: models.OrderTypeLimit, Status: models.OrderStatusPending},
	} {
		matcher.Match(bid)
	}

	limitSell := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "MSFT",
		Price: 301.00, Quantity: 8,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 3 {
		t.Fatalf("Expected 3 trades (bid4 too low), got %d", len(trades))
	}

	if trades[0].Price != 305.00 {
		t.Errorf("Expected best bid 305 first, got %f", trades[0].Price)
	}
	if trades[1].Price != 303.00 {
		t.Errorf("Expected second trade at 303, got %f", trades[1].Price)
	}
	if trades[2].Price != 301.00 {
		t.Errorf("Expected third trade at 301, got %f", trades[2].Price)
	}

	totalFilled := 0
	for _, tr := range trades {
		totalFilled += tr.Quantity
	}
	if totalFilled != 8 {
		t.Errorf("Expected total filled 8, got %d", totalFilled)
	}

	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["MSFT"]
	if len(books.Bids) != 2 {
		t.Errorf("Expected 2 bids remaining (bid3 partial + bid4 unmatched), got %d", len(books.Bids))
	}
}

// ---------------------------------------------------------------------------
// Incoming order smaller than resting order (bestOrder partially consumed)
// ---------------------------------------------------------------------------

func TestMatch_IncomingOrderSmallerThanResting(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "NVDA",
		Price: 500.00, Quantity: 20,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	limitBuy := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "NVDA",
		Price: 500.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(limitBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 5 {
		t.Errorf("Expected trade quantity 5, got %d", trades[0].Quantity)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["NVDA"]
	storedAsk := books.Orders["ask1"]
	if storedAsk.Quantity != 15 {
		t.Errorf("Expected ask remaining 15, got %d", storedAsk.Quantity)
	}
	if storedAsk.Status != models.OrderStatusPartiallyFilled {
		t.Errorf("Expected ask PartiallyFilled, got %s", storedAsk.Status)
	}
	if len(books.Asks) != 1 {
		t.Fatalf("Expected ask still in heap, got %d", len(books.Asks))
	}
	if books.Asks[0].Quantity != 15 {
		t.Errorf("Expected ask heap quantity 15, got %d", books.Asks[0].Quantity)
	}
}

// ---------------------------------------------------------------------------
// Multiple Tickers
// ---------------------------------------------------------------------------

func TestMatch_MultipleTickers_Independent(t *testing.T) {
	matcher := NewMatcher()

	aaplAsk := models.Order{
		ID: "aapl-ask", UserID: "s1", Ticker: "AAPL",
		Price: 150.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(aaplAsk)

	tslaBid := models.Order{
		ID: "tsla-bid", UserID: "b1", Ticker: "TSLA",
		Price: 200.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(tslaBid)

	aaplBuy := models.Order{
		ID: "aapl-buy", UserID: "b2", Ticker: "AAPL",
		Price: 150.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	trades, _, err := matcher.Match(aaplBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade on AAPL, got %d", len(trades))
	}
	if trades[0].Ticker != "AAPL" {
		t.Errorf("Expected ticker AAPL, got %s", trades[0].Ticker)
	}

	tslaBooks := matcher.OrderBooks["TSLA"]
	if len(tslaBooks.Bids) != 1 {
		t.Errorf("TSLA bids should be untouched, got %d", len(tslaBooks.Bids))
	}
}

// ---------------------------------------------------------------------------
// Trade field verification
// ---------------------------------------------------------------------------

func TestMatch_TradeFieldsCorrect_BuySide(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "META",
		Price: 400.00, Quantity: 5,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	buyOrder := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "META",
		Price: 400.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, _, err := matcher.Match(buyOrder)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	if trade.ID == "" {
		t.Error("Expected trade ID to be non-empty")
	}
	if trade.Ticker != "META" {
		t.Errorf("Expected ticker META, got %s", trade.Ticker)
	}
	if trade.BuyOrderID != "ask1" {
		t.Errorf("Expected BuyOrderID ask1 (bestOrder.ID), got %s", trade.BuyOrderID)
	}
	if trade.SellOrderID != "buy1" {
		t.Errorf("Expected SellOrderID buy1 (order.ID), got %s", trade.SellOrderID)
	}
	if trade.BuyerUserID != "seller1" {
		t.Errorf("Expected BuyerUserID seller1 (bestOrder.UserID), got %s", trade.BuyerUserID)
	}
	if trade.SellerUserID != "buyer1" {
		t.Errorf("Expected SellerUserID buyer1 (order.UserID), got %s", trade.SellerUserID)
	}
	if trade.CreatedAt.IsZero() {
		t.Error("Expected CreatedAt to be non-zero")
	}
}

func TestMatch_TradeFieldsCorrect_SellSide(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "META",
		Price: 400.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	sellOrder := models.Order{
		ID: "sell1", UserID: "seller1", Ticker: "META",
		Price: 400.00, Quantity: 5,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, _, err := matcher.Match(sellOrder)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	if trade.BuyOrderID != "sell1" {
		t.Errorf("Expected BuyOrderID sell1 (swapped), got %s", trade.BuyOrderID)
	}
	if trade.SellOrderID != "bid1" {
		t.Errorf("Expected SellOrderID bid1 (swapped), got %s", trade.SellOrderID)
	}
	if trade.BuyerUserID != "seller1" {
		t.Errorf("Expected BuyerUserID seller1 (swapped), got %s", trade.BuyerUserID)
	}
	if trade.SellerUserID != "buyer1" {
		t.Errorf("Expected SellerUserID buyer1 (swapped), got %s", trade.SellerUserID)
	}
}

// ---------------------------------------------------------------------------
// Order book state verification
// ---------------------------------------------------------------------------

func TestMatch_OrderBookState_OrdersMapUpdated(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "seller1", Ticker: "AMZN",
		Price: 180.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	buyOrder := models.Order{
		ID: "buy1", UserID: "buyer1", Ticker: "AMZN",
		Price: 180.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(buyOrder)

	books := matcher.OrderBooks["AMZN"]

	storedAsk := books.Orders["ask1"]
	if storedAsk.Status != models.OrderStatusFilled {
		t.Errorf("Expected ask1 Filled in orders map, got %s", storedAsk.Status)
	}
	if storedAsk.Quantity != 0 {
		t.Errorf("Expected ask1 quantity 0, got %d", storedAsk.Quantity)
	}

	storedBuy := books.Orders["buy1"]
	if storedBuy.Status != models.OrderStatusFilled {
		t.Errorf("Expected buy1 Filled in orders map, got %s", storedBuy.Status)
	}
	if storedBuy.Quantity != 0 {
		t.Errorf("Expected buy1 quantity 0, got %d", storedBuy.Quantity)
	}
}

// ---------------------------------------------------------------------------
// Existing book reuse (second order for same ticker uses existing book)
// ---------------------------------------------------------------------------

func TestMatch_ExistingBookReused(t *testing.T) {
	matcher := NewMatcher()

	order1 := models.Order{
		ID: "o1", UserID: "u1", Ticker: "AAPL",
		Price: 150.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(order1)

	order2 := models.Order{
		ID: "o2", UserID: "u2", Ticker: "AAPL",
		Price: 155.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(order2)

	books := matcher.OrderBooks["AAPL"]
	if len(books.Bids) != 2 {
		t.Errorf("Expected 2 bids in existing book, got %d", len(books.Bids))
	}
	if len(books.Orders) != 2 {
		t.Errorf("Expected 2 orders in map, got %d", len(books.Orders))
	}
}

// ---------------------------------------------------------------------------
// NewMatcher initialisation
// ---------------------------------------------------------------------------

func TestNewMatcher(t *testing.T) {
	matcher := NewMatcher()

	if matcher == nil {
		t.Fatal("Expected non-nil matcher")
	}
	if matcher.OrderBooks == nil {
		t.Fatal("Expected non-nil OrderBooks map")
	}
	if len(matcher.OrderBooks) != 0 {
		t.Errorf("Expected empty OrderBooks, got %d", len(matcher.OrderBooks))
	}
}

// ---------------------------------------------------------------------------
// Market order on existing book (book exists but opposite side empty)
// ---------------------------------------------------------------------------

func TestMatch_MarketBuyOrder_BookExistsButNoAsks(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "b1", Ticker: "AAPL",
		Price: 150.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	marketBuy := models.Order{
		ID: "mbuy1", UserID: "b2", Ticker: "AAPL",
		Price: 0.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	_, updatedOrder, err := matcher.Match(marketBuy)

	if err == nil {
		t.Fatal("Expected error for market buy with no asks")
	}
	if updatedOrder.Status != models.OrderStatusRejected {
		t.Errorf("Expected Rejected, got %s", updatedOrder.Status)
	}
}

func TestMatch_MarketSellOrder_BookExistsButNoBids(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "s1", Ticker: "AAPL",
		Price: 150.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	marketSell := models.Order{
		ID: "msell1", UserID: "s2", Ticker: "AAPL",
		Price: 0.00, Quantity: 5,
		Side: models.OrderSideSell, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	_, updatedOrder, err := matcher.Match(marketSell)

	if err == nil {
		t.Fatal("Expected error for market sell with no bids")
	}
	if updatedOrder.Status != models.OrderStatusRejected {
		t.Errorf("Expected Rejected, got %s", updatedOrder.Status)
	}
}

// ---------------------------------------------------------------------------
// Quantity-1 edge cases
// ---------------------------------------------------------------------------

func TestMatch_SingleQuantityOrders(t *testing.T) {
	matcher := NewMatcher()

	askOrder := models.Order{
		ID: "ask1", UserID: "s1", Ticker: "X",
		Price: 10.00, Quantity: 1,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(askOrder)

	buyOrder := models.Order{
		ID: "buy1", UserID: "b1", Ticker: "X",
		Price: 10.00, Quantity: 1,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(buyOrder)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}
	if trades[0].Quantity != 1 {
		t.Errorf("Expected trade quantity 1, got %d", trades[0].Quantity)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}
}

// ---------------------------------------------------------------------------
// Chained fills: multiple small orders in heap consumed by one large order
// ---------------------------------------------------------------------------

func TestMatch_LargeBuyConsumesMultipleSmallAsks(t *testing.T) {
	matcher := NewMatcher()

	for i := 0; i < 5; i++ {
		ask := models.Order{
			ID: "ask" + string(rune('A'+i)), UserID: "s" + string(rune('1'+i)), Ticker: "CHAIN",
			Price: 50.00, Quantity: 2,
			Side: models.OrderSideSell, Type: models.OrderTypeLimit,
			Status: models.OrderStatusPending,
		}
		matcher.Match(ask)
	}

	bigBuy := models.Order{
		ID: "bigbuy", UserID: "buyer", Ticker: "CHAIN",
		Price: 50.00, Quantity: 10,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(bigBuy)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 5 {
		t.Fatalf("Expected 5 trades, got %d", len(trades))
	}

	totalFilled := 0
	for _, tr := range trades {
		totalFilled += tr.Quantity
		if tr.Quantity != 2 {
			t.Errorf("Expected each trade quantity 2, got %d", tr.Quantity)
		}
	}
	if totalFilled != 10 {
		t.Errorf("Expected total 10, got %d", totalFilled)
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["CHAIN"]
	if len(books.Asks) != 0 {
		t.Errorf("Expected asks heap empty, got %d", len(books.Asks))
	}
}

// ---------------------------------------------------------------------------
// Sell-side: large sell consumes multiple small bids
// ---------------------------------------------------------------------------

func TestMatch_LargeSellConsumesMultipleSmallBids(t *testing.T) {
	matcher := NewMatcher()

	for i := 0; i < 5; i++ {
		bid := models.Order{
			ID: "bid" + string(rune('A'+i)), UserID: "b" + string(rune('1'+i)), Ticker: "CHAIN",
			Price: 50.00, Quantity: 2,
			Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
			Status: models.OrderStatusPending,
		}
		matcher.Match(bid)
	}

	bigSell := models.Order{
		ID: "bigsell", UserID: "seller", Ticker: "CHAIN",
		Price: 50.00, Quantity: 10,
		Side: models.OrderSideSell, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}

	trades, updatedOrder, err := matcher.Match(bigSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 5 {
		t.Fatalf("Expected 5 trades, got %d", len(trades))
	}
	if updatedOrder.Status != models.OrderStatusFilled {
		t.Errorf("Expected Filled, got %s", updatedOrder.Status)
	}

	books := matcher.OrderBooks["CHAIN"]
	if len(books.Bids) != 0 {
		t.Errorf("Expected bids heap empty, got %d", len(books.Bids))
	}
}

// ---------------------------------------------------------------------------
// Market sell matched against multiple bids (verify sell-side swap per trade)
// ---------------------------------------------------------------------------

func TestMatch_MarketSell_TradeFieldSwap(t *testing.T) {
	matcher := NewMatcher()

	bidOrder := models.Order{
		ID: "bid1", UserID: "buyer1", Ticker: "SWAP",
		Price: 50.00, Quantity: 5,
		Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
		Status: models.OrderStatusPending,
	}
	matcher.Match(bidOrder)

	marketSell := models.Order{
		ID: "msell1", UserID: "seller1", Ticker: "SWAP",
		Price: 0.00, Quantity: 5,
		Side: models.OrderSideSell, Type: models.OrderTypeMarket,
		Status: models.OrderStatusPending,
	}

	trades, _, err := matcher.Match(marketSell)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if len(trades) != 1 {
		t.Fatalf("Expected 1 trade, got %d", len(trades))
	}

	trade := trades[0]
	if trade.BuyOrderID != "msell1" {
		t.Errorf("Expected BuyOrderID msell1 (sell-side swap), got %s", trade.BuyOrderID)
	}
	if trade.SellOrderID != "bid1" {
		t.Errorf("Expected SellOrderID bid1 (sell-side swap), got %s", trade.SellOrderID)
	}
	if trade.BuyerUserID != "seller1" {
		t.Errorf("Expected BuyerUserID seller1 (sell-side swap), got %s", trade.BuyerUserID)
	}
	if trade.SellerUserID != "buyer1" {
		t.Errorf("Expected SellerUserID buyer1 (sell-side swap), got %s", trade.SellerUserID)
	}
}


func TestMatch_ConcurrentOrders_RaceCondition(t *testing.T) {
    matcher := NewMatcher()

    // Seed one ask with 1 share
    ask := models.Order{
        ID: "ask1", UserID: "seller1", Ticker: "AAPL",
        Price: 150.00, Quantity: 1,
        Side: models.OrderSideSell, Type: models.OrderTypeLimit,
        Status: models.OrderStatusPending,
    }
    matcher.Match(ask)

    // Two goroutines both try to buy that 1 share simultaneously
    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        matcher.Match(models.Order{
            ID: "buy1", UserID: "b1", Ticker: "AAPL",
            Price: 150.00, Quantity: 1,
            Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
            Status: models.OrderStatusPending,
        })
    }()

    go func() {
        defer wg.Done()
        matcher.Match(models.Order{
            ID: "buy2", UserID: "b2", Ticker: "AAPL",
            Price: 150.00, Quantity: 1,
            Side: models.OrderSideBuy, Type: models.OrderTypeLimit,
            Status: models.OrderStatusPending,
        })
    }()

    wg.Wait()
}