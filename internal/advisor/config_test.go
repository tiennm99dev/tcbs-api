package advisor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeRules(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "rules.json")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDefault(t *testing.T) {
	c := Default()
	if c.SingleSoftCap != 0.20 || c.SingleHardCap != 0.25 || c.SectorCap != 0.40 || c.FeeRate != 0.001 ||
		c.CandidateMinScore != 70 || c.CoE != 0.13 || c.Scoring.VOL != (Band{0.20, 0.60}) {
		t.Errorf("defaults changed: %+v", c)
	}
}

func TestLoadOverrides(t *testing.T) {
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name    string
		body    string
		wantErr string
		check   func(t *testing.T, c Config)
	}{
		{
			name: "named fields only",
			body: `{"SingleSoftCap": 0.18, "Scoring": {"VOL": {"Best": 0.25}, "MinPeers": 7}}`,
			check: func(t *testing.T, c Config) {
				if c.SingleSoftCap != 0.18 || c.SingleHardCap != 0.25 {
					t.Errorf("caps %v %v", c.SingleSoftCap, c.SingleHardCap)
				}
				if c.Scoring.VOL != (Band{0.25, 0.60}) || c.Scoring.MinPeers != 7 || c.Scoring.MDD != (Band{0.15, 0.50}) {
					t.Errorf("scoring %+v", c.Scoring)
				}
				if !c.Now.Equal(now) {
					t.Errorf("Now changed: %v", c.Now)
				}
			},
		},
		{
			name: "curve replaced",
			body: `{"Scoring": {"PBGeneral": [{"X": 2, "Score": 100}, {"X": 4, "Score": 0}]}}`,
			check: func(t *testing.T, c Config) {
				if len(c.Scoring.PBGeneral) != 2 || c.Scoring.PBGeneral[1] != (Point{4, 0}) {
					t.Errorf("PBGeneral %+v", c.Scoring.PBGeneral)
				}
			},
		},
		{name: "unknown field", body: `{"SingleSoftCapp": 0.1}`, wantErr: "unknown field"},
		{name: "bad json", body: `{`, wantErr: "parse rules"},
		{name: "soft above hard", body: `{"SingleSoftCap": 0.3}`, wantErr: "above SingleHardCap"},
		{name: "now ignored", body: `{"Now": "2020-01-01T00:00:00Z"}`, wantErr: "unknown field"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Default()
			c.Now = now
			before := c
			err := LoadOverrides(writeRules(t, tc.body), &c)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err = %v, want %q", err, tc.wantErr)
				}
				if c.SingleSoftCap != before.SingleSoftCap {
					t.Errorf("config changed on error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, c)
		})
	}

	c := Default()
	if err := LoadOverrides(filepath.Join(t.TempDir(), "missing.json"), &c); err == nil {
		t.Error("missing file: no error")
	}
}
