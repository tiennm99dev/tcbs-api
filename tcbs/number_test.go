package tcbs

import (
	"encoding/json"
	"testing"
)

func TestNumberUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		in      string
		want    float64
		wantErr bool
	}{
		{name: "number", in: `1000`, want: 1000},
		{name: "numeric string", in: `"1000"`, want: 1000},
		{name: "scientific string", in: `"0E-10"`, want: 0},
		{name: "null", in: `null`, want: 0},
		{name: "empty string", in: `""`, want: 0},
		{name: "scientific number", in: `1.5e3`, want: 1500},
		{name: "invalid string", in: `"abc"`, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var n Number
			err := json.Unmarshal([]byte(tt.in), &n)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error for %s, got %v", tt.in, n)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if n.Float() != tt.want {
				t.Errorf("expected %v, got %v", tt.want, n.Float())
			}
		})
	}
}

func TestNumberInStruct(t *testing.T) {
	var v struct {
		A Number `json:"a"`
		B Number `json:"b"`
	}
	if err := json.Unmarshal([]byte(`{"a":"12.5","b":null}`), &v); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.A != 12.5 || v.B != 0 {
		t.Errorf("unexpected values: %+v", v)
	}
}
