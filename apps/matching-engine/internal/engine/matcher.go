package engine
import (
	"fmt"
	"time"
	"github.com/khantopa/opentrade/matching-engine/internal/models"
)

type Matcher struct {
	OrderBooks map[string]*models.OrderBook
}

type Trade struct {
	ID string
	Ticker string
	BuyOrderID string
	SellOrderID string
	AvgPrice float64
	Quantity int
	SellerUserID string
	BuyerUserID string
	CreatedAt time.Time
}

func (m *Matcher) Match(order models.Order) ([]Trade, error) {
	books, ok := m.OrderBooks[order.Ticker]

	if !ok {
		m.OrderBooks[order.Ticker] = &models.OrderBook{
			Ticker: order.Ticker,
			Bids: models.BidHeap{},
			Asks: models.AskHeap{},
			Orders: make(map[string]models.Order),
		}

		books = m.OrderBooks[order.Ticker]
	}

	books.Orders[order.ID] = order

	// Market Order Rejection
	if order.Type == models.OrderTypeMarket {
		if order.Side == models.OrderSideBuy && len(books.Asks) == 0 {
			order.Status = models.OrderStatusRejected
			books.Orders[order.ID] = order
			return nil, fmt.Errorf("market buy order rejected: no asks available for ticker %s", order.Ticker)
		}

		if order.Side == models.OrderSideSell && len(books.Bids) == 0 {
			order.Status = models.OrderStatusRejected
			books.Orders[order.ID] = order
			return nil, fmt.Errorf("market sell order rejected: no bids available for ticker %s", order.Ticker)
		}
	}

	return []Trade{}, nil

}