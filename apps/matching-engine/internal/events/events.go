package events

import "time"

type Event[T any] struct {
	ID string
	Type EventType
	OccuredAt time.Time
	Data T
}

type EventType string 

const (
	OrderBookCreated EventType = "ORDER_BOOK_CREATED"
	OrderPlaced EventType = "ORDER_PLACED"
	OrderFilled EventType = "ORDER_FILLED"
	OrderPartiallyFilled EventType = "ORDER_PARTIALLY_FILLED"
	OrderCanceled EventType = "ORDER_CANCELED"
	OrderRejected EventType = "ORDER_REJECTED"
	BidAdded EventType = "BID_ADDED"
	AskAdded EventType = "ASK_ADDED"
	BidRemoved EventType = "BID_REMOVED"
	AskRemoved EventType = "ASK_REMOVED"
	TradeCreated EventType = "TRADE_CREATED"
)


type OrderBookCreatedData struct {
	Ticker string
}

type TradeCreatedData struct {
	Ticker string
	BuyOrderID string
	SellOrderID string
	Price float64
	Quantity int
	SellerUserID string
	BuyerUserID string
}

type OrderEventData struct {
	OrderID string
	Ticker string
	UserID string
	Price float64
	Quantity int
}


type EventPublisher interface {
	Publish(eventType EventType, data any) error
}

type MockPublisher struct {
	Events []struct {
		Type EventType
		Data any
	}
}

func (m *MockPublisher) Publish(eventType EventType, data any) error {
	m.Events = append(m.Events, struct {
		Type EventType
		Data any
	}{
		 eventType, data})
	return nil
}