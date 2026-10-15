package generation

import (
	"path/filepath"
	"testing"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/content"
	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/db"
)

func TestResearchCheckpointRetainsEarlierAttempts(t *testing.T) {
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "checkpoints.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, checkErr := database.ExecContext(ctx, "CREATE TABLE generation_events(run_id TEXT, type TEXT)"); checkErr != nil {
		t.Fatal(checkErr)
	}
	s := &Service{DB: database, Content: &content.Store{Dir: t.TempDir()}}
	const runID = "checkpoint-run"
	legacyPath := researchArtifactPath(runID, 0)
	checkpoint, outputPath, err := s.researchCheckpoint(ctx, runID)
	if err != nil || checkpoint != nil || outputPath != legacyPath {
		t.Fatalf("first attempt: checkpoint=%v path=%s err=%v", checkpoint, outputPath, err)
	}
	if _, writeErr := s.Content.WriteJSON(legacyPath, map[string]string{"notes": "first findings"}); writeErr != nil {
		t.Fatal(writeErr)
	}
	for range 2 {
		if _, insertErr := database.ExecContext(ctx, "INSERT INTO generation_events VALUES(?, 'run_resumed')", runID); insertErr != nil {
			t.Fatal(insertErr)
		}
	}
	// The intervening attempt stopped before research: fall back to the last
	// available checkpoint while preserving a fresh destination for this retry.
	checkpoint, outputPath, err = s.researchCheckpoint(ctx, runID)
	if err != nil || checkpoint == nil || checkpoint.Notes != "first findings" || outputPath != researchArtifactPath(runID, 2) {
		t.Fatalf("retry: checkpoint=%v path=%s err=%v", checkpoint, outputPath, err)
	}
	if _, err := s.Content.WriteJSON(researchArtifactPath(runID, 1), map[string]string{"notes": "second findings"}); err != nil {
		t.Fatal(err)
	}
	checkpoint, _, err = s.researchCheckpoint(ctx, runID)
	if err != nil || checkpoint == nil || checkpoint.Notes != "second findings" {
		t.Fatalf("latest available checkpoint=%v err=%v", checkpoint, err)
	}
	if _, err := s.Content.Write(researchArtifactPath(runID, 1), []byte("invalid JSON")); err == nil {
		t.Fatal("saved research was overwritten")
	}
}

func TestResearchCheckpointRejectsInvalidSavedJSON(t *testing.T) {
	ctx := t.Context()
	database, err := db.Open(ctx, filepath.Join(t.TempDir(), "invalid.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if _, err := database.ExecContext(ctx, "CREATE TABLE generation_events(run_id TEXT, type TEXT)"); err != nil {
		t.Fatal(err)
	}
	s := &Service{DB: database, Content: &content.Store{Dir: t.TempDir()}}
	if _, err := database.ExecContext(ctx, "INSERT INTO generation_events VALUES('invalid', 'run_resumed')"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Content.Write(researchArtifactPath("invalid", 0), []byte("invalid JSON")); err != nil {
		t.Fatal(err)
	}
	if _, _, err := s.researchCheckpoint(ctx, "invalid"); err == nil {
		t.Fatal("invalid checkpoint was silently ignored")
	}
}
