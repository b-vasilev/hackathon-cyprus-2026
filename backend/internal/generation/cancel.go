package generation

import "context"

const canceledErrorCode = "cancelled" //nolint:misspell // Preserve the stored API code expected by existing runs and clients.

// Cancel records a terminal state before interrupting any in-flight work.
// Failed runs use a separate error code for cancellation to preserve the stored status contract.
// Repeating cancellation or canceling a finished run returns its current state.
func (s *Service) Cancel(ctx context.Context, id string) (Run, error) {
	if err := s.failRun(ctx, id, canceledErrorCode, "Generation canceled by the user."); err != nil {
		return Run{}, err
	}
	if cancel, ok := s.active.Load(id); ok {
		cancel.(context.CancelFunc)()
	}
	return s.GetRun(ctx, id)
}
