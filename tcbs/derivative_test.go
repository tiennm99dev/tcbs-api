package tcbs

import (
	"context"
	"net/http"
	"testing"
)

func TestGetDerivativeCashStatus(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/khronos/v1/account/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("accountId") != "ACC1" {
			t.Errorf("unexpected accountId: %s", r.URL.Query().Get("accountId"))
		}
		writeJSON(t, w, DerivativeResponse[*TotalCashDerivativeResponse]{
			RC:   "0",
			Data: &TotalCashDerivativeResponse{NAV: 50000000, Cash: 10000000},
		})
	})

	resp, err := client.GetDerivativeCashStatus(context.Background(), "ACC1", "SUB1", "0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data == nil || resp.Data.NAV != 50000000 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetDerivativeClosedPositions(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[[]AssetPositionCloseDerivativeResponse]{
			Data: []AssetPositionCloseDerivativeResponse{{Symbol: "VN30F2503", Side: "B"}},
		})
	})

	resp, err := client.GetDerivativeClosedPositions(context.Background(), DerivativePositionCloseParams{
		AccountID: "ACC1", SubAccountID: "SUB1", PageNo: 1, PageSize: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Symbol != "VN30F2503" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetDerivativeOpenPositions(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[[]AssetPositionOpenDerivativeResponse]{
			Data: []AssetPositionOpenDerivativeResponse{{Symbol: "VN30F2503", Net: 5}},
		})
	})

	resp, err := client.GetDerivativeOpenPositions(context.Background(), "ACC1", "SUB1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].Net != 5 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetDerivativeNormalOrders(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[[]DerivativeNormalOrderResponse]{
			Data: []DerivativeNormalOrderResponse{{OrderNo: "N001", Symbol: "VN30F2503"}},
		})
	})

	resp, err := client.GetDerivativeNormalOrders(context.Background(), DerivativeOrdersParams{
		PageNo: 1, PageSize: 10, AccountID: "ACC1", Symbol: "ALL,ALL", Status: "0",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].OrderNo != "N001" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPlaceDerivativeNormalOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		writeJSON(t, w, DerivativeResponse[*DerivativeNormalOrderPlaceResponse]{
			RC:   "0",
			Data: &DerivativeNormalOrderPlaceResponse{OrderNo: "N002", Symbol: "VN30F2503"},
		})
	})

	resp, err := client.PlaceDerivativeNormalOrder(context.Background(), &DerivativeNormalOrderRequest{
		AccountID: "ACC1", SubAccountID: "SUB1", Side: "B",
		Symbol: "VN30F2503", Price: 1200, Volume: 1, OrderType: "LO",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data == nil || resp.Data.OrderNo != "N002" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetDerivativeConditionOrders(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/khronos/v1/order/condition/detail" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, DerivativeResponse[[]DerivativeConditionOrderResponse]{
			Data: []DerivativeConditionOrderResponse{{OrderNo: "C001", Symbol: "VN30F2503"}},
		})
	})

	resp, err := client.GetDerivativeConditionOrders(context.Background(), "ACC1", "SUB1", 1, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 || resp.Data[0].OrderNo != "C001" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestPlaceDerivativeConditionOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[*DerivativeConditionOrderPlaceResponse]{
			Data: &DerivativeConditionOrderPlaceResponse{OrderNo: 101, Symbol: "VN30F2503"},
		})
	})

	resp, err := client.PlaceDerivativeConditionOrder(context.Background(), &DerivativeConditionOrderRequest{
		AccountID: "ACC1", SubAccountID: "SUB1", Side: "B",
		Symbol: "VN30F2503", Price: 1200, Volume: 1, OrderType: "LO",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data == nil || resp.Data.OrderNo != 101 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestChangeDerivativeNormalOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[string]{RC: "0", Data: "ok"})
	})

	_, err := client.ChangeDerivativeNormalOrder(context.Background(), &DerivativeChangeNormalOrderRequest{
		AccountID: "ACC1", SubAccountID: "SUB1", OrderNo: "N001", RefID: "ref1", NVol: 2, NPrice: 1300,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangeDerivativeConditionOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[string]{RC: "0", Data: "ok"})
	})

	_, err := client.ChangeDerivativeConditionOrder(context.Background(), &DerivativeChangeConditionOrderRequest{
		AccountID: "ACC1", PKOrderNo: "PK001", Type: "SL", RefID: "ref1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCancelDerivativeNormalOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[*DerivativeCancelNormalOrderResponse]{
			Data: &DerivativeCancelNormalOrderResponse{OrderNo: "N001", Status: "cancelled"},
		})
	})

	resp, err := client.CancelDerivativeNormalOrder(context.Background(), &DerivativeCancelNormalOrderRequest{
		AccountID: "ACC1", OrderNo: "N001", Cmd: "cancel",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Data == nil || resp.Data.Status != "cancelled" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestCancelDerivativeConditionOrder(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, DerivativeResponse[string]{RC: "0", Data: "ok"})
	})

	_, err := client.CancelDerivativeConditionOrder(context.Background(), &DerivativeCancelConditionOrderRequest{
		AccountID: "ACC1", SubAccountID: "SUB1", OrderNo: "C001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetDerivativeMarketInfo(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []DerivativeMarketInfo{
			{Ticker: "VN30F2503", LastPrice: 1250.5, OpenInterest: 30000},
		})
	})

	resp, err := client.GetDerivativeMarketInfo(context.Background(), []string{"VN30F2503"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].Ticker != "VN30F2503" {
		t.Errorf("unexpected response: %+v", resp)
	}
}
