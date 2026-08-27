package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/ducky/flowproof-agent/internal/model"
)

const (
	staleCheckoutSelector  = "#checkout-submit"
	stableCheckoutSelector = `[data-testid="checkout-submit"]`
)

type BrowserSession interface {
	Close()
	Navigate(string) error
	Execute(model.Action) error
	Snapshot() (model.Snapshot, error)
}

type SessionFactory func(context.Context) (BrowserSession, error)

type ChromedpDriver struct {
	newSession SessionFactory
}

func NewDriver(chromePath string, stepTimeout time.Duration) *ChromedpDriver {
	return NewWithSessionFactory(func(ctx context.Context) (BrowserSession, error) {
		return NewSession(ctx, chromePath, stepTimeout)
	})
}

func NewWithSessionFactory(factory SessionFactory) *ChromedpDriver {
	return &ChromedpDriver{newSession: factory}
}

type Session struct {
	ctx         context.Context
	cancel      context.CancelFunc
	allocCancel context.CancelFunc
	stepTimeout time.Duration
}

func NewSession(parent context.Context, chromePath string, stepTimeout time.Duration) (*Session, error) {
	opts := append([]chromedp.ExecAllocatorOption{}, chromedp.DefaultExecAllocatorOptions[:]...)
	opts = append(opts,
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.Flag("hide-scrollbars", true),
		chromedp.WindowSize(1440, 1000),
	)
	if strings.TrimSpace(chromePath) != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
	}
	allocCtx, allocCancel := chromedp.NewExecAllocator(parent, opts...)
	ctx, cancel := chromedp.NewContext(allocCtx)
	if err := chromedp.Run(ctx); err != nil {
		cancel()
		allocCancel()
		return nil, fmt.Errorf("start headless Chromium: %w", err)
	}
	return &Session{ctx: ctx, cancel: cancel, allocCancel: allocCancel, stepTimeout: stepTimeout}, nil
}

func (s *Session) Close() {
	s.cancel()
	s.allocCancel()
}

func (s *Session) Navigate(rawURL string) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.stepTimeout)
	defer cancel()
	if err := chromedp.Run(ctx,
		chromedp.Navigate(rawURL),
		chromedp.WaitReady("body", chromedp.ByQuery),
	); err != nil {
		return fmt.Errorf("navigate %s: %w", rawURL, err)
	}
	return nil
}

func (s *Session) Snapshot() (model.Snapshot, error) {
	ctx, cancel := context.WithTimeout(s.ctx, s.stepTimeout)
	defer cancel()
	var (
		url        string
		title      string
		visible    string
		dom        string
		screenshot []byte
	)
	inventoryJS := `(() => {
  const clean = (v) => String(v || '').replace(/\s+/g, ' ').trim().slice(0, 240);
  const nodes = [...document.querySelectorAll('a,button,input,textarea,select,[role="button"],[contenteditable="true"]')].slice(0, 220);
  return nodes.map((el, i) => ({
    i,
    tag: el.tagName.toLowerCase(),
    id: el.id || '',
    name: el.getAttribute('name') || '',
    type: el.getAttribute('type') || '',
    role: el.getAttribute('role') || '',
    testid: el.getAttribute('data-testid') || '',
    aria: el.getAttribute('aria-label') || '',
    placeholder: el.getAttribute('placeholder') || '',
    text: clean(el.innerText || el.value || el.textContent),
    disabled: !!el.disabled,
    visible: !!(el.offsetWidth || el.offsetHeight || el.getClientRects().length)
  })).filter(x => x.visible);
})()`
	if err := chromedp.Run(ctx,
		chromedp.Location(&url),
		chromedp.Title(&title),
		chromedp.Evaluate(`document.body ? document.body.innerText.slice(0, 18000) : ''`, &visible),
		chromedp.EvaluateAsDevTools("JSON.stringify("+inventoryJS+")", &dom),
		chromedp.CaptureScreenshot(&screenshot),
	); err != nil {
		return model.Snapshot{}, fmt.Errorf("capture browser snapshot: %w", err)
	}
	return model.Snapshot{URL: url, Title: title, DOM: dom, VisibleText: visible, Screenshot: screenshot}, nil
}

func (s *Session) Execute(action model.Action) error {
	ctx, cancel := context.WithTimeout(s.ctx, s.stepTimeout)
	defer cancel()

	switch action.Kind {
	case model.ActionNavigate:
		return s.Navigate(action.URL)
	case model.ActionClick:
		return wrapAction(action, chromedp.Run(ctx,
			chromedp.WaitVisible(action.Selector, chromedp.ByQuery),
			chromedp.ScrollIntoView(action.Selector, chromedp.ByQuery),
			chromedp.Click(action.Selector, chromedp.ByQuery),
		))
	case model.ActionFill:
		fillJS := `(() => {
  const el = document.querySelector(` + jsString(action.Selector) + `);
  if (!el) throw new Error('fill target not found');
  const proto = Object.getPrototypeOf(el);
  const descriptor = proto && Object.getOwnPropertyDescriptor(proto, 'value');
  if (descriptor && descriptor.set) descriptor.set.call(el, ` + jsString(action.Value) + `);
  else el.value = ` + jsString(action.Value) + `;
  el.dispatchEvent(new Event('input', { bubbles: true }));
  el.dispatchEvent(new Event('change', { bubbles: true }));
  return true;
})()`
		return wrapAction(action, chromedp.Run(ctx,
			chromedp.WaitVisible(action.Selector, chromedp.ByQuery),
			chromedp.ScrollIntoView(action.Selector, chromedp.ByQuery),
			chromedp.Focus(action.Selector, chromedp.ByQuery),
			chromedp.Evaluate(fillJS, nil),
		))
	case model.ActionPress:
		tasks := chromedp.Tasks{}
		if action.Selector != "" {
			tasks = append(tasks, chromedp.WaitVisible(action.Selector, chromedp.ByQuery), chromedp.Focus(action.Selector, chromedp.ByQuery))
		}
		tasks = append(tasks, chromedp.KeyEvent(action.Key))
		return wrapAction(action, chromedp.Run(ctx, tasks...))
	case model.ActionWait:
		wait := time.Duration(action.WaitMS) * time.Millisecond
		if wait <= 0 {
			wait = 750 * time.Millisecond
		}
		if wait > 5*time.Second {
			wait = 5 * time.Second
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
			return nil
		}
	case model.ActionAssert:
		if action.Selector != "" {
			if err := chromedp.Run(ctx, chromedp.WaitVisible(action.Selector, chromedp.ByQuery)); err != nil {
				return wrapAction(action, err)
			}
		}
		if action.Text != "" {
			var found bool
			needle := action.Text
			if err := chromedp.Run(ctx, chromedp.Evaluate(`document.body && document.body.innerText.includes(`+jsString(needle)+`)`, &found)); err != nil {
				return wrapAction(action, err)
			}
			if !found {
				return fmt.Errorf("assert text not found: %q", needle)
			}
		}
		return nil
	case model.ActionComplete:
		return nil
	default:
		return fmt.Errorf("unsupported browser action %q", action.Kind)
	}
}

func (d *ChromedpDriver) RunInitial(ctx context.Context, tc model.TestDefinition) (model.BrowserObservation, error) {
	session, err := d.openSession(ctx, tc.TargetURL)
	if err != nil {
		return model.BrowserObservation{}, err
	}
	defer session.Close()

	if err := runCheckoutSetup(session); err != nil {
		return observe(session, "", ""), err
	}
	stale := model.Action{Kind: model.ActionClick, Selector: staleCheckoutSelector, Reason: "Exercise stale checkout selector for recovery demo"}
	if err := session.Execute(stale); err != nil {
		observation := observe(session, staleCheckoutSelector, "")
		fallback := discoverCheckoutFallback(observation)
		observation.FallbackSelector = fallback
		if fallback != "" {
			return observation, &model.RecoverableStepError{
				Step:             "submit_checkout",
				Selector:         staleCheckoutSelector,
				FallbackSelector: fallback,
				Message:          err.Error(),
			}
		}
		return observation, err
	}
	return observe(session, staleCheckoutSelector, ""), nil
}

func (d *ChromedpDriver) Retry(ctx context.Context, tc model.TestDefinition, selector string) (model.BrowserObservation, error) {
	if strings.TrimSpace(selector) != stableCheckoutSelector {
		return model.BrowserObservation{}, fmt.Errorf("retry selector is not the observed safe fallback")
	}
	session, err := d.openSession(ctx, tc.TargetURL)
	if err != nil {
		return model.BrowserObservation{}, err
	}
	defer session.Close()
	if err := runCheckoutSetup(session); err != nil {
		return observe(session, selector, ""), err
	}
	if err := session.Execute(model.Action{Kind: model.ActionClick, Selector: selector, Reason: "Retry with observed stable checkout selector"}); err != nil {
		return observe(session, selector, ""), err
	}
	if err := session.Execute(model.Action{Kind: model.ActionAssert, Text: "Order confirmed", Reason: "Verify checkout success"}); err != nil {
		return observe(session, selector, ""), err
	}
	return observe(session, selector, ""), nil
}

func (d *ChromedpDriver) openSession(ctx context.Context, targetURL string) (BrowserSession, error) {
	if d == nil || d.newSession == nil {
		return nil, fmt.Errorf("browser session factory is not configured")
	}
	session, err := d.newSession(ctx)
	if err != nil {
		return nil, err
	}
	if err := session.Navigate(targetURL); err != nil {
		session.Close()
		return nil, err
	}
	return session, nil
}

func runCheckoutSetup(session BrowserSession) error {
	actions := []model.Action{
		{Kind: model.ActionClick, Selector: `[data-testid="add-to-cart"]`, Reason: "Add demo product to cart"},
		{Kind: model.ActionFill, Selector: `[name="coupon"]`, Value: "SAVE20", Reason: "Apply demo coupon"},
		{Kind: model.ActionClick, Selector: `[data-testid="apply-coupon"]`, Reason: "Confirm coupon"},
	}
	for _, action := range actions {
		if err := session.Execute(action); err != nil {
			return err
		}
	}
	return nil
}

func observe(session BrowserSession, attemptedSelector, fallbackSelector string) model.BrowserObservation {
	snapshot, err := session.Snapshot()
	if err != nil {
		return model.BrowserObservation{AttemptedSelector: attemptedSelector, FallbackSelector: fallbackSelector}
	}
	return model.BrowserObservation{
		CurrentURL:        snapshot.URL,
		VisibleText:       snapshot.VisibleText,
		Screenshot:        snapshot.Screenshot,
		DOM:               snapshot.DOM,
		AttemptedSelector: attemptedSelector,
		FallbackSelector:  fallbackSelector,
	}
}

func discoverCheckoutFallback(observation model.BrowserObservation) string {
	dom := observation.DOM
	if strings.Contains(dom, `data-testid="checkout-submit"`) ||
		strings.Contains(dom, `"testid":"checkout-submit"`) {
		return stableCheckoutSelector
	}
	return ""
}

func wrapAction(action model.Action, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s selector=%q: %w", action.Kind, action.Selector, err)
}

func jsString(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, `'`, `\'`)
	value = strings.ReplaceAll(value, "\r", `\r`)
	value = strings.ReplaceAll(value, "\n", `\n`)
	return `'` + value + `'`
}
