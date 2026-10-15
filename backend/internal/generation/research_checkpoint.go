package generation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/research"
)

type savedResearchCheckpoint struct {
	research.Checkpoint
	Path string
	Hash string
}

func researchArtifactPath(runID string, attempt int) string {
	if attempt == 0 {
		return "runs/" + runID + "/research/findings.json"
	}
	return fmt.Sprintf("runs/%s/research/attempt-%d/findings.json", runID, attempt+1)
}

// Each retry gets an immutable artifact; earlier evidence remains available.
func (s *Service) researchCheckpoint(ctx context.Context, runID string) (*savedResearchCheckpoint, string, error) {
	var attempt int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM generation_events WHERE run_id=? AND type='run_resumed'", runID).Scan(&attempt); err != nil {
		return nil, "", fmt.Errorf("read research attempt: %w", err)
	}
	outputPath := researchArtifactPath(runID, attempt)
	if attempt == 0 {
		return nil, outputPath, nil
	}
	root, err := os.OpenRoot(s.Content.Dir)
	if err != nil {
		return nil, "", fmt.Errorf("open saved research: %w", err)
	}
	defer root.Close()
	for previous := attempt - 1; previous >= 0; previous-- {
		checkpointPath := researchArtifactPath(runID, previous)
		file, err := root.Open(checkpointPath)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, "", fmt.Errorf("open research checkpoint: %w", err)
		}
		const maxCheckpointBytes = 32 << 20
		data, readErr := io.ReadAll(io.LimitReader(file, maxCheckpointBytes+1))
		closeErr := file.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return nil, "", fmt.Errorf("read research checkpoint: %w", err)
		}
		if len(data) > maxCheckpointBytes {
			return nil, "", errors.New("saved research checkpoint exceeds the size limit")
		}
		var checkpoint research.Checkpoint
		if err := json.Unmarshal(data, &checkpoint); err != nil {
			return nil, "", fmt.Errorf("decode research checkpoint: %w", err)
		}
		hash := sha256.Sum256(data)
		return &savedResearchCheckpoint{Checkpoint: checkpoint, Path: checkpointPath, Hash: hex.EncodeToString(hash[:])}, outputPath, nil
	}
	return nil, outputPath, nil
}
