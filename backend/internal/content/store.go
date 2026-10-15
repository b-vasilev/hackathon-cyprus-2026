// Package content stores immutable passport payloads outside the metadata database.
package content

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Store confines all file access to the application's private content directory.
type Store struct{ Dir string }

// WriteJSON saves a readable JSON artifact and returns its checksum.
func (s *Store) WriteJSON(path string, value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode content: %w", err)
	}
	return s.Write(path, append(data, '\n'))
}

// SaveCheckpoint atomically replaces mutable, in-progress state. Accepted
// artifacts still use WriteJSON and retain its immutable-file contract.
func (s *Store) SaveCheckpoint(path string, value any) error {
	if !validPath(path) {
		return errors.New("invalid checkpoint path")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	root, err := os.OpenRoot(s.Dir)
	if err != nil {
		return err
	}
	defer root.Close()
	if err := root.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary := filepath.Join(filepath.Dir(path), ".checkpoint-"+rand.Text())
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(temporary) }()
	err = json.NewEncoder(file).Encode(value)
	if err == nil {
		err = file.Sync()
	}
	if err := errors.Join(err, file.Close()); err != nil {
		return err
	}
	if err := root.Rename(temporary, path); err != nil {
		return err
	}
	directory, err := root.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

// Write publishes a complete immutable file before its reference enters SQLite.
func (s *Store) Write(path string, data []byte) (string, error) {
	if !validPath(path) {
		return "", errors.New("invalid content path")
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return "", fmt.Errorf("create content directory: %w", err)
	}
	root, err := os.OpenRoot(s.Dir)
	if err != nil {
		return "", fmt.Errorf("open content directory: %w", err)
	}
	defer root.Close()
	if mkdirErr := root.MkdirAll(filepath.Dir(path), 0o700); mkdirErr != nil {
		return "", fmt.Errorf("create content folder: %w", mkdirErr)
	}
	temporary := ".writing-" + rand.Text()
	file, err := root.OpenFile(temporary, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", fmt.Errorf("create content file: %w", err)
	}
	defer func() { _ = root.Remove(temporary) }()
	_, writeErr := file.Write(data)
	if writeErr == nil {
		writeErr = file.Sync()
	}
	closeErr := file.Close()
	if saveErr := errors.Join(writeErr, closeErr); saveErr != nil {
		return "", fmt.Errorf("write content file: %w", saveErr)
	}
	if linkErr := root.Link(temporary, path); linkErr != nil {
		if !errors.Is(linkErr, os.ErrExist) {
			return "", fmt.Errorf("publish content file: %w", linkErr)
		}
		existing, readErr := root.ReadFile(path)
		if readErr != nil || !bytes.Equal(existing, data) {
			return "", errors.New("an immutable content file already exists with different data")
		}
	}
	directory, err := root.Open(filepath.Dir(path))
	if err != nil {
		return "", fmt.Errorf("open content folder: %w", err)
	}
	defer directory.Close()
	if err := directory.Sync(); err != nil {
		return "", fmt.Errorf("sync content folder: %w", err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

// Read verifies that stored bytes still match the database reference.
func (s *Store) Read(path, checksum string) ([]byte, error) {
	if !validPath(path) || len(checksum) != sha256.Size*2 {
		return nil, errors.New("invalid content reference")
	}
	root, err := os.OpenRoot(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("open content directory: %w", err)
	}
	defer root.Close()
	data, err := root.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read content file: %w", err)
	}
	hash := sha256.Sum256(data)
	if hex.EncodeToString(hash[:]) != checksum {
		return nil, errors.New("content checksum mismatch")
	}
	return data, nil
}

// ReadJSON decodes a verified artifact using the existing application contract.
func (s *Store) ReadJSON(path, checksum string, value any) error {
	data, err := s.Read(path, checksum)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("decode content file: %w", err)
	}
	return nil
}

func validPath(path string) bool {
	return filepath.IsLocal(path) && path != "." && filepath.Clean(path) == path
}

// RemoveAll removes an artifact folder within the private content directory.
func (s *Store) RemoveAll(path string) error {
	if !validPath(path) {
		return errors.New("invalid content path")
	}
	root, err := os.OpenRoot(s.Dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("open content directory: %w", err)
	}
	defer root.Close()
	if err := root.RemoveAll(path); err != nil {
		return fmt.Errorf("remove content folder: %w", err)
	}
	return nil
}
