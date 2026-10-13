package tcbs

import (
	"context"
	"net/http"
	"testing"
)

func TestGetStockPrices(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tartarus/v1/tickerCommons" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		tickers := r.URL.Query().Get("tickers")
		if tickers != "FPT,VNM" {
			t.Errorf("unexpected tickers: %s", tickers)
		}
		// The documented envelope and row shape of tickerCommons.
		_, _ = w.Write([]byte(`{"data":[` +
			`{"symbol":"FPT","ceilPrice":106100,"floorPrice":92300,"refPrice":99200,"bidPrice01":98700,"matchPrice":120000,"room":84},` +
			`{"symbol":"VNM","refPrice":"79500","matchPrice":80000}` +
			`],"tradingDate":"29/10/2024"}`))
	})

	resp, err := client.GetStockPrices(context.Background(), []string{"FPT", "VNM"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 2 {
		t.Fatalf("expected 2 items, got %d", len(resp))
	}
	if resp[0].Symbol != "FPT" || resp[0].MatchPrice != 120000 || resp[0].RefPrice != 99200 || resp[0].CeilPrice != 106100 || resp[0].BidPrice01 != 98700 {
		t.Errorf("unexpected FPT data: %+v", resp[0])
	}
	if resp[1].Symbol != "VNM" || resp[1].RefPrice != 79500 {
		t.Errorf("unexpected VNM data: %+v", resp[1])
	}
}

func TestGetIntradayHistory(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nyx/v1/intraday/FPT/his/paging" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, IntradayHistoryResponse{
			Ticker: "FPT",
			Data:   []IntradayHistoryItem{{P: 120000, V: 100}},
		})
	})

	resp, err := client.GetIntradayHistory(context.Background(), IntradayHistoryParams{
		Ticker: "FPT", Page: 0, Size: 20,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Data))
	}
}

func TestGetSupplyDemand(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nyx/v1/intraday/FPT/bsa" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, SupplyDemand15mResponse{
			Ticker: "FPT",
			Data:   []SupplyDemand15mItem{{BU: 100, SD: 50}},
		})
	})

	resp, err := client.GetSupplyDemand(context.Background(), "FPT", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Data) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Data))
	}
}

func TestGetSupplyDemandExt(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nyx/v1/intraday/FPT/bsa-ext" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, SupplyDemand15mResponse{Ticker: "FPT"})
	})

	_, err := client.GetSupplyDemandExt(context.Background(), "FPT", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetSupplyDemandMonth(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nyx/v1/intraday/FPT/bsa-month" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, SupplyDemandResponse{Ticker: "FPT"})
	})

	_, err := client.GetSupplyDemandMonth(context.Background(), "FPT", "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetForeignRoom(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []ForeignRoomInfo{{Ticker: "FPT", TotalRoom: 1000}})
	})

	resp, err := client.GetForeignRoom(context.Background(), []string{"FPT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].Ticker != "FPT" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetPutThroughInfo(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []PutThroughMatchInfo{{Symbol: "FPT", Vol: 500}})
	})

	resp, err := client.GetPutThroughInfo(context.Background(), []string{"FPT"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].Symbol != "FPT" {
		t.Errorf("unexpected response: %+v", resp)
	}
}
