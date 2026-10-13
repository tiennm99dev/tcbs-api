package tcbs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Number decodes a JSON number, a numeric string ("1000", "0E-10") or
// null/"" (as 0).
type Number float64

// UnmarshalJSON implements json.Unmarshaler.
func (n *Number) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*n = 0
		return nil
	}

	s := string(b)
	if b[0] == '"' {
		if err := json.Unmarshal(b, &s); err != nil {
			return fmt.Errorf("tcbs: decode number: %w", err)
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*n = 0
			return nil
		}
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return fmt.Errorf("tcbs: decode number %q: %w", s, err)
	}
	*n = Number(f)
	return nil
}

// Float returns the value as a float64.
func (n Number) Float() float64 {
	return float64(n)
}
