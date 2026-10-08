package consent

import "testing"

func TestDecideDispatch(t *testing.T) {
	tests := []struct {
		name         string
		event        OrderEvent
		granted      bool
		wantCategory string
		wantDispatch bool
	}{
		{"checkout granted", EventCheckout, true, "analytics", true},
		{"fulfillment revoked", EventFulfillment, false, "essential", false},
		{"receipt granted", EventReceipt, true, "marketing", true},
		{"updates revoked", EventOrderUpdate, false, "third_party", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecideDispatch("ord_2048", tt.event, tt.granted)
			if err != nil {
				t.Fatal(err)
			}
			if got.Category != tt.wantCategory || got.Dispatch != tt.wantDispatch {
				t.Fatalf("got category=%q dispatch=%v", got.Category, got.Dispatch)
			}
		})
	}
}
