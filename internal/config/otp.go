package config

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strings"

	"golang.org/x/term"
)

// otpLen is the length of a TCBS iOTP code.
const otpLen = 6

// errBadOTP never echoes the input back.
var errBadOTP = errors.New("otp: expected 6 digits")

// PromptOTP writes the prompt to out (stderr in the CLI). When fd is a
// terminal it reads with term.ReadPassword (no echo); otherwise it reads one
// line from in. It trims the input and requires 6 digits. The OTP is never
// stored or echoed.
func PromptOTP(in io.Reader, fd int, out io.Writer) (string, error) {
	fmt.Fprint(out, "TCBS iOTP (6 digits): ")
	var raw string
	if fd >= 0 && term.IsTerminal(fd) {
		b, err := term.ReadPassword(fd)
		fmt.Fprintln(out)
		if err != nil {
			return "", fmt.Errorf("otp: read terminal: %w", err)
		}
		raw = string(b)
	} else {
		line, err := bufio.NewReader(in).ReadString('\n')
		if err != nil && (!errors.Is(err, io.EOF) || line == "") {
			if errors.Is(err, io.EOF) {
				return "", errors.New("otp: no input")
			}
			return "", fmt.Errorf("otp: read input: %w", err)
		}
		raw = line
	}
	otp := strings.TrimSpace(raw)
	if len(otp) != otpLen {
		return "", errBadOTP
	}
	for _, r := range otp {
		if r < '0' || r > '9' {
			return "", errBadOTP
		}
	}
	return otp, nil
}
