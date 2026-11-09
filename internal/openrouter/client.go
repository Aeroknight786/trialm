// Package openrouter implements authentication, credits and endpoint discovery.
package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const API = "https://openrouter.ai/api/v1"

type Client struct {
	BaseURL, Key string
	HTTP         *http.Client
}

func (c Client) Do(ctx context.Context, method, path string, body, out any) error {
	base := c.BaseURL
	if base == "" {
		base = API
		// A local stand-in for OpenRouter, in end-to-end tests of sign-in.
		if v := os.Getenv("TRIALM_OPENROUTER_URL"); v != "" {
			base = v
		}
	}
	var r io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		r = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, strings.TrimRight(base, "/")+path, r)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Title", "triaLM")
	req.Header.Set("HTTP-Referer", "https://github.com/aeroknight786/trialm")
	if c.Key != "" {
		req.Header.Set("Authorization", "Bearer "+c.Key)
	}
	hc := c.HTTP
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}
	resp, err := hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Never echo a remote body: an auth service may reflect a secret.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("OpenRouter %s: HTTP %d", path, resp.StatusCode)
	}
	return json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(out)
}

// Credits is what this key can still spend. A key with its own limit, such
// as a $1 trial key minted on someone else's account, reports what is left
// of that limit; otherwise it is the account's balance.
func (c Client) Credits(ctx context.Context) (float64, error) {
	var k struct {
		Data struct {
			Limit          *float64 `json:"limit"`
			LimitRemaining *float64 `json:"limit_remaining"`
		} `json:"data"`
	}
	if err := c.Do(ctx, "GET", "/key", nil, &k); err != nil {
		return 0, err
	}
	if k.Data.Limit != nil && k.Data.LimitRemaining != nil {
		return *k.Data.LimitRemaining, nil
	}
	var r struct {
		Data struct {
			TotalCredits float64 `json:"total_credits"`
			TotalUsage   float64 `json:"total_usage"`
		} `json:"data"`
	}
	err := c.Do(ctx, "GET", "/credits", nil, &r)
	return r.Data.TotalCredits - r.Data.TotalUsage, err
}

// Grant is what a sign-in hands back: the OpenRouter key, plus, from the
// Mortiphi trial sign-in, a token for sending feedback.
type Grant struct {
	Key       string   `json:"key"`
	Token     string   `json:"token"`
	ExpiresAt string   `json:"expires_at"`
	Remaining *float64 `json:"remaining_usd"` // what the trial site made the key to spend
}

// GrantCredits is what a key just made can spend. OpenRouter can take longer
// than NewKeyCredits waits to accept a key it just made; the trial site said
// what the key may spend when it made it, so that stands in until then.
func (c Client) GrantCredits(ctx context.Context, g Grant) (float64, error) {
	v, err := c.NewKeyCredits(ctx)
	if err != nil && g.Remaining != nil {
		return *g.Remaining, nil
	}
	return v, err
}

func (c Client) Exchange(ctx context.Context, code, verifier string) (string, error) {
	g, err := c.Grant(ctx, code, verifier)
	return g.Key, err
}
func (c Client) Grant(ctx context.Context, code, verifier string) (Grant, error) {
	var g Grant
	err := c.Do(ctx, "POST", "/auth/keys", map[string]string{"code": code, "code_verifier": verifier, "code_challenge_method": "S256"}, &g)
	if err == nil && g.Key == "" {
		err = fmt.Errorf("no key was returned")
	}
	return g, err
}

// Price is USD per token, as supplied by the endpoints API.
type Price struct {
	Prompt     string `json:"prompt"`
	Completion string `json:"completion"`
	CacheRead  string `json:"input_cache_read"`
	Request    string `json:"request"`
}
type Endpoint struct {
	ModelID             string   `json:"model_id"`
	SafetyRetention     *bool    `json:"safety_retention"`
	Name                string   `json:"name"`
	ProviderName        string   `json:"provider_name"`
	Tag                 string   `json:"tag"`
	Quantization        string   `json:"quantization"`
	ContextLength       int      `json:"context_length"`
	SupportedParameters []string `json:"supported_parameters"`
	DataPolicy          struct {
		DataCollection string `json:"data_collection"`
		ZDR            bool   `json:"zdr"`
	} `json:"data_policy"`
	// Some cached endpoint records expose these at the top level.
	DataCollection string   `json:"data_collection"`
	ZDR            bool     `json:"zdr"`
	Pricing        Price    `json:"pricing"`
	Uptime         *float64 `json:"uptime_last_30m"`
}

func (c Client) Endpoints(ctx context.Context, id string) ([]Endpoint, error) {
	var r struct {
		Data struct {
			Endpoints []Endpoint `json:"endpoints"`
		} `json:"data"`
	}
	// Preserve the owner/model slash, while preventing query/path injection.
	parts := strings.Split(id, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	err := c.Do(ctx, "GET", "/models/"+strings.Join(parts, "/")+"/endpoints", nil, &r)
	return r.Data.Endpoints, err
}

// PrivateEndpoints joins unlabelled model records to the published ZDR list.
// The normal endpoint schema does not promise data_policy fields. Only an
// exact model/tag/precision match with explicit safety_retention:false can
// establish eligibility; a contradictory allow policy is never overwritten.
// https://openrouter.ai/docs/api/api-reference/endpoints/preview-the-impact-of-zdr-on-the-available-endpoints
func (c Client) PrivateEndpoints(ctx context.Context, id string, zdr bool) ([]Endpoint, error) {
	eps, err := c.Endpoints(ctx, id)
	if err != nil {
		return nil, err
	}
	need := false
	for _, e := range eps {
		if e.DataCollection == "" && e.DataPolicy.DataCollection == "" {
			need = true
		}
		if zdr && !e.ZDR && !e.DataPolicy.ZDR {
			need = true
		}
	}
	if !need {
		return eps, nil
	}
	var list struct {
		Data []Endpoint `json:"data"`
	}
	if err = c.Do(ctx, "GET", "/endpoints/zdr", nil, &list); err != nil {
		return nil, err
	}
	for i, e := range eps {
		policy := e.DataCollection
		if policy == "" {
			policy = e.DataPolicy.DataCollection
		}
		if policy != "" && policy != "deny" {
			continue
		}
		for _, known := range list.Data {
			if known.ModelID == id && known.Tag == e.Tag && known.Quantization == e.Quantization && known.SafetyRetention != nil && !*known.SafetyRetention {
				eps[i].DataCollection = "deny"
				eps[i].ZDR = true
				safe := false
				eps[i].SafetyRetention = &safe
				break
			}
		}
	}
	return eps, nil
}

// NewKeyCredits is Credits for a key minted moments ago: OpenRouter can take
// a few seconds to accept a new key, so it asks again for up to ~20 s.
func (c Client) NewKeyCredits(ctx context.Context) (float64, error) {
	var err error
	for _, wait := range []time.Duration{0, time.Second, 2 * time.Second, 3 * time.Second, 5 * time.Second, 8 * time.Second} {
		select {
		case <-time.After(wait):
		case <-ctx.Done():
			return 0, ctx.Err()
		}
		var v float64
		if v, err = c.Credits(ctx); err == nil {
			return v, nil
		}
	}
	return 0, fmt.Errorf("OpenRouter did not accept the new trial key: %w", err)
}
