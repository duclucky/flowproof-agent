package orchestrator

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ducky/flowproof-agent/internal/model"
	"github.com/ducky/flowproof-agent/internal/store"
)

const stableCheckoutSelector = `[data-testid="checkout-submit"]`

type fakeDriver struct {
	initialCalls  int
	retryCalls    int
	retrySelector string
}

func (f *fakeDriver) RunInitial(_ context.Context, tc model.TestDefinition) (model.BrowserObservation, error) {
	f.initialCalls++
	return model.BrowserObservation{
			CurrentURL:        tc.TargetURL,
			VisibleText:       "Cart ready. Checkout is available.",
			Screenshot:        []byte("fake-png"),
			AttemptedSelector: "#checkout-submit",
			FallbackSelector:  stableCheckoutSelector,
		}, &model.RecoverableStepError{
			Step:             "submit_checkout",
			Selector:         "#checkout-submit",
			FallbackSelector: stableCheckoutSelector,
			Message:          "selector #checkout-submit not found",
		}
}

func (f *fakeDriver) Retry(_ context.Context, tc model.TestDefinition, selector string) (model.BrowserObservation, error) {
	f.retryCalls++
	f.retrySelector = selector
	return model.BrowserObservation{
		CurrentURL:        tc.TargetURL,
		VisibleText:       "Order confirmed. QA-2048",
		Screenshot:        []byte("fake-success-png"),
		AttemptedSelector: selector,
	}, nil
}

func newTestService(t *testing.T) (*Service, *fakeDriver) {
	t.Helper()
	st, err := store.NewFileStore(filepath.Join(t.TempDir(), "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	driver := &fakeDriver{}
	return New(st, driver), driver
}

func createDemoTest(t *testing.T, svc *Service) model.TestDefinition {
	t.Helper()
	tc, err := svc.CreateTest(model.CreateTestRequest{
		TargetURL: "http://127.0.0.1:8080/demo-store",
		Objective: "Complete checkout and verify the order confirmation",
	})
	if err != nil {
		t.Fatalf("CreateTest: %v", err)
	}
	return tc
}

func TestRunStopsAtRecoverableSelectorFailureWithEvidence(t *testing.T) {
	svc, driver := newTestService(t)
	tc := createDemoTest(t, svc)

	run, err := svc.StartRun(context.Background(), tc.ID)
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	if run.Status != model.RunFailedRecoverable {
		t.Fatalf("status = %q, want %q", run.Status, model.RunFailedRecoverable)
	}
	if driver.initialCalls != 1 {
		t.Fatalf("initial calls = %d, want 1", driver.initialCalls)
	}
	if run.FailureAnalysis == nil {
		t.Fatal("expected structured failure analysis")
	}
	if run.FailureAnalysis.FailedSelector != "#checkout-submit" {
		t.Fatalf("failed selector = %q", run.FailureAnalysis.FailedSelector)
	}
	if run.FailureAnalysis.FallbackSelector != stableCheckoutSelector {
		t.Fatalf("fallback selector = %q", run.FailureAnalysis.FallbackSelector)
	}
	if len(run.Events) < 3 {
		t.Fatalf("events = %d, want lifecycle evidence", len(run.Events))
	}
	if len(run.Evidence) == 0 || run.Evidence[len(run.Evidence)-1].DataURL == "" {
		t.Fatal("expected screenshot evidence data URL")
	}
}

func TestInspectRetryAndExportUseRecoveredStableSelector(t *testing.T) {
	svc, driver := newTestService(t)
	tc := createDemoTest(t, svc)
	run, err := svc.StartRun(context.Background(), tc.ID)
	if err != nil {
		t.Fatal(err)
	}

	analysis, err := svc.InspectFailure(run.ID)
	if err != nil {
		t.Fatalf("InspectFailure: %v", err)
	}
	if analysis.FallbackSelector != stableCheckoutSelector || !strings.Contains(strings.ToLower(analysis.Explanation), "stale") {
		t.Fatalf("unexpected analysis: %#v", analysis)
	}

	run, err = svc.RetryFailedStep(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("RetryFailedStep: %v", err)
	}
	if run.Status != model.RunSucceeded {
		t.Fatalf("status after retry = %q, want succeeded", run.Status)
	}
	if driver.retryCalls != 1 || driver.retrySelector != stableCheckoutSelector {
		t.Fatalf("retry calls=%d selector=%q", driver.retryCalls, driver.retrySelector)
	}
	if run.RecoveredSelector != stableCheckoutSelector {
		t.Fatalf("recovered selector = %q", run.RecoveredSelector)
	}

	code, err := svc.ExportRegressionTest(run.ID)
	if err != nil {
		t.Fatalf("ExportRegressionTest: %v", err)
	}
	for _, want := range []string{"@playwright/test", stableCheckoutSelector, "Order confirmed"} {
		if !strings.Contains(code, want) {
			t.Fatalf("export missing %q:\n%s", want, code)
		}
	}
}

func TestRetryRejectsRunsThatAreNotRecoverable(t *testing.T) {
	svc, _ := newTestService(t)
	tc := createDemoTest(t, svc)
	run, err := svc.StartRun(context.Background(), tc.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.RetryFailedStep(context.Background(), run.ID); err != nil {
		t.Fatal(err)
	}

	_, err = svc.RetryFailedStep(context.Background(), run.ID)
	if !errors.Is(err, ErrInvalidRunState) {
		t.Fatalf("second retry error = %v, want ErrInvalidRunState", err)
	}
}
