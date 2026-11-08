package openrouter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestCallbackScopeAndCancellation(t *testing.T) {
	c, err := ListenCallback()
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	u, _ := url.Parse(c.URL)
	u.Host = strings.Replace(u.Host, "localhost", "127.0.0.1", 1)
	bad := *u
	bad.Path = "/callback/guess"
	r, err := http.Get(bad.String() + "?code=guess")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 404 {
		t.Fatal(r.StatusCode)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if _, err = c.Wait(ctx); err == nil {
		t.Fatal("wait ignored cancellation")
	}
	r, err = http.Get(u.String())
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	if r.StatusCode != 400 {
		t.Fatal(r.StatusCode)
	}
	r, err = http.Get(u.String() + "?code=ok")
	if err != nil {
		t.Fatal(err)
	}
	r.Body.Close()
	code, err := c.Wait(context.Background())
	if err != nil || code != "ok" {
		t.Fatal(code, err)
	}
}

// A minted trial key reports its own remaining limit, not the account's
// balance (which belongs to whoever minted it).
func TestCreditsPreferTheKeyLimit(t *testing.T) {
	limit := `{"data":{"limit":1,"limit_remaining":0.84}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/key":
			_, _ = w.Write([]byte(limit))
		case "/credits":
			_, _ = w.Write([]byte(`{"data":{"total_credits":35,"total_usage":19}}`))
		}
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, Key: "k"}
	if got, err := c.Credits(context.Background()); err != nil || got != 0.84 {
		t.Fatal(got, err)
	}
	limit = `{"data":{"limit":null,"limit_remaining":null}}`
	if got, err := c.Credits(context.Background()); err != nil || got != 16 {
		t.Fatal(got, err)
	}
}
