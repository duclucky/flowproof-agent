package browser

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ducky/flowproof-agent/internal/model"
)

const testStableSelector = `[data-testid="checkout-submit"]`

type fakeBrowserSession struct {
	actions []model.Action
	dom     string
}

func (s *fakeBrowserSession) Close() {}

func (s *fakeBrowserSession) Navigate(rawURL string) error {
	s.actions = append(s.actions, model.Action{Kind: model.ActionNavigate, URL: rawURL})
	return nil
}

func (s *fakeBrowserSession) Execute(action model.Action) error {
	s.actions = append(s.actions, action)
	if action.Kind == model.ActionClick && action.Selector == "#checkout-submit" {
		return errors.New("selector #checkout-submit not found")
	}
	return nil
}

func (s *fakeBrowserSession) Snapshot() (model.Snapshot, error) {
	dom := s.dom
	if dom == "" {
		dom = `<button data-testid="checkout-submit">Complete checkout</button><div>Order confirmed</div>`
	}
	return model.Snapshot{
		URL:         "http://127.0.0.1:8080/demo-store",
		DOM:         dom,
		VisibleText: "Checkout ready Order confirmed",
		Screenshot:  []byte("png"),
	}, nil
}

func TestDriverInitialFlowReturnsRecoverableStaleSelectorEvidence(t *testing.T) {
	session := &fakeBrowserSession{}
	driver := NewWithSessionFactory(func(context.Context) (BrowserSession, error) { return session, nil })
	tc := model.TestDefinition{TargetURL: "http://127.0.0.1:8080/demo-store"}

	observation, err := driver.RunInitial(context.Background(), tc)
	var recoverable *model.RecoverableStepError
	if !errors.As(err, &recoverable) {
		t.Fatalf("RunInitial error = %v, want RecoverableStepError", err)
	}
	if recoverable.Selector != "#checkout-submit" || recoverable.FallbackSelector != testStableSelector {
		t.Fatalf("unexpected recovery: %#v", recoverable)
	}
	if observation.FallbackSelector != testStableSelector || len(observation.Screenshot) == 0 {
		t.Fatalf("missing observation evidence: %#v", observation)
	}

	selectors := actionSelectors(session.actions)
	for _, want := range []string{`[data-testid="add-to-cart"]`, `[name="coupon"]`, `[data-testid="apply-coupon"]`, "#checkout-submit"} {
		if !strings.Contains(selectors, want) {
			t.Fatalf("initial actions %q missing %q", selectors, want)
		}
	}
}

func TestDriverRetryReplaysSetupAndUsesStableSelector(t *testing.T) {
	session := &fakeBrowserSession{}
	driver := NewWithSessionFactory(func(context.Context) (BrowserSession, error) { return session, nil })
	tc := model.TestDefinition{TargetURL: "http://127.0.0.1:8080/demo-store"}

	observation, err := driver.Retry(context.Background(), tc, testStableSelector)
	if err != nil {
		t.Fatalf("Retry: %v", err)
	}
	if !strings.Contains(observation.VisibleText, "Order confirmed") {
		t.Fatalf("retry observation = %#v", observation)
	}
	selectors := actionSelectors(session.actions)
	if !strings.Contains(selectors, testStableSelector) {
		t.Fatalf("retry actions %q missing stable selector", selectors)
	}
	if strings.Contains(selectors, "#checkout-submit") {
		t.Fatalf("retry actions still use stale selector: %q", selectors)
	}
}

func TestDriverDoesNotInventFallbackWhenPageDoesNotExposeIt(t *testing.T) {
	session := &fakeBrowserSession{dom: `<button class="pay">Complete checkout</button>`}
	driver := NewWithSessionFactory(func(context.Context) (BrowserSession, error) { return session, nil })
	tc := model.TestDefinition{TargetURL: "http://127.0.0.1:8080/demo-store"}

	observation, err := driver.RunInitial(context.Background(), tc)
	var recoverable *model.RecoverableStepError
	if err == nil {
		t.Fatal("RunInitial unexpectedly succeeded")
	}
	if errors.As(err, &recoverable) {
		t.Fatalf("invented recoverable fallback: %#v", recoverable)
	}
	if observation.FallbackSelector != "" {
		t.Fatalf("invented fallback selector %q", observation.FallbackSelector)
	}
}

func actionSelectors(actions []model.Action) string {
	parts := make([]string, 0, len(actions))
	for _, action := range actions {
		if action.Selector != "" {
			parts = append(parts, action.Selector)
		}
	}
	return strings.Join(parts, " | ")
}
