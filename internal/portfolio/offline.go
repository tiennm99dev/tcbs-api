package portfolio

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
)

// FileSource reads a snapshot saved with Save (or written by hand).
type FileSource struct{ Path string }

// Snapshot decodes the snapshot JSON at Path.
func (f FileSource) Snapshot(ctx context.Context) (Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return Snapshot{}, err
	}
	b, err := os.ReadFile(f.Path)
	if err != nil {
		return Snapshot{}, fmt.Errorf("portfolio: read snapshot: %w", err)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var s Snapshot
	if err := dec.Decode(&s); err != nil {
		return Snapshot{}, fmt.Errorf("portfolio: decode snapshot %s: %w", f.Path, err)
	}
	if len(s.Accounts) == 0 {
		return Snapshot{}, fmt.Errorf("portfolio: snapshot %s has no accounts", f.Path)
	}
	return s, nil
}

// Save writes s as indented JSON with mode 0600. The custody code is masked.
func Save(path string, s Snapshot) error {
	s.Custody = MaskCustody(s.Custody)
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("portfolio: encode snapshot: %w", err)
	}
	if err := writeFile0600(path, append(b, '\n')); err != nil {
		return fmt.Errorf("portfolio: save snapshot: %w", err)
	}
	return nil
}

// writeFile0600 writes data to path and forces mode 0600, also when the file
// already existed with a wider mode.
func writeFile0600(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}
