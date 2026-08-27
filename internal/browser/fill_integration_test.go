package browser

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/chromedp/chromedp"
	"github.com/ducky/flowproof-agent/internal/model"
)

func TestSessionExecuteFillSetsInputValueInRealChromium(t *testing.T) {
	if os.Getenv("FLOWPROOF_BROWSER_INTEGRATION") != "1" {
		t.Skip("set FLOWPROOF_BROWSER_INTEGRATION=1 to run real Chromium integration test")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<!doctype html><input name="coupon" autocomplete="off">`))
	}))
	defer srv.Close()

	session, err := NewSession(context.Background(), os.Getenv("FLOWPROOF_CHROME_PATH"), 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if err := session.Navigate(srv.URL); err != nil {
		t.Fatal(err)
	}
	if err := session.Execute(model.Action{Kind: model.ActionFill, Selector: `[name="coupon"]`, Value: "SAVE20"}); err != nil {
		t.Fatalf("fill ordinary input: %v", err)
	}
	var got string
	if err := chromedp.Run(session.ctx, chromedp.Value(`[name="coupon"]`, &got, chromedp.ByQuery)); err != nil {
		t.Fatal(err)
	}
	if got != "SAVE20" {
		t.Fatalf("value = %q, want SAVE20", got)
	}
}
