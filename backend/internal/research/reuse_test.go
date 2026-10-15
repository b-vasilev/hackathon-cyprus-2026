package research

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func TestReuseAcceptedCheckpointKeepsPartialStopAndFrontierWithoutExternalWork(t *testing.T) {
	for _, status := range []string{researchComplete, "partial"} {
		t.Run(status, func(t *testing.T) {
			_, collection, checkpoint := fixtureCheckpoint(t, status)
			if status == "partial" {
				checkpoint.Ledger.StopReason = "Attempt turn budget reached (59/60); remaining obligations are preserved"
				checkpoint.Ledger.Frontier = append(checkpoint.Ledger.Frontier, "Specific earlier unresolved source boundary")
			}
			before := checkpoint.Notes
			beforeFrontier := slices.Clone(checkpoint.Ledger.Frontier)
			workDir := filepath.Join(t.TempDir(), "must-not-be-created")
			runner := Runner{WorkDir: workDir, RepomixBin: "/source-reader-must-not-run", MaxTurns: 1}
			result, err := runner.ReuseAcceptedCheckpoint(t.Context(), collection, checkpoint)
			if err != nil {
				t.Fatal(err)
			}
			if result.Ledger.Status != status || result.Ledger.StopReason != checkpoint.Ledger.StopReason || !reflect.DeepEqual(result.Ledger.Frontier, beforeFrontier) {
				t.Fatalf("accepted status/stop/frontier changed: %+v", result.Ledger)
			}
			assertReuseDidNoExternalWork(t, result, workDir)
			if result.Notes != before || checkpoint.Notes != before || !reflect.DeepEqual(checkpoint.Ledger.Frontier, beforeFrontier) {
				t.Fatal("accepted findings or original checkpoint were changed")
			}
			if strings.Contains(result.Context, checkpoint.Context) || !strings.Contains(result.Context, "3: func run() {") {
				t.Fatal("source context was trusted or clipped instead of rebuilt")
			}
			for _, match := range citationPattern.FindAllString(result.Notes, -1) {
				if !strings.Contains(result.Context, match) {
					t.Fatalf("original citation was lost: %s", match)
				}
			}
			if len(result.Collection.Documents) != len(collection.Documents) {
				t.Fatal("previously read source documents were lost")
			}
		})
	}
}

func TestReuseAcceptedCheckpointRejectsUnacceptedOrInconsistentEvidence(t *testing.T) {
	for _, kind := range []string{"empty_context", "in_progress", "missing_stop", "complete_with_frontier", "revision", "document", "range", "unread_citation", "uncited_role", "content_hash"} {
		t.Run(kind, func(t *testing.T) {
			_, collection, checkpoint := fixtureCheckpoint(t, "partial")
			switch kind {
			case "empty_context":
				checkpoint.Context = " \n"
			case "in_progress":
				checkpoint.Ledger.Status = "in_progress"
			case "missing_stop":
				checkpoint.Ledger.StopReason = ""
			case "complete_with_frontier":
				checkpoint.Ledger.Status = researchComplete
			case "revision":
				checkpoint.Snapshots[0].Revision = "changed-revision"
			case "document":
				checkpoint.Snapshots[0].Coverage.Files[0].ID = "changed-document"
			case "range":
				checkpoint.Snapshots[0].Coverage.Files[0].ReadRanges[0].EndLine = 10000
			case "unread_citation":
				checkpoint.Snapshots[0].Coverage.Files[0].ReadRanges = nil
			case "uncited_role":
				for i := range checkpoint.Ledger.Findings {
					checkpoint.Ledger.Findings[i].Notes = strings.ReplaceAll(checkpoint.Ledger.Findings[i].Notes, "[BBBBBBBBBBBBBBBBBBBBBBBBBB:L1-L2]", "")
				}
				checkpoint.Notes = strings.ReplaceAll(checkpoint.Notes, "[BBBBBBBBBBBBBBBBBBBBBBBBBB:L1-L2]", "")
			case "content_hash":
				hash := sha256.Sum256([]byte(collection.Documents[0].Content))
				collection.Documents[0].ContentHash = hex.EncodeToString(hash[:])
				collection.Documents[0].Content = strings.ReplaceAll(collection.Documents[0].Content, "package", "changed")
			}
			runner := Runner{}
			if _, err := runner.ReuseAcceptedCheckpoint(t.Context(), collection, checkpoint); err == nil {
				t.Fatal("invalid accepted checkpoint was reused")
			}
		})
	}
}

func TestReuseAcceptedCheckpointHonorsCancellation(t *testing.T) {
	_, collection, checkpoint := fixtureCheckpoint(t, researchComplete)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	runner := Runner{}
	if _, err := runner.ReuseAcceptedCheckpoint(ctx, collection, checkpoint); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled reuse did not stop: %v", err)
	}
}

func assertReuseDidNoExternalWork(t *testing.T, result Result, workDir string) {
	t.Helper()
	if result.Steps != 0 || result.ToolCalls != 0 || result.MCPReads != 0 || result.InputTokens != 0 || result.OutputTokens != 0 {
		t.Fatal("reuse performed new model or source tool work")
	}
	if _, err := os.Stat(workDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("reuse touched the filesystem workspace: %v", err)
	}
}
