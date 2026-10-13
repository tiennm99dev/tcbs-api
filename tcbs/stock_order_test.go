package tcbs

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestPlaceOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/akhlys/v1/accounts/ACC001/orders" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req PlaceOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if req.Symbol != "FPT" || req.Quantity != 100 {
			t.Errorf("unexpected request: %+v", req)
		}
		writeJSON(t, w, PlaceOrderResponse{OrderID: "ORD-1"})
	})

	resp, err := client.PlaceOrder(context.Background(), "ACC001", &PlaceOrderRequest{
		Symbol:    "FPT",
		ExecType:  "NB",
		Quantity:  100,
		Price:     120000,
		PriceType: "LO",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OrderID != "ORD-1" {
		t.Errorf("expected order ID 'ORD-1', got %q", resp.OrderID)
	}
}

func TestUpdateOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		writeJSON(t, w, UpdateOrderResponse{OrderID: "ORD-1", Message: "ok"})
	})

	resp, err := client.UpdateOrder(context.Background(), "ACC001", "ORD-1", &UpdateOrderRequest{
		Price: 125000, Quantity: 200,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.OrderID != "ORD-1" {
		t.Errorf("expected 'ORD-1', got %q", resp.OrderID)
	}
}

func TestCancelOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		writeJSON(t, w, CancelOrderResponse{TotalCount: 1})
	})

	resp, err := client.CancelOrder(context.Background(), "ACC001", &CancelOrderRequest{
		OrdersList: []OrderIDRef{{OrderID: "ORD-1"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("expected totalCount 1, got %d", resp.TotalCount)
	}
}
