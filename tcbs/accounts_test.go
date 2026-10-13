package tcbs

import (
	"context"
	"net/http"
	"testing"
)

func TestActiveStockAccounts(t *testing.T) {
	in := []MarginQuotaResponse{
		{AccountNo: "0001N", AccountStatus: "A", AccountType: "normal"},
		{AccountNo: "0001G", AccountStatus: "A", AccountType: "margin"},
		{AccountNo: "0001X", AccountStatus: "C", AccountType: "normal"},
		{AccountNo: "0001W", AccountStatus: "A", AccountType: "CW"},
		{AccountNo: "0001S", AccountStatus: "A", AccountType: "SPW"},
		{AccountNo: "0001T", AccountStatus: "A", AccountType: "spt"},
		{AccountNo: "0001P", AccountStatus: "P", AccountType: "copier"},
	}
	got := ActiveStockAccounts(in)
	want := []string{"0001N", "0001G", "0001P"}
	if len(got) != len(want) {
		t.Fatalf("expected %d accounts, got %d: %+v", len(want), len(got), got)
	}
	for i, a := range got {
		if a.AccountNo != want[i] {
			t.Errorf("account %d: expected %s, got %s", i, want[i], a.AccountNo)
		}
	}
}

func TestActiveStockAccounts_Empty(t *testing.T) {
	if got := ActiveStockAccounts(nil); len(got) != 0 {
		t.Errorf("expected no accounts, got %+v", got)
	}
}

func TestListAccounts(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/aion/v1/customers/105C000001/accounts" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"accountNo":"0001N","accountStatus":"A","accountType":"normal","marginLimit":"0"},
			{"accountNo":"0001G","accountStatus":"A","accountType":"margin","marginLimit":500000000}]`))
	})

	got, err := client.ListAccounts(context.Background(), "105C000001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || got[1].MarginLimit != 500000000 {
		t.Errorf("unexpected response: %+v", got)
	}
}
