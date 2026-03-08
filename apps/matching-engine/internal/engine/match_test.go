package engine

import (
	"testing"
	"github.com/khantopa/opentrade/matching-engine/internal/models"
)

// Write test case for match function when order type is market and no book exists for the ticker

func TestMatch_MarketOrderNoBook(t *testing.T) {
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

	trades, err := matcher.Match(order)

	if err == nil {
		t.Fatalf("Expected error for market order with no book, got nil")
	}

	expectedErrorMessage := "market buy order rejected: no asks available for ticker AAPL"
	if err.Error() != expectedErrorMessage {
		t.Errorf("Expected error message '%s', got '%s'", expectedErrorMessage, err.Error())
	}

	if len(trades) != 0 {
		t.Errorf("Expected no trades to be executed, got %d", len(trades))
	}
}
