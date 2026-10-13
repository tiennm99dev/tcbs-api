package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
)

// LoadDotEnv parses KEY=VALUE lines. It accepts a leading `export`, # comments,
// and single or double quotes. It never overrides keys already set in the
// process env: callers look up the process env first and use the returned map
// only as a fallback. Errors name the line number, never the value.
func LoadDotEnv(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return parseDotEnv(f, path)
}

// parseDotEnv reads dotenv lines from r; name only labels errors.
func parseDotEnv(r io.Reader, name string) (map[string]string, error) {
	out := make(map[string]string)
	sc := bufio.NewScanner(r)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		line := strings.TrimSpace(sc.Text())
		if lineNo == 1 {
			line = strings.TrimPrefix(line, "\uFEFF")
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "export"); ok && rest != "" && (rest[0] == ' ' || rest[0] == '\t') {
			line = strings.TrimSpace(rest)
		}
		key, raw, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("%s: line %d: expected KEY=VALUE", name, lineNo)
		}
		key = strings.TrimSpace(key)
		if !validKey(key) {
			return nil, fmt.Errorf("%s: line %d: invalid key", name, lineNo)
		}
		val, err := dotEnvValue(strings.TrimSpace(raw))
		if err != nil {
			return nil, fmt.Errorf("%s: line %d: %w", name, lineNo, err)
		}
		out[key] = val
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", name, err)
	}
	return out, nil
}

// dotEnvValue unquotes a value. Unquoted values lose a trailing " # comment".
func dotEnvValue(raw string) (string, error) {
	if raw == "" {
		return "", nil
	}
	if q := raw[0]; q == '"' || q == '\'' {
		end := strings.IndexByte(raw[1:], q)
		if end < 0 {
			return "", errors.New("unterminated quote")
		}
		rest := strings.TrimSpace(raw[end+2:])
		if rest != "" && !strings.HasPrefix(rest, "#") {
			return "", errors.New("unexpected text after closing quote")
		}
		return raw[1 : end+1], nil
	}
	if i := strings.Index(raw, " #"); i >= 0 {
		raw = raw[:i]
	}
	if i := strings.Index(raw, "\t#"); i >= 0 {
		raw = raw[:i]
	}
	return strings.TrimSpace(raw), nil
}

// validKey reports whether k looks like a shell variable name.
func validKey(k string) bool {
	if k == "" {
		return false
	}
	for i, r := range k {
		switch {
		case r == '_', r >= 'A' && r <= 'Z', r >= 'a' && r <= 'z':
		case r >= '0' && r <= '9' && i > 0:
		default:
			return false
		}
	}
	return true
}

// isNotExist reports whether err means the file is missing.
func isNotExist(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}
