package signin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aeroknight786/trialm/internal/openrouter"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/verify"
)

func TestHeadlessCodeAgainstFake(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/auth/keys" {
			fmt.Fprint(w, `{"key":"fake-secret"}`)
		} else {
			fmt.Fprint(w, `{"data":{"total_credits":10,"total_usage":1}}`)
		}
	}))
	defer srv.Close()
	s := store.Store{Dir: t.TempDir()}
	t.Setenv("TRIALM_OPENROUTER_URL", srv.URL)
	h, e := BeginHeadless(s, false)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(h.URL, h.verifier) || !strings.Contains(h.URL, "S256") {
		t.Fatal("invalid public authorization URL")
	}
	if e = h.Finish(context.Background(), "http://127.0.0.1/callback?code=one-use-code"); e != nil {
		t.Fatal(e)
	}
	t.Setenv("OPENROUTER_API_KEY", "")
	if s.Key() != "fake-secret" {
		t.Fatal("key not saved")
	}
}

// A sign-in page that stops (?error=) explains itself, and the trial
// sign-in offers the user's own key instead.
func TestExplainSignInStops(t *testing.T) {
	cases := map[error]string{
		openrouter.SignInError("trial_used"):    "used up",
		openrouter.SignInError("capacity"):      "full right now",
		openrouter.SignInError("trials_paused"): "paused",
		errors.New("dial tcp: no route"):        "did not finish",
	}
	for err, want := range cases {
		if got := Explain(err, true, "press o"); !strings.Contains(got, want) || !strings.Contains(strings.ToLower(got), "press o to use your own openrouter key") {
			t.Fatal(err, got)
		}
	}
	if got := Explain(errors.New("x"), false, "press o"); !strings.Contains(got, "OpenRouter") {
		t.Fatal(got)
	}
}

// A shared setup report carries speeds and check results, never the check
// details, which can hold file paths and error text.
func TestSetupReportLeavesDetailsOnThisMac(t *testing.T) {
	r := verify.Report{Ready: true, DecodeTPS: map[int]float64{0: 41.5}, PrefillTPS: 900, Checks: []verify.Check{{Name: "harness", OK: true, Detail: "/Users/someone/project/secret.txt"}}}
	data := SetupReport(setup.Plan{Model: "qwen3.6:35b-a3b", Quant: "Q4_K_M", Runtime: "llamacpp"}, r, "Apple M5")
	b, _ := json.Marshal(data)
	if strings.Contains(string(b), "/Users/") || !strings.Contains(string(b), `"harness":true`) || !strings.Contains(string(b), "41.5") {
		t.Fatal(string(b))
	}
}
