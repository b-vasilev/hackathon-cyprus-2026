package generation

import (
	"context"
	"errors"
	"testing"
)

func TestGenerateQuizAfterPassport(t *testing.T) {
	for _, test := range []struct {
		name    string
		enabled bool
		fail    bool
	}{
		{name: "disabled"},
		{name: "enabled", enabled: true},
		{name: "quiz failure preserves passport", enabled: true, fail: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newRetryFixture(t, test.enabled)
			if f.run.GenerateQuiz != test.enabled {
				t.Fatal("quiz choice was not saved")
			}
			var called bool
			f.service.StartQuiz = func(ctx context.Context, revisionID string) error {
				called = true
				revision, err := f.service.Revision(ctx, revisionID)
				if err != nil || revision.RunID != f.run.ID {
					t.Fatal("quiz started without the saved passport", err)
				}
				run, err := f.service.GetRun(ctx, f.run.ID)
				if err != nil || run.Status == "succeeded" {
					t.Fatal("run finished before quiz started", err)
				}
				if test.fail {
					return errors.New("connection unavailable")
				}
				return nil
			}
			if err := f.service.process(t.Context(), f.run); err != nil {
				t.Fatal(err)
			}
			if called != test.enabled {
				t.Fatal("quiz did not follow the saved choice")
			}
			run, err := f.service.GetRun(t.Context(), f.run.ID)
			if err != nil || run.Status != "succeeded" {
				t.Fatal("passport did not succeed", err)
			}
			if test.fail {
				found := false
				for _, event := range run.Events {
					found = found || event.Type == "quiz_generation_failed"
				}
				if !found {
					t.Fatal("quiz failure was not recorded")
				}
			}
		})
	}
}

func TestRetryCanChangeQuizChoice(t *testing.T) {
	for _, initial := range []bool{false, true} {
		f := newRetryFixture(t, initial)
		f.failRun(t)
		input := f.input()
		input.GenerateQuiz = !initial
		run, err := f.service.Start(t.Context(), f.run.ProductID, input)
		if err != nil || run.GenerateQuiz != !initial {
			t.Fatal("retry did not save the changed quiz choice", err)
		}
	}
}
