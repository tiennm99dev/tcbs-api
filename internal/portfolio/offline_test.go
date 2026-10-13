package portfolio

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "holdings.json")
	// A pre-existing world-readable file must end up 0600.
	if err := os.WriteFile(path, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	in := Snapshot{
		AsOf:    time.Date(2026, 10, 13, 15, 0, 0, 0, time.UTC),
		Custody: "105C001234",
		Accounts: []Account{{
			AccountNo: "0001N", Type: "normal", Status: "A",
			Cash:     Cash{Balance: 1000},
			Holdings: []Holding{{Symbol: "FPT", TotalQty: 100, AvgCost: 95000, RefPrice: 120000, SecType: "001"}},
		}},
	}
	if err := Save(path, in); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode %o, want 600", perm)
	}
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "105C001234") {
		t.Errorf("custody code saved unmasked: %s", b)
	}

	got, err := FileSource{Path: path}.Snapshot(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := in
	want.Custody = "105C***1234"
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round trip:\n got %+v\nwant %+v", got, want)
	}
}

func TestFileSource_Errors(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		return p
	}
	for name, path := range map[string]string{
		"missing":       filepath.Join(dir, "absent.json"),
		"bad json":      write("bad.json", "{"),
		"unknown field": write("unknown.json", `{"accounts":[{"accountNo":"1","bogus":1}]}`),
		"no accounts":   write("empty.json", `{"accounts":[]}`),
	} {
		if _, err := (FileSource{Path: path}).Snapshot(context.Background()); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestRepoHoldingsFixture(t *testing.T) {
	s, err := FileSource{Path: filepath.Join("..", "..", "testdata", "holdings.json")}.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("testdata/holdings.json: %v", err)
	}
	sum := Aggregate(s, nil, false)
	if sum.NAV <= 0 || len(sum.Positions) == 0 {
		t.Errorf("fixture should yield a positive NAV and positions: %+v", sum)
	}
}

func TestMaskCustody(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"105C001234":  "105C***1234",
		"105C***1234": "105C***1234",
		"short":       "***",
	}
	for in, want := range cases {
		if got := MaskCustody(in); got != want {
			t.Errorf("MaskCustody(%q) = %q, want %q", in, got, want)
		}
	}
}
