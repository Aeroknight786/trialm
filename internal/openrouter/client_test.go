package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// A key the trial site just made counts what the site said it may spend
// while OpenRouter is not yet accepting it, as happened on a second Mac.
func TestGrantCreditsWhileOpenRouterCatchesUp(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(401) }))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
	defer cancel()
	left := 0.79
	c := Client{BaseURL: srv.URL, Key: "new"}
	if v, err := c.GrantCredits(ctx, Grant{Key: "new", Remaining: &left}); err != nil || v != 0.79 {
		t.Fatal(v, err)
	}
	if _, err := c.GrantCredits(ctx, Grant{Key: "new"}); err == nil {
		t.Fatal("an own key OpenRouter refuses must not be kept")
	}
}
