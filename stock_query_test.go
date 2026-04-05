package tcbs

import (
	"context"
	"net/http"
	"testing"
)

func TestGetOrders(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aion/v1/accounts/ACC001/orders" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, OrderSearchResponse{TotalCount: 5})
	})

	resp, err := client.GetOrders(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 5 {
		t.Errorf("expected 5, got %d", resp.TotalCount)
	}
}

func TestGetPurchasingPower(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, PurchasingPowerResponse{PP0: 100000000})
	})

	resp, err := client.GetPurchasingPower(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PP0 != 100000000 {
		t.Errorf("expected PP0=100000000, got %.0f", resp.PP0)
	}
}

func TestGetMarginQuota(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []MarginQuotaResponse{{AccountNo: "ACC001", MarginLimit: 500000}})
	})

	resp, err := client.GetMarginQuota(context.Background(), "CUS001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].AccountNo != "ACC001" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetStockAssets(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, SeInfoDTO{
			AccountNo: "ACC001",
			Stock:     []StockHoldingInfo{{Symbol: "FPT", TotalQtty: 1000}},
		})
	})

	resp, err := client.GetStockAssets(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Stock) != 1 || resp.Stock[0].Symbol != "FPT" {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetOrderByID(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aion/v1/accounts/ACC001/orders/ORD-1" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, OrderSearchResponse{TotalCount: 1})
	})

	resp, err := client.GetOrderByID(context.Background(), "ACC001", "ORD-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("expected 1, got %d", resp.TotalCount)
	}
}

func TestGetMatchingDetails(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, CommandMatchInformationResponse{TotalCount: 2})
	})

	resp, err := client.GetMatchingDetails(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 2 {
		t.Errorf("expected 2, got %d", resp.TotalCount)
	}
}

func TestGetPurchasingPowerBySymbol(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, PurchasingPowerResponse{PP0: 50000000})
	})

	resp, err := client.GetPurchasingPowerBySymbol(context.Background(), "ACC001", "FPT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PP0 != 50000000 {
		t.Errorf("unexpected PP0: %.0f", resp.PP0)
	}
}

func TestGetPurchasingPowerBySymbolPrice(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, PurchasingPowerResponse{MaxBuyQuantity: 100})
	})

	resp, err := client.GetPurchasingPowerBySymbolPrice(context.Background(), "ACC001", "FPT", "120000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.MaxBuyQuantity != 100 {
		t.Errorf("unexpected MaxBuyQuantity: %.0f", resp.MaxBuyQuantity)
	}
}

func TestGetMarginAccountInfo(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, []MarginAccountInfoResponse{{AccountNo: "ACC001", RTT: 1.5}})
	})

	resp, err := client.GetMarginAccountInfo(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp) != 1 || resp[0].RTT != 1.5 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetSupplementaryLoanPackages(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, SupplementaryLoanPackageResponse{
			MarginSureViews: []MarginSureView{{Name: "pkg1"}},
		})
	})

	resp, err := client.GetSupplementaryLoanPackages(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.MarginSureViews) != 1 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetLoans(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, LoanResponse{Size: 1, Content: []LoanItem{{Symbol: "FPT"}}})
	})

	resp, err := client.GetLoans(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Size != 1 {
		t.Errorf("expected size 1, got %d", resp.Size)
	}
}

func TestGetCashBalance(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, CashInvestmentResponse{TotalCount: 1})
	})

	resp, err := client.GetCashBalance(context.Background(), "ACC001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TotalCount != 1 {
		t.Errorf("expected 1, got %d", resp.TotalCount)
	}
}

func TestGetMarginInfo(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(t, w, MarginInfoResponse{
			Response: &MarginInfoData{TotalRow: 3},
		})
	})

	resp, err := client.GetMarginInfo(context.Background(), MarginInfoParams{
		AccountNo: "ACC001", FromDate: "2025-01-01", ToDate: "2025-01-31",
		Page: "0", Size: "10", CustodyCD: "CUS001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Response == nil || resp.Response.TotalRow != 3 {
		t.Errorf("unexpected response: %+v", resp)
	}
}

func TestGetCashStatements(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/erebos/v2/digital/trans-hist-cashStatements" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("fromDate") != "2025-01-01" {
			t.Errorf("unexpected fromDate: %s", r.URL.Query().Get("fromDate"))
		}
		writeJSON(t, w, TransHistCashStatementsResponse{
			Response: &TransHistCashStatementsData{TotalCount: 3},
		})
	})

	resp, err := client.GetCashStatements(context.Background(), CashStatementParams{
		AccountNo: "ACC001", FromDate: "2025-01-01", ToDate: "2025-01-31",
		PageSize: "10", PageIndex: "0", TransactionCode: "",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Response == nil || resp.Response.TotalCount != 3 {
		t.Errorf("unexpected response: %+v", resp)
	}
}
