package tcbs

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestTransferMoney(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/physis/v1/stock/transfer" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req MoneyTransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if req.SourceAccountNumber != "ACC1" || req.Amount != 1000000 {
			t.Errorf("unexpected request: %+v", req)
		}
		writeJSON(t, w, MoneyTransferResponse{Status: "ok"})
	})

	resp, err := client.TransferMoney(context.Background(), &MoneyTransferRequest{
		SourceAccountNumber:      "ACC1",
		DestinationAccountNumber: "ACC2",
		Amount:                   1000000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", resp.Status)
	}
}

func TestDepositMargin(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/khronos/v1/cash/deposit/update" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, MarginDepositResponse{TransactionID: "TXN-1"})
	})

	resp, err := client.DepositMargin(context.Background(), &MarginDepositRequest{
		AccountID: "ACC1", SubAccountID: "SUB1", Amount: 5000000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TransactionID != "TXN-1" {
		t.Errorf("expected TXN-1, got %q", resp.TransactionID)
	}
}

func TestWithdrawMargin(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/khronos/v1/cash/withdraw/update" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		writeJSON(t, w, MarginWithdrawResponse{TransactionID: "TXN-2"})
	})

	resp, err := client.WithdrawMargin(context.Background(), &MarginWithdrawRequest{
		AccountID: "ACC1", SubAccountID: "SUB1", Amount: 3000000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.TransactionID != "TXN-2" {
		t.Errorf("expected TXN-2, got %q", resp.TransactionID)
	}
}
