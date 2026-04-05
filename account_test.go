package tcbs

import (
	"context"
	"net/http"
	"testing"
)

func TestGetSubAccountInfo(t *testing.T) {
	client, _ := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/eros/v2/get-profile/by-username/105C001" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("fields") != "basicInfo" {
			t.Errorf("unexpected fields param: %s", r.URL.Query().Get("fields"))
		}
		writeJSON(t, w, AccountInformationResponse{
			BasicInfo: &BasicInfo{Code105C: "105C001", Status: "active"},
		})
	})

	resp, err := client.GetSubAccountInfo(context.Background(), "105C001", "basicInfo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.BasicInfo == nil || resp.BasicInfo.Code105C != "105C001" {
		t.Error("unexpected response")
	}
}
