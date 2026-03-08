package engine
import (
	"errors"
	"time"
	"github.com/khantopa/opentrade/matching-engine/internal/models"
	"container/heap"
	"github.com/google/uuid"
)

type Matcher struct {
	OrderBooks map[string]*models.OrderBook
}


type OrderHeap interface {
	heap.Interface
	Peek() models.Order
}

type Trade struct {
	ID string
	Ticker string
	BuyOrderID string
	SellOrderID string
	Price float64
	Quantity int
	SellerUserID string
	BuyerUserID string
	CreatedAt time.Time
}

type PriceCheck func(orderPrice float64, heapPrice float64) bool

func NewMatcher() *Matcher {
    return &Matcher{
        OrderBooks: make(map[string]*models.OrderBook),
    }
}

func (m *Matcher) Match(order models.Order) ([]Trade, models.Order, error) {
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

		errorMessage := ""

		if order.Side == models.OrderSideBuy && len(books.Asks) == 0 {
			order.Status = models.OrderStatusRejected
			errorMessage = "market buy order rejected: no asks available for ticker " + order.Ticker
		}  
		
		if order.Side == models.OrderSideSell && len(books.Bids) == 0 {
			order.Status = models.OrderStatusRejected
			errorMessage = "market sell order rejected: no bids available for ticker " + order.Ticker
		}

		if errorMessage != "" {
		  books.Orders[order.ID] = order
			return nil, order, errors.New(errorMessage)
		}
	}

	var trades []Trade
	var updatedOrder models.Order

	if order.Side == models.OrderSideBuy {
		trades, updatedOrder = m.matchOrder(order, books, &books.Asks, func(orderPrice, heapPrice float64) bool {
			return order.Type == models.OrderTypeMarket || orderPrice >= heapPrice
		})
		if(updatedOrder.Quantity > 0 && order.Type != models.OrderTypeMarket) {
			heap.Push(&books.Bids, updatedOrder)
		} else if (updatedOrder.Quantity > 0 && order.Type == models.OrderTypeMarket) {
			updatedOrder.Status = models.OrderStatusRejected
			books.Orders[updatedOrder.ID] = updatedOrder
		}
	} else {
		trades, updatedOrder = m.matchOrder(order, books, &books.Bids, func(orderPrice, heapPrice float64) bool {
			return order.Type == models.OrderTypeMarket || orderPrice <= heapPrice
		})
		if(updatedOrder.Quantity > 0 && order.Type != models.OrderTypeMarket) {
			heap.Push(&books.Asks, updatedOrder)
		} else if (updatedOrder.Quantity > 0 && order.Type == models.OrderTypeMarket) {
			updatedOrder.Status = models.OrderStatusRejected
			books.Orders[updatedOrder.ID] = updatedOrder
		}
	}
	

	return trades, updatedOrder, nil


}


func (m *Matcher) matchOrder(
	order models.Order,
	books *models.OrderBook,
	oppositeHeap OrderHeap,
	priceCheck PriceCheck,
) ([]Trade, models.Order) {
	trades := []Trade{}
	
	for order.Quantity > 0 && oppositeHeap.Len() > 0 {

		bestOrder := oppositeHeap.Peek()
		
		if !priceCheck(order.Price, bestOrder.Price) {
			break
		}

		fillQuantity := min(order.Quantity, bestOrder.Quantity)

		order.Quantity -= fillQuantity
		bestOrder.Quantity -= fillQuantity

		if bestOrder.Quantity == 0 {
			bestOrder.Status = models.OrderStatusFilled
			heap.Pop(oppositeHeap)
		} else {
			bestOrder.Status = models.OrderStatusPartiallyFilled
			heap.Pop(oppositeHeap)
			heap.Push(oppositeHeap, bestOrder)
		}

		books.Orders[bestOrder.ID] = bestOrder
		
		buyOrderID := bestOrder.ID
		sellOrderID := order.ID
		buyerUserID := bestOrder.UserID
		sellerUserID := order.UserID

		if order.Side == models.OrderSideSell {
			buyOrderID, sellOrderID = sellOrderID, buyOrderID
			buyerUserID, sellerUserID = sellerUserID, buyerUserID
		}

		trade := Trade{
			ID: generateTradeID(),
			Ticker: order.Ticker,
			BuyOrderID: buyOrderID,
			SellOrderID: sellOrderID,
			Price: bestOrder.Price,
			Quantity: fillQuantity,
			BuyerUserID: buyerUserID,
			SellerUserID: sellerUserID,
			CreatedAt: time.Now(),
		}

		trades = append(trades, trade)
	}

	if(order.Quantity == 0) {
		order.Status = models.OrderStatusFilled
	} else if len(trades) > 0 {
		order.Status = models.OrderStatusPartiallyFilled
	} else {
		order.Status = models.OrderStatusPending
	}

	books.Orders[order.ID] = order


	return trades, order;
}


func generateTradeID() string {
	return uuid.New().String()
}