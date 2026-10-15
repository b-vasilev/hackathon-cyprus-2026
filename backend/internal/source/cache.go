package source

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

func (c *Collector) packagePath(key string) string {
	hash := sha256.Sum256([]byte(key))
	return filepath.Join(c.WorkDir, "packages", hex.EncodeToString(hash[:])+".json")
}

func (c *Collector) loadPackage(key string) (map[string]string, bool) {
	path := c.packagePath(key)
	info, err := os.Stat(path)
	if err != nil || info.Size() > int64(c.MaxBytes)*3 { return nil, false }
	data, err := os.ReadFile(path) //nolint:gosec // Hash-derived app-owned cache path.
	var files map[string]string
	if err != nil || json.Unmarshal(data, &files) != nil { return nil, false }
	return files, len(files)>0
}

func (c *Collector) savePackage(key string, files map[string]string) error {
	path := c.packagePath(key)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil { return fmt.Errorf("create source cache: %w", err) }
	data, err := json.Marshal(files)
	if err != nil { return fmt.Errorf("encode source cache: %w", err) }
	tmp, err := os.CreateTemp(filepath.Dir(path), ".package-")
	if err != nil { return fmt.Errorf("create source cache file: %w", err) }
	defer func(){_ = os.Remove(tmp.Name())}()
	_, writeErr := tmp.Write(data)
	closeErr := tmp.Close()
	if writeErr != nil { return fmt.Errorf("write source cache: %w", writeErr) }
	if closeErr != nil { return fmt.Errorf("close source cache: %w", closeErr) }
	if err := os.Rename(tmp.Name(), path); err != nil { return fmt.Errorf("publish source cache: %w", err) }
	return nil
}
