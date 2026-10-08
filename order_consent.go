package consent

import "fmt"

type OrderEvent string

const (
	EventCheckout    OrderEvent = "checkout"
	EventFulfillment OrderEvent = "fulfillment"
	EventReceipt     OrderEvent = "receipt"
	EventOrderUpdate OrderEvent = "customer_order_update"
)

var consentCategory = map[OrderEvent]string{
	EventCheckout:    "analytics",
	EventFulfillment: "essential",
	EventReceipt:     "marketing",
	EventOrderUpdate: "third_party",
}

func CategoryFor(event OrderEvent) (string, error) {
	category, ok := consentCategory[event]
	if !ok {
		return "", fmt.Errorf("unknown order event %q", event)
	}
	return category, nil
}

type DispatchDecision struct {
	OrderID  string     `json:"order_id"`
	Event    OrderEvent `json:"event"`
	Category string     `json:"category"`
	Dispatch bool       `json:"dispatch"`
}

func DecideDispatch(orderID string, event OrderEvent, granted bool) (DispatchDecision, error) {
	category, err := CategoryFor(event)
	if err != nil {
		return DispatchDecision{}, err
	}
	return DispatchDecision{
		OrderID:  orderID,
		Event:    event,
		Category: category,
		Dispatch: granted,
	}, nil
}
