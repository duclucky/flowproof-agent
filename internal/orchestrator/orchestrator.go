package orchestrator

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ducky/flowproof-agent/internal/model"
	"github.com/ducky/flowproof-agent/internal/store"
)

var (
	ErrInvalidRunState = errors.New("invalid run state")
	ErrTestNotFound    = errors.New("test not found")
)

type Driver interface {
	RunInitial(context.Context, model.TestDefinition) (model.BrowserObservation, error)
	Retry(context.Context, model.TestDefinition, string) (model.BrowserObservation, error)
}

type Service struct {
	store  store.Store
	driver Driver

	mu    sync.RWMutex
	tests map[string]model.TestDefinition
}

func New(st store.Store, driver Driver) *Service {
	return &Service{store: st, driver: driver, tests: make(map[string]model.TestDefinition)}
}

func (s *Service) CreateTest(req model.CreateTestRequest) (model.TestDefinition, error) {
	if strings.TrimSpace(req.TargetURL) == "" {
		return model.TestDefinition{}, fmt.Errorf("target URL is required")
	}
	if strings.TrimSpace(req.Objective) == "" {
		return model.TestDefinition{}, fmt.Errorf("objective is required")
	}
	tc := model.TestDefinition{
		ID:        newID("test"),
		TargetURL: strings.TrimSpace(req.TargetURL),
		Objective: strings.TrimSpace(req.Objective),
		CreatedAt: time.Now().UTC(),
	}
	s.mu.Lock()
	s.tests[tc.ID] = tc
	s.mu.Unlock()
	return tc, nil
}

func (s *Service) StartRun(ctx context.Context, testID string) (model.Run, error) {
	tc, ok := s.getTest(testID)
	if !ok {
		return model.Run{}, ErrTestNotFound
	}
	now := time.Now().UTC()
	run := model.Run{
		ID:        newID("run"),
		TestID:    tc.ID,
		TargetURL: tc.TargetURL,
		Objective: tc.Objective,
		Status:    model.RunQueued,
		MaxSteps:  8,
		Events: []model.RunEvent{{
			Seq: 1, At: now, Type: "queued", Message: "QA run queued",
		}},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.store.Create(run); err != nil {
		return model.Run{}, err
	}
	run, err := s.store.Update(run.ID, func(r *model.Run) error {
		r.Status = model.RunRunning
		r.UpdatedAt = time.Now().UTC()
		appendEvent(r, "running", "Browser QA workflow started", "")
		return nil
	})
	if err != nil {
		return model.Run{}, err
	}

	observation, driveErr := s.driver.RunInitial(ctx, tc)
	if driveErr == nil {
		return s.finishSuccess(run.ID, observation, "QA workflow completed without recovery", "")
	}

	var recoverable *model.RecoverableStepError
	if errors.As(driveErr, &recoverable) {
		return s.store.Update(run.ID, func(r *model.Run) error {
			r.Status = model.RunFailedRecoverable
			r.CurrentURL = observation.CurrentURL
			r.Failure = recoverable.Error()
			r.FailureAnalysis = &model.FailureAnalysis{
				Step:             recoverable.Step,
				FailedSelector:   recoverable.Selector,
				FallbackSelector: recoverable.FallbackSelector,
				Explanation:      fmt.Sprintf("The selector %s is stale. The page exposes stable fallback %s.", recoverable.Selector, recoverable.FallbackSelector),
				Recoverable:      true,
			}
			r.UpdatedAt = time.Now().UTC()
			appendObservationEvidence(r, observation, "failure")
			appendEvent(r, "failed_recoverable", "Selector failed; safe recovery is available", recoverable.Error())
			return nil
		})
	}

	failed, updateErr := s.store.Update(run.ID, func(r *model.Run) error {
		r.Status = model.RunFailed
		r.Failure = driveErr.Error()
		r.CurrentURL = observation.CurrentURL
		r.UpdatedAt = time.Now().UTC()
		completed := r.UpdatedAt
		r.CompletedAt = &completed
		appendObservationEvidence(r, observation, "failure")
		appendEvent(r, "failed", "Browser QA workflow failed", driveErr.Error())
		return nil
	})
	if updateErr != nil {
		return model.Run{}, updateErr
	}
	return failed, driveErr
}

func (s *Service) GetRun(id string) (model.Run, error) {
	return s.store.Get(id)
}

func (s *Service) InspectFailure(runID string) (model.FailureAnalysis, error) {
	run, err := s.store.Get(runID)
	if err != nil {
		return model.FailureAnalysis{}, err
	}
	if run.Status != model.RunFailedRecoverable || run.FailureAnalysis == nil {
		return model.FailureAnalysis{}, ErrInvalidRunState
	}
	return *run.FailureAnalysis, nil
}

func (s *Service) RetryFailedStep(ctx context.Context, runID string) (model.Run, error) {
	run, err := s.store.Get(runID)
	if err != nil {
		return model.Run{}, err
	}
	if run.Status != model.RunFailedRecoverable || run.FailureAnalysis == nil || strings.TrimSpace(run.FailureAnalysis.FallbackSelector) == "" {
		return model.Run{}, ErrInvalidRunState
	}
	tc, ok := s.getTest(run.TestID)
	if !ok {
		return model.Run{}, ErrTestNotFound
	}
	selector := run.FailureAnalysis.FallbackSelector
	if _, err := s.store.Update(run.ID, func(r *model.Run) error {
		r.Status = model.RunRunning
		r.UpdatedAt = time.Now().UTC()
		appendEvent(r, "recovery_started", "Retrying failed step with observed stable selector", "")
		return nil
	}); err != nil {
		return model.Run{}, err
	}

	observation, driveErr := s.driver.Retry(ctx, tc, selector)
	if driveErr != nil {
		failed, updateErr := s.store.Update(run.ID, func(r *model.Run) error {
			r.Status = model.RunFailed
			r.Failure = driveErr.Error()
			r.UpdatedAt = time.Now().UTC()
			completed := r.UpdatedAt
			r.CompletedAt = &completed
			appendObservationEvidence(r, observation, "recovery_failure")
			appendEvent(r, "recovery_failed", "Recovery attempt failed", driveErr.Error())
			return nil
		})
		if updateErr != nil {
			return model.Run{}, updateErr
		}
		return failed, driveErr
	}
	return s.finishSuccess(run.ID, observation, "Recovered stale selector and verified order confirmation", selector)
}

func (s *Service) ExportRegressionTest(runID string) (string, error) {
	run, err := s.store.Get(runID)
	if err != nil {
		return "", err
	}
	if run.Status != model.RunSucceeded || strings.TrimSpace(run.RecoveredSelector) == "" {
		return "", ErrInvalidRunState
	}
	target := tsSingleQuoted(run.TargetURL)
	selector := tsSingleQuoted(run.RecoveredSelector)
	return fmt.Sprintf(`import { test, expect } from '@playwright/test';

test('checkout remains recoverable', async ({ page }) => {
  await page.goto('%s');
  await page.locator('%s').click();
  await expect(page.getByText('Order confirmed')).toBeVisible();
});
`, target, selector), nil
}

func tsSingleQuoted(value string) string {
	replacer := strings.NewReplacer("\\", "\\\\", "'", "\\'", "\r", "\\r", "\n", "\\n")
	return replacer.Replace(value)
}

func (s *Service) finishSuccess(runID string, observation model.BrowserObservation, summary, recoveredSelector string) (model.Run, error) {
	return s.store.Update(runID, func(r *model.Run) error {
		r.Status = model.RunSucceeded
		r.CurrentURL = observation.CurrentURL
		r.Summary = summary
		r.Failure = ""
		r.RecoveredSelector = recoveredSelector
		r.UpdatedAt = time.Now().UTC()
		completed := r.UpdatedAt
		r.CompletedAt = &completed
		appendObservationEvidence(r, observation, "success")
		appendEvent(r, "succeeded", summary, "")
		return nil
	})
}

func (s *Service) getTest(id string) (model.TestDefinition, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tc, ok := s.tests[id]
	return tc, ok
}

func appendEvent(run *model.Run, typ, message, eventErr string) {
	run.Events = append(run.Events, model.RunEvent{
		Seq: len(run.Events) + 1, At: time.Now().UTC(), Type: typ, Message: message, Error: eventErr,
	})
}

func appendObservationEvidence(run *model.Run, observation model.BrowserObservation, label string) {
	if observation.CurrentURL != "" {
		run.CurrentURL = observation.CurrentURL
	}
	if observation.VisibleText != "" {
		run.Evidence = append(run.Evidence, model.Evidence{
			Step: len(run.Events), Kind: "text", Label: label + " visible text", Text: observation.VisibleText, CapturedAt: time.Now().UTC(),
		})
	}
	if len(observation.Screenshot) > 0 {
		run.Evidence = append(run.Evidence, model.Evidence{
			Step: len(run.Events), Kind: "screenshot", Label: label + " screenshot",
			DataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(observation.Screenshot), CapturedAt: time.Now().UTC(),
		})
	}
}

func newID(prefix string) string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return fmt.Sprintf("%s-%x", prefix, b[:])
}
