package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/b-vasilev/hackathon-cyprus-2026/backend/internal/generation"
)

// generationEvents streams consistent snapshots with only events after the cursor.
// SQLite is the event source, so reconnects also work after a server restart.
func (f *Factory) generationEvents(w http.ResponseWriter, r *http.Request) {
	cursor := int64(0)
	value := r.Header.Get("Last-Event-ID")
	if value == "" {
		value = r.URL.Query().Get("after")
	}
	if value != "" {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil || parsed < 0 {
			problem(w, http.StatusBadRequest, "invalid_cursor", "The event cursor must be a non-negative integer.")
			return
		}
		cursor = parsed
	}
	run, err := f.Service.GetRun(r.Context(), r.PathValue("id"))
	if err != nil {
		respond(w, r, nil, err)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache, no-transform")
	w.Header().Set("X-Accel-Buffering", "no")
	controller := http.NewResponseController(w)
	if checkErr := streamRun(w, controller, run, &cursor, "snapshot"); checkErr != nil {
		return
	}
	updates := time.NewTicker(time.Second)
	defer updates.Stop()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()
	for run.Status == "queued" || run.Status == "running" {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if checkErr := streamFrame(w, controller, ": heartbeat\n\n"); checkErr != nil {
				return
			}
		case <-updates.C:
			previousStatus := run.Status
			run, err = f.Service.GetRun(r.Context(), run.ID)
			if err != nil {
				return // EventSource reconnects and reads the durable cursor again.
			}
			if latestEventID(run) > cursor || run.Status != previousStatus {
				if err := streamRun(w, controller, run, &cursor, "update"); err != nil {
					return
				}
			}
		}
	}
}

func latestEventID(run generation.Run) int64 {
	if len(run.Events) == 0 {
		return 0
	}
	return run.Events[len(run.Events)-1].ID
}

func streamRun(w http.ResponseWriter, controller *http.ResponseController, run generation.Run, cursor *int64, kind string) error {
	latest := latestEventID(run)
	if *cursor > latest {
		*cursor = 0
	}
	first := 0
	for first < len(run.Events) && run.Events[first].ID <= *cursor {
		first++
	}
	run.Events = run.Events[first:]
	data, err := json.Marshal(run)
	if err != nil {
		return fmt.Errorf("encode generation stream: %w", err)
	}
	if err := streamFrame(w, controller, fmt.Sprintf("retry: 2000\nid: %d\nevent: %s\ndata: %s\n\n", latest, kind, data)); err != nil {
		return err
	}
	*cursor = latest
	return nil
}

func streamFrame(w http.ResponseWriter, controller *http.ResponseController, frame string) error {
	// Refresh the normal server write deadline for this long-lived response only.
	if err := controller.SetWriteDeadline(time.Now().Add(20 * time.Second)); err != nil && !errors.Is(err, http.ErrNotSupported) {
		return fmt.Errorf("set stream deadline: %w", err)
	}
	// #nosec G705 -- Frames contain JSON-encoded SSE data served as text/event-stream, never HTML.
	if _, err := fmt.Fprint(w, frame); err != nil {
		return fmt.Errorf("write generation stream: %w", err)
	}
	if err := controller.Flush(); err != nil {
		return fmt.Errorf("flush generation stream: %w", err)
	}
	return nil
}
