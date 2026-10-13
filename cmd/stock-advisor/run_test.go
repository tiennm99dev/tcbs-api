package main

import (
	"context"
	"io/fs"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/miti99dev/stock-advisor/internal/research"
)

const offlineHoldings = "../../testdata/holdings.json"

// e2eUniverse mixes held symbols, scoreable candidates, a bank, a KBS-only
// symbol and an unknown one.
const e2eUniverse = "ACB,DGC,FPT,GAS,MWG,PNJ,ZZZ"

func noEnv(string) string { return "" }

func secretEnv(k string) string {
	if k == "TCBS_API_KEY" {
		return testAPIKey
	}
	return ""
}

func emptyEnvFile(t *testing.T) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "empty.env")
	if err := os.WriteFile(p, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

// normalise replaces the random fixture-server address in the output.
func normalise(s, url string) string {
	return strings.ReplaceAll(s, strings.TrimPrefix(url, "http://"), "fake.test")
}

func TestRun_OfflineGolden(t *testing.T) {
	pub := newResearchServer(t)
	dir := t.TempDir()
	out := filepath.Join(dir, "out", "report.md")
	saved := filepath.Join(dir, "saved.json")
	var stdout, stderr strings.Builder
	code := run(context.Background(), testDeps("", pub.URL), []string{
		"--env-file", emptyEnvFile(t),
		"--offline-holdings", offlineHoldings,
		"--universe", e2eUniverse,
		"--cache-dir", filepath.Join(dir, "cache"),
		"--out", out, "--save-holdings", saved, "-v",
	}, noEnv, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Errorf("stdout should be empty when --out is a file: %q", stdout.String())
	}
	for _, p := range []string{out, saved} {
		info, err := os.Stat(p)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0o600 {
			t.Errorf("%s mode %o, want 600", filepath.Base(p), perm)
		}
	}
	if info, err := os.Stat(filepath.Join(dir, "cache")); err != nil || info.Mode().Perm() != 0o700 {
		t.Errorf("cache dir: %v, %v", info, err)
	}

	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	got := normalise(string(b), pub.URL)
	golden := filepath.Join("testdata", "report.golden.md")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("%v (run with UPDATE_GOLDEN=1 to create it)", err)
	}
	if got != string(want) {
		t.Errorf("report differs from %s (UPDATE_GOLDEN=1 rewrites it):\n%s", golden, got)
	}

	for _, acct := range []string{"0000001N", "0000001G", "105C"} {
		if strings.Contains(got, acct) {
			t.Errorf("report leaks %q", acct)
		}
	}
}

// TestRun_OfflineCached runs twice on one cache dir; the second run must
// produce the same report from the cache alone.
func TestRun_OfflineCached(t *testing.T) {
	pub := newResearchServer(t)
	cache := filepath.Join(t.TempDir(), "cache")
	args := []string{"--env-file", emptyEnvFile(t), "--offline-holdings", offlineHoldings,
		"--universe", e2eUniverse, "--cache-dir", cache}

	var first, stderr strings.Builder
	if code := run(context.Background(), testDeps("", pub.URL), args, noEnv, strings.NewReader(""), &first, &stderr); code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	pub.Close()
	var second strings.Builder
	if code := run(context.Background(), testDeps("", pub.URL), args, noEnv, strings.NewReader(""), &second, &stderr); code != exitOK {
		t.Fatalf("cached run: exit %d, stderr: %s", code, stderr.String())
	}
	for _, sym := range []string{"FPT", "VCB", "REE"} {
		if !strings.Contains(second.String(), "### "+sym+" - ") {
			t.Errorf("cached report lacks the %s research block", sym)
		}
	}
	if !strings.Contains(second.String(), "| 1 | ") {
		t.Error("cached report has no candidates")
	}
}

// TestRun_ResearchDown checks that dead public sources only degrade the
// report: the run still succeeds and says what is missing.
func TestRun_ResearchDown(t *testing.T) {
	dead := httptest.NewServer(nil)
	dead.Close()
	var stdout, stderr strings.Builder
	code := run(context.Background(), testDeps("", dead.URL), []string{
		"--env-file", emptyEnvFile(t), "--offline-holdings", offlineHoldings,
		"--universe", "VN30", "--cache-dir", "",
	}, noEnv, strings.NewReader(""), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	report := stdout.String()
	for _, want := range []string{"Insufficient data", "VN-Index bars unavailable", "no daily bars from any source", "(ref)", "not financial advice"} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q", want)
		}
	}
}

func TestRun_Online(t *testing.T) {
	pub := newResearchServer(t)
	fake := &fakeTCBS{}
	tcbsSrv := httptest.NewServer(fake)
	t.Cleanup(tcbsSrv.Close)

	var stdout, stderr strings.Builder
	code := run(context.Background(), testDeps(tcbsSrv.URL, pub.URL), []string{
		"--env-file", emptyEnvFile(t), "--custody", testCustody,
		"--universe", e2eUniverse, "--cache-dir", filepath.Join(t.TempDir(), "cache"),
	}, secretEnv, strings.NewReader(testOTP+"\n"), &stdout, &stderr)
	if code != exitOK {
		t.Fatalf("exit %d, stderr: %s", code, stderr.String())
	}
	if !fake.called("/tartarus/v1/tickerCommons") {
		t.Error("held quotes were not read from TCBS")
	}
	report := stdout.String()
	for _, want := range []string{"TCBS OpenAPI", "TCBS tickerCommons", "(ref)", "### FPT - FPT Corporation", "OLD"} {
		if !strings.Contains(report, want) {
			t.Errorf("report lacks %q", want)
		}
	}
	if strings.Contains(report, "TCBS quotes unavailable") {
		t.Error("TCBS quotes failed to decode; the report fell back to the public close")
	}
}

// TestRun_NoSecretLeak scans every output of the online path for the API
// key and the token: on success, when sign-in fails with 400 and when the
// account calls fail with 500.
func TestRun_NoSecretLeak(t *testing.T) {
	cases := []struct {
		name string
		fake *fakeTCBS
		want int
	}{
		{"success", &fakeTCBS{}, exitOK},
		{"token 400", &fakeTCBS{failToken: true}, exitFatal},
		{"accounts 500", &fakeTCBS{failAccounts: true}, exitFatal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			pub := newResearchServer(t)
			tcbsSrv := httptest.NewServer(c.fake)
			t.Cleanup(tcbsSrv.Close)
			dir := t.TempDir()
			out := filepath.Join(dir, "report.md")
			saved := filepath.Join(dir, "holdings.json")
			cache := filepath.Join(dir, "cache")
			raw := filepath.Join(dir, "raw")

			var stdout, stderr strings.Builder
			code := run(context.Background(), testDeps(tcbsSrv.URL, pub.URL), []string{
				"--env-file", emptyEnvFile(t), "--custody", testCustody, "--universe", e2eUniverse,
				"--out", out, "--save-holdings", saved, "--cache-dir", cache, "--dump-raw", raw, "-v",
			}, secretEnv, strings.NewReader(testOTP+"\n"), &stdout, &stderr)
			if code != c.want {
				t.Fatalf("exit %d, want %d; stderr: %s", code, c.want, stderr.String())
			}
			if c.want == exitOK {
				for _, p := range []string{out, saved} {
					if _, err := os.Stat(p); err != nil {
						t.Errorf("missing %s: %v", filepath.Base(p), err)
					}
				}
				if entries, _ := os.ReadDir(raw); len(entries) == 0 {
					t.Error("--dump-raw wrote nothing")
				}
			}

			scan := func(name, content string) {
				if strings.Contains(content, "SECRET") {
					t.Errorf("%s contains a secret", name)
				}
			}
			scan("stdout", stdout.String())
			scan("stderr", stderr.String())
			if strings.Contains(stderr.String(), testOTP) {
				t.Error("stderr contains the OTP")
			}
			for _, root := range []string{out, saved, cache, raw} {
				_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
					if err != nil || d.IsDir() {
						return nil
					}
					b, rerr := os.ReadFile(p)
					if rerr != nil {
						t.Errorf("read %s: %v", p, rerr)
						return nil
					}
					scan(p, string(b))
					return nil
				})
			}
		})
	}
}

func TestRun_UsageAndErrors(t *testing.T) {
	dir := t.TempDir()
	empty := emptyEnvFile(t)
	badRules := filepath.Join(dir, "rules.json")
	if err := os.WriteFile(badRules, []byte(`{"NoSuchField": 1}`), 0o600); err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		args []string
		want int
	}{
		{"help", []string{"-h"}, exitOK},
		{"unknown flag", []string{"--env-file", empty, "--bogus"}, exitUsage},
		{"no api key", []string{"--env-file", empty, "--custody", testCustody}, exitUsage},
		{"bad rules file", []string{"--env-file", empty, "--offline-holdings", offlineHoldings, "--rules", badRules}, exitUsage},
		{"missing offline file", []string{"--env-file", empty, "--offline-holdings", filepath.Join(dir, "absent.json"), "--cache-dir", ""}, exitFatal},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var stdout, stderr strings.Builder
			if got := run(context.Background(), testDeps("", ""), c.args, noEnv, strings.NewReader(""), &stdout, &stderr); got != c.want {
				t.Errorf("exit %d, want %d; stderr: %s", got, c.want, stderr.String())
			}
		})
	}
}

func TestPickPeers(t *testing.T) {
	core := []string{"FPT", "VCB"}
	many := make([]string, 0, maxPeers+5)
	for i := 0; i < maxPeers+5; i++ {
		many = append(many, "P"+strings.Repeat("X", i))
	}
	peers, capped := pickPeers(core, map[string]research.Profile{
		"FPT": {Peers: []string{"CMG", "FPT", "VCB"}},
		"VCB": {Peers: many},
	})
	if !capped || len(peers) != maxPeers || peers[0] != "CMG" {
		t.Errorf("peers %v (len %d), capped %v", peers[:3], len(peers), capped)
	}
}
