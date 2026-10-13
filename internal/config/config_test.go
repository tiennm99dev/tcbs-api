package config

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func writeEnv(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func envFunc(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadDotEnv(t *testing.T) {
	path := writeEnv(t, `# a comment
TCBS_API_KEY="dq-key with spaces"
export TCBS_CUSTODY_CODE='105C000001'
PLAIN=value # trailing comment
EMPTY=
  SPACED = padded
HASH="a#b"
exported=1
`)
	got, err := LoadDotEnv(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{
		"TCBS_API_KEY":      "dq-key with spaces",
		"TCBS_CUSTODY_CODE": "105C000001",
		"PLAIN":             "value",
		"EMPTY":             "",
		"SPACED":            "padded",
		"HASH":              "a#b",
		"exported":          "1",
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %#v, want %#v", got, want)
	}
}

func TestLoadDotEnv_BadLines(t *testing.T) {
	cases := map[string]string{
		"no equals":         "GOOD=1\nTCBS_API_KEY sk-secret-value\n",
		"bad key":           "GOOD=1\n1BAD=sk-secret-value\n",
		"unterminated":      "GOOD=1\nKEY=\"sk-secret-value\n",
		"text after quotes": "GOOD=1\nKEY=\"a\" sk-secret-value\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := LoadDotEnv(writeEnv(t, content))
			if err == nil {
				t.Fatal("expected an error")
			}
			if !strings.Contains(err.Error(), "line 2") {
				t.Errorf("error should name line 2: %v", err)
			}
			if strings.Contains(err.Error(), "sk-secret-value") {
				t.Errorf("error leaks the value: %v", err)
			}
		})
	}
}

func TestParse_Precedence(t *testing.T) {
	envFile := writeEnv(t, "TCBS_API_KEY=file-key\nTCBS_CUSTODY_CODE=105CFILE0001\n")

	t.Run("dotenv fallback", func(t *testing.T) {
		c, err := Parse([]string{"--env-file", envFile}, envFunc(nil))
		if err != nil {
			t.Fatal(err)
		}
		if c.APIKey != "file-key" || c.CustodyCode != "105CFILE0001" {
			t.Errorf("got key=%q custody=%q", c.APIKey, c.CustodyCode)
		}
	})
	t.Run("process env beats dotenv", func(t *testing.T) {
		c, err := Parse([]string{"--env-file", envFile}, envFunc(map[string]string{
			EnvAPIKey: "env-key", EnvCustodyCode: "105cenv00001",
		}))
		if err != nil {
			t.Fatal(err)
		}
		if c.APIKey != "env-key" || c.CustodyCode != "105CENV00001" {
			t.Errorf("got key=%q custody=%q", c.APIKey, c.CustodyCode)
		}
	})
	t.Run("flag beats process env", func(t *testing.T) {
		c, err := Parse([]string{"--env-file", envFile, "--custody", "105CFLAG0001"}, envFunc(map[string]string{
			EnvCustodyCode: "105CENV00001",
		}))
		if err != nil {
			t.Fatal(err)
		}
		if c.CustodyCode != "105CFLAG0001" {
			t.Errorf("got custody=%q", c.CustodyCode)
		}
	})
}

func TestParse_Defaults(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "absent.env")
	// The default .env path may be missing; chdir so no real .env is read.
	t.Chdir(dir)
	c, err := Parse(nil, envFunc(map[string]string{EnvAPIKey: "k", EnvCustodyCode: "105C000001"}))
	if err != nil {
		t.Fatalf("missing default .env should be fine: %v", err)
	}
	if c.Candidates != 10 || c.Out != "-" || c.CacheDir != ".cache/stock-advisor" ||
		c.FeeRate != 0.001 || c.Timeout != 20*time.Second || c.UniverseGroup() != UniverseVN100 ||
		c.Accounts != nil || c.Refresh || c.IncludeBankCash || c.Verbose {
		t.Errorf("unexpected defaults: %v", c)
	}
	if _, err := Parse([]string{"--env-file", missing}, envFunc(map[string]string{EnvAPIKey: "k", EnvCustodyCode: "c"})); err == nil {
		t.Error("an explicit missing --env-file should fail")
	}
}

func TestParse_Lists(t *testing.T) {
	c, err := Parse([]string{
		"--offline-holdings", "h.json", "--env-file", writeEnv(t, ""),
		"--accounts", " 0001N, 0001G ,,0001N", "--universe", "fpt, vcb,FPT",
	}, envFunc(nil))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(c.Accounts, []string{"0001N", "0001G"}) {
		t.Errorf("accounts: %v", c.Accounts)
	}
	if !reflect.DeepEqual(c.Universe, []string{"FPT", "VCB"}) || c.UniverseGroup() != "" {
		t.Errorf("universe: %v", c.Universe)
	}
}

func TestParse_Errors(t *testing.T) {
	envFile := writeEnv(t, "")
	cases := map[string][]string{
		"missing key":       {"--env-file", envFile, "--custody", "105C000001"},
		"missing custody":   nil,
		"bad candidates":    {"--env-file", envFile, "--offline-holdings", "h", "--candidates", "-1"},
		"bad fee":           {"--env-file", envFile, "--offline-holdings", "h", "--fee-rate", "0.5"},
		"bad timeout":       {"--env-file", envFile, "--offline-holdings", "h", "--timeout", "0s"},
		"unknown flag":      {"--env-file", envFile, "--nope"},
		"positional":        {"--env-file", envFile, "--offline-holdings", "h", "extra"},
		"empty universe":    {"--env-file", envFile, "--offline-holdings", "h", "--universe", " , "},
		"bad dotenv syntax": {"--env-file", writeEnv(t, "oops\n"), "--offline-holdings", "h"},
	}
	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			env := map[string]string{}
			if name == "missing custody" {
				args = []string{"--env-file", envFile}
				env[EnvAPIKey] = "k"
			}
			if _, err := Parse(args, envFunc(env)); err == nil {
				t.Error("expected an error")
			}
		})
	}
	if _, err := Parse([]string{"-h"}, envFunc(nil)); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h: expected flag.ErrHelp, got %v", err)
	}
}

func TestConfigString_RedactsAPIKey(t *testing.T) {
	c := Config{APIKey: "sk-test-SECRET", CustodyCode: "105C000001"}
	for _, s := range []string{c.String(), fmt.Sprint(c), fmt.Sprintf("%v", c), fmt.Sprintf("%+v", c), fmt.Sprintf("%#v", c)} {
		if strings.Contains(s, "SECRET") {
			t.Errorf("API key leaked: %s", s)
		}
		if !strings.Contains(s, "APIKey:***") {
			t.Errorf("expected redaction marker: %s", s)
		}
	}
}

func TestPrintUsage(t *testing.T) {
	var b strings.Builder
	PrintUsage(&b)
	for _, want := range []string{"-offline-holdings", "-include-bank-cash", EnvAPIKey} {
		if !strings.Contains(b.String(), want) {
			t.Errorf("usage lacks %q", want)
		}
	}
}
