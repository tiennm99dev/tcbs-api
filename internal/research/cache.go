package research

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Cache TTLs per kind of data.
const (
	TTLRatios       = 24 * time.Hour
	TTLEvents       = 24 * time.Hour
	TTLProfile      = 30 * 24 * time.Hour
	TTLConstituents = 7 * 24 * time.Hour
)

// Cache is a file cache of normalised public data, keyed by
// "<provider>/<kind>/<name>.json" under Dir. An empty Dir disables it.
type Cache struct {
	Dir     string
	Now     func() time.Time
	Refresh bool // when set, Get always misses so every entry is refetched
}

type cacheEntry struct {
	Saved time.Time       `json:"saved"`
	Data  json.RawMessage `json:"data"`
}

func (c Cache) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// Get decodes the entry for key into v when it exists and is younger than
// ttl.
func (c Cache) Get(key string, ttl time.Duration, v any) (ok bool) {
	return c.GetFresh(key, func(saved, now time.Time) bool {
		age := now.Sub(saved)
		return age >= 0 && age < ttl
	}, v)
}

// GetFresh decodes the entry for key into v when it exists and
// fresh(saved, now) holds for the time it was saved.
func (c Cache) GetFresh(key string, fresh func(saved, now time.Time) bool, v any) (ok bool) {
	if c.Dir == "" || c.Refresh {
		return false
	}
	path, err := c.path(key)
	if err != nil {
		return false
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var e cacheEntry
	if err := json.Unmarshal(b, &e); err != nil || len(e.Data) == 0 {
		return false
	}
	if !fresh(e.Saved, c.now()) {
		return false
	}
	return json.Unmarshal(e.Data, v) == nil
}

// Put stores v under key. The file is written to a temp file with mode 0600
// and renamed into place, so readers never see a partial entry.
func (c Cache) Put(key string, v any) error {
	if c.Dir == "" {
		return nil
	}
	path, err := c.path(key)
	if err != nil {
		return err
	}
	data, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("research: cache encode %s: %w", key, err)
	}
	b, err := json.Marshal(cacheEntry{Saved: c.now(), Data: data})
	if err != nil {
		return fmt.Errorf("research: cache encode %s: %w", key, err)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("research: cache dir: %w", err)
	}
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("research: cache temp file: %w", err)
	}
	tmp := f.Name()
	defer os.Remove(tmp) // no-op after a successful rename

	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return fmt.Errorf("research: cache chmod: %w", err)
	}
	if _, err := f.Write(b); err != nil {
		f.Close()
		return fmt.Errorf("research: cache write: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("research: cache close: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("research: cache rename: %w", err)
	}
	return nil
}

// path maps a key to a file under Dir and rejects keys that would escape it.
func (c Cache) path(key string) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(key))
	if key == "" || filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("research: invalid cache key %q", key)
	}
	return filepath.Join(c.Dir, clean), nil
}

// cacheKey builds "<provider>/<kind>/<name>.json". Names that are not plain
// ticker-like strings are replaced by a short hash.
func cacheKey(provider, kind, name string) string {
	if !safeName(name) {
		sum := sha256.Sum256([]byte(name))
		name = hex.EncodeToString(sum[:8])
	}
	return provider + "/" + kind + "/" + name + ".json"
}

func safeName(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		switch {
		case r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

// marketTZ is the exchange time zone. Vietnam has no daylight saving time,
// so the fixed +07:00 fallback is exact when tzdata is unavailable.
var marketTZ = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err != nil {
		return time.FixedZone("ICT", 7*60*60)
	}
	return loc
}()

// DailyTTL is how long cached daily bars stay fresh when read at now: 6h on
// weekdays between 09:00 and 15:30 exchange time, when today's bar is still
// moving, and 18h otherwise.
func DailyTTL(now time.Time) time.Duration {
	t := now.In(marketTZ)
	if wd := t.Weekday(); wd == time.Saturday || wd == time.Sunday {
		return 18 * time.Hour
	}
	minutes := t.Hour()*60 + t.Minute()
	if minutes >= 9*60 && minutes < 15*60+30 {
		return 6 * time.Hour
	}
	return 18 * time.Hour
}

// sessionClose is the end of the continuous session, 15:30 exchange time.
const sessionClose = 15*time.Hour + 30*time.Minute

// DailyFresh reports whether daily bars saved at saved are still current at
// now. They must be younger than the DailyTTL of both times, and no session
// close may fall between saving and reading: bars saved before a close hold a
// partial or missing bar for that day.
func DailyFresh(saved, now time.Time) bool {
	age := now.Sub(saved)
	if age < 0 || age >= min(DailyTTL(saved), DailyTTL(now)) {
		return false
	}
	return !saved.Before(lastSessionClose(now))
}

// lastSessionClose is the latest weekday 15:30 exchange time at or before
// now. Exchange holidays are not modelled.
func lastSessionClose(now time.Time) time.Time {
	t := now.In(marketTZ)
	day := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, marketTZ)
	for {
		if wd := day.Weekday(); wd != time.Saturday && wd != time.Sunday {
			if c := day.Add(sessionClose); !c.After(t) {
				return c
			}
		}
		day = day.AddDate(0, 0, -1)
	}
}
