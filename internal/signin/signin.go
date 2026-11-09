// Package signin gets an OpenRouter key: from the free trial sign-in at
// mortiphi.com or from the user's own OpenRouter account. It also sends the
// feedback a user chooses to send.
package signin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/openrouter"
	"github.com/aeroknight786/trialm/internal/setup"
	"github.com/aeroknight786/trialm/internal/store"
	"github.com/aeroknight786/trialm/internal/verify"
)

// Authorize signs in and keeps the key; it never sends a secret to the model
// or the terminal renderer. trial: Mortiphi's free trial sign-in (GitHub or
// Google, $1 of credit); otherwise the user's own OpenRouter account.
func Authorize(ctx context.Context, s store.Store, trial bool, open func(string) error, pasted string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	authURL, c := signIn(trial)
	g := openrouter.Grant{Key: strings.TrimSpace(pasted)}
	if g.Key == "" {
		pkce, err := openrouter.NewPKCE()
		if err != nil {
			return err
		}
		cb, err := openrouter.ListenCallback()
		if err != nil {
			return err
		}
		defer cb.Close()
		if err = open(pkce.URL(authURL, cb.URL)); err != nil {
			return err
		}
		code, err := cb.Wait(ctx)
		if err != nil {
			return err
		}
		if g, err = c.Grant(ctx, code, pkce.Verifier); err != nil {
			return err
		}
	}
	return saveAuthorization(ctx, s, g)
}

// signIn is where a sign-in goes: the trial site or OpenRouter itself.
func signIn(trial bool) (string, openrouter.Client) {
	if !trial {
		return "", openrouter.Client{}
	}
	authURL, api := openrouter.TrialURLs()
	return authURL, openrouter.Client{BaseURL: api}
}

func OpenBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Run()
}

// Headless is a sign-in finished by pasting the code from the browser's
// address bar, for machines that cannot receive the browser's redirect.
type Headless struct {
	store    store.Store
	client   openrouter.Client
	verifier string
	URL      string
	close    func()
}

func BeginHeadless(s store.Store, trial bool) (*Headless, error) {
	authURL, c := signIn(trial)
	p, e := openrouter.NewPKCE()
	if e != nil {
		return nil, e
	}
	cb, e := openrouter.ListenCallback()
	if e != nil {
		return nil, e
	}
	return &Headless{store: s, client: c, verifier: p.Verifier, URL: p.URL(authURL, cb.URL), close: func() { cb.Close() }}, nil
}
func (h *Headless) Close() { h.close() }
func (h *Headless) Finish(ctx context.Context, code string) error {
	defer h.close()
	if u, e := url.Parse(code); e == nil && u.Query().Get("code") != "" {
		code = u.Query().Get("code")
	}
	g, e := h.client.Grant(ctx, strings.TrimSpace(code), h.verifier)
	if e != nil {
		return e
	}
	return saveAuthorization(ctx, h.store, g)
}

// saveAuthorization checks the key against OpenRouter before keeping it,
// with what it can spend and, from a trial sign-in, the feedback token.
func saveAuthorization(ctx context.Context, s store.Store, g openrouter.Grant) error {
	balance, e := (openrouter.Client{Key: g.Key}).GrantCredits(ctx, g)
	if e != nil {
		return e
	}
	if e := s.SaveKey(g.Key); e != nil {
		return e
	}
	if g.Token != "" {
		if e := s.Write("trial.json", map[string]string{"token": g.Token, "expires_at": g.ExpiresAt}); e != nil {
			return e
		}
	}
	return s.Write("credits.json", figure.Exact(balance, "USD", figure.Measured, "openrouter:/key"))
}

// Explain says why a sign-in did not finish and what to do instead; own
// names the way to use one's own OpenRouter key ("press o", or a command).
func Explain(err error, trial bool, own string) string {
	if !trial {
		return "Could not connect OpenRouter. Press p to paste a key or try again."
	}
	var e openrouter.SignInError
	errors.As(err, &e)
	switch e {
	case "trial_used":
		return "Your $1 of free trial credit is used up. Set the model up locally (free), or " + own + " to use your own OpenRouter key."
	case "capacity":
		return "Free trials are full right now. " + strings.ToUpper(own[:1]) + own[1:] + " to use your own OpenRouter key."
	case "trials_paused":
		return "Free trials are paused right now. " + strings.ToUpper(own[:1]) + own[1:] + " to use your own OpenRouter key."
	}
	why := "" // the reason, on one short line, so the next report says what failed
	if err != nil {
		why = strings.Join(strings.Fields(err.Error()), " ")
		why = " (" + why[:min(90, len(why))] + ")"
	}
	return "Free trial sign-in did not finish" + why + ". Try again, or " + own + " to use your own OpenRouter key."
}

// SendFeedback posts one note the user chose to send to the trial site,
// signed with the token from the trial sign-in when there is one.
func SendFeedback(ctx context.Context, s store.Store, version, kind string, data map[string]any) error {
	var t struct {
		Token string `json:"token"`
	}
	_ = s.Read("trial.json", &t)
	_, api := openrouter.TrialURLs()
	body, err := json.Marshal(map[string]any{"kind": kind, "version": version, "data": data})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, "POST", api+"/feedback", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if t.Token != "" {
		req.Header.Set("Authorization", "Bearer "+t.Token)
	}
	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("feedback: HTTP %d", resp.StatusCode)
	}
	return nil
}

// TrialFeedback is what a trial note sends: the rating, the note, the model
// and this Mac's name, and the trial's timings. Never code, prompts or paths.
func TrialFeedback(e store.TrialEntry, note, tool, mac string) map[string]any {
	r := e.Report
	data := map[string]any{
		"rating": e.Rating, "note": strings.TrimSpace(note), "tool": tool, "mac": mac,
		"model": e.Model, "quant": e.Quant, "runtime": e.Runtime, "context": e.Context,
		"first":      map[string]any{"in": r.First.In, "out": r.First.Out, "seconds": r.First.Seconds},
		"follow_ups": map[string]any{"count": r.FollowUpCount, "in": r.FollowUps.In, "out": r.FollowUps.Out, "seconds": r.FollowUps.Seconds},
	}
	if r.SpendKnown {
		data["spend_usd"] = r.SpendUSD
	}
	return data
}

// SetupReport is what sharing a setup sends: the model, engine and context,
// this Mac's name, and the measured speeds, for people with similar Macs.
// Check details (which can hold paths and errors) stay on this Mac.
func SetupReport(p setup.Plan, r verify.Report, mac string) map[string]any {
	checks := map[string]bool{}
	for _, c := range r.Checks {
		checks[c.Name] = c.OK
	}
	return map[string]any{
		"model": p.Model, "quant": p.Quant, "runtime": p.Runtime, "tool": p.Harness, "context": p.ContextTokens, "mac": mac,
		"ready": r.Ready, "checks": checks, "decode_tps": r.DecodeTPS, "prefill_tps": r.PrefillTPS, "follow_up_ttft_s": r.FollowUpTTFT,
	}
}
