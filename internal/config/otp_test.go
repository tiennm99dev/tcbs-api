package config

import (
	"strings"
	"testing"
)

func TestPromptOTP_NonTTY(t *testing.T) {
	var out strings.Builder
	otp, err := PromptOTP(strings.NewReader("  123456 \n999999\n"), -1, &out)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if otp != "123456" {
		t.Errorf("got %q", otp)
	}
	if !strings.Contains(out.String(), "OTP") {
		t.Errorf("prompt missing: %q", out.String())
	}
	if strings.Contains(out.String(), "123456") {
		t.Errorf("prompt echoed the OTP: %q", out.String())
	}
}

func TestPromptOTP_NoTrailingNewline(t *testing.T) {
	otp, err := PromptOTP(strings.NewReader("654321"), -1, &strings.Builder{})
	if err != nil || otp != "654321" {
		t.Errorf("got %q, %v", otp, err)
	}
}

func TestPromptOTP_Invalid(t *testing.T) {
	for _, in := range []string{"", "\n", "12345\n", "1234567\n", "12a456\n", "12 456\n", "１２３４５６\n"} {
		otp, err := PromptOTP(strings.NewReader(in), -1, &strings.Builder{})
		if err == nil {
			t.Errorf("input %q: expected an error, got %q", in, otp)
			continue
		}
		if trimmed := strings.TrimSpace(in); trimmed != "" && strings.Contains(err.Error(), trimmed) {
			t.Errorf("input %q: error echoes the input: %v", in, err)
		}
	}
}
