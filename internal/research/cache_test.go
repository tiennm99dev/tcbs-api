package research

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheTTL(t *testing.T) {
	now := testNow
	c := Cache{Dir: t.TempDir(), Now: func() time.Time { return now }}
	key := cacheKey("vndirect", "ratios", "FPT")
	if err := c.Put(key, Fundamentals{Symbol: "FPT", PE: ptr(12)}); err != nil {
		t.Fatal(err)
	}

	var f Fundamentals
	if !c.Get(key, time.Hour, &f) || f.Symbol != "FPT" || f.PE == nil || *f.PE != 12 {
		t.Fatalf("hit = %+v", f)
	}
	now = testNow.Add(59 * time.Minute)
	if !c.Get(key, time.Hour, &f) {
		t.Error("want a hit before the TTL")
	}
	now = testNow.Add(time.Hour)
	if c.Get(key, time.Hour, &f) {
		t.Error("want a miss at the TTL")
	}
	if c.Get(cacheKey("vndirect", "ratios", "VCB"), time.Hour, &f) {
		t.Error("want a miss for an absent key")
	}
}

func TestCacheRefresh(t *testing.T) {
	c := Cache{Dir: t.TempDir(), Now: func() time.Time { return testNow }}
	key := cacheKey("kbs", "profit", "FPT")
	if err := c.Put(key, []float64{1, 2, 3}); err != nil {
		t.Fatal(err)
	}
	c.Refresh = true
	var v []float64
	if c.Get(key, time.Hour, &v) {
		t.Error("Refresh must force a miss")
	}
	// Writes still land, so the next run without Refresh reads them.
	if err := c.Put(key, []float64{4, 5, 6}); err != nil {
		t.Fatal(err)
	}
	c.Refresh = false
	if !c.Get(key, time.Hour, &v) || v[0] != 4 {
		t.Errorf("got %v", v)
	}
}

func TestCacheAtomicWrite(t *testing.T) {
	dir := t.TempDir()
	c := Cache{Dir: dir, Now: func() time.Time { return testNow }}
	key := cacheKey("bars", "daily", "FPT-20260101-20261013")
	for i := 0; i < 3; i++ {
		if err := c.Put(key, Series{Symbol: "FPT", Bars: make([]Bar, i)}); err != nil {
			t.Fatal(err)
		}
	}

	path := filepath.Join(dir, "bars", "daily", "FPT-20260101-20261013.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("file mode = %o, want 600", mode)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want only the final file (no temp files left)", len(entries))
	}
	var s Series
	if !c.Get(key, time.Hour, &s) || len(s.Bars) != 2 {
		t.Errorf("last write not visible: %+v", s)
	}
}

func TestCacheCorruptEntryMisses(t *testing.T) {
	dir := t.TempDir()
	c := Cache{Dir: dir, Now: func() time.Time { return testNow }}
	path := filepath.Join(dir, "x", "y", "z.json")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	var v any
	if c.Get("x/y/z.json", time.Hour, &v) {
		t.Error("want a miss for a corrupt entry")
	}
}

func TestCacheRejectsEscapingKeys(t *testing.T) {
	c := Cache{Dir: t.TempDir()}
	for _, key := range []string{"", "../x.json", "/etc/x.json", "a/../../x.json"} {
		if err := c.Put(key, 1); err == nil {
			t.Errorf("Put(%q) succeeded", key)
		}
	}
}

func TestCacheDisabled(t *testing.T) {
	var c Cache
	if err := c.Put("a/b/c.json", 1); err != nil {
		t.Fatal(err)
	}
	var v int
	if c.Get("a/b/c.json", time.Hour, &v) {
		t.Error("a cache without Dir must always miss")
	}
}

func TestCacheKey(t *testing.T) {
	if got := cacheKey("vndirect", "profile", "FPT"); got != "vndirect/profile/FPT.json" {
		t.Errorf("got %q", got)
	}
	if got := cacheKey("x", "y", "../../etc"); got == "x/y/../../etc.json" || filepath.Dir(got) != "x/y" {
		t.Errorf("unsafe name not hashed: %q", got)
	}
}

func TestDailyTTL(t *testing.T) {
	ict := time.FixedZone("ICT", 7*3600)
	for _, tc := range []struct {
		at   time.Time
		want time.Duration
	}{
		{time.Date(2026, 10, 13, 10, 0, 0, 0, ict), 6 * time.Hour},  // Tuesday, market hours
		{time.Date(2026, 10, 13, 15, 29, 0, 0, ict), 6 * time.Hour}, // just before 15:30
		{time.Date(2026, 10, 13, 15, 30, 0, 0, ict), 18 * time.Hour},
		{time.Date(2026, 10, 13, 8, 59, 0, 0, ict), 18 * time.Hour},
		{time.Date(2026, 10, 17, 10, 0, 0, 0, ict), 18 * time.Hour}, // Saturday
	} {
		if got := DailyTTL(tc.at); got != tc.want {
			t.Errorf("DailyTTL(%v) = %v, want %v", tc.at, got, tc.want)
		}
	}
}

func TestDailyFresh(t *testing.T) {
	ict := time.FixedZone("ICT", 7*3600)
	at := func(d, h, m int) time.Time { return time.Date(2026, 10, d, h, m, 0, 0, ict) }
	for _, tc := range []struct {
		name        string
		saved, read time.Time
		want        bool
	}{
		{"intraday save, read after the close", at(13, 10, 0), at(13, 16, 0), false},
		{"pre-open save, read after the close", at(13, 8, 0), at(13, 16, 0), false},
		{"intraday save, read an hour later", at(13, 10, 0), at(13, 11, 0), true},
		{"intraday save, read past 6h", at(13, 9, 0), at(13, 15, 1), false},
		{"after-close save, read that evening", at(13, 16, 0), at(13, 22, 0), true},
		{"after-close save, read next morning", at(13, 16, 0), at(14, 8, 0), true},
		{"after-close save, read in the next session", at(13, 16, 0), at(14, 9, 30), false},
		{"Friday evening save, read Saturday", at(16, 18, 0), at(17, 10, 0), true},
		{"Saturday save, read Monday after the close", at(17, 10, 0), at(19, 16, 0), false},
		{"saved in the future", at(13, 16, 0), at(13, 15, 0), false},
	} {
		if got := DailyFresh(tc.saved, tc.read); got != tc.want {
			t.Errorf("%s: DailyFresh = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestDailyCacheMissAfterClose(t *testing.T) {
	ict := time.FixedZone("ICT", 7*3600)
	now := time.Date(2026, 10, 13, 10, 0, 0, 0, ict)
	c := Cache{Dir: t.TempDir(), Now: func() time.Time { return now }}
	key := cacheKey("bars", "daily", "FPT-20250101-20261013")
	if err := c.Put(key, Series{Symbol: "FPT"}); err != nil {
		t.Fatal(err)
	}
	var s Series
	now = now.Add(time.Hour)
	if !c.GetFresh(key, DailyFresh, &s) {
		t.Error("want a hit within the session")
	}
	now = time.Date(2026, 10, 13, 16, 0, 0, 0, ict)
	if c.GetFresh(key, DailyFresh, &s) {
		t.Error("want a miss after the close for bars saved during the session")
	}
}
