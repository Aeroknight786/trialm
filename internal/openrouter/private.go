package openrouter

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PrivateModels maps Hugging Face ids (lower case) to the OpenRouter models
// some provider serves without retaining prompts, from the public model and
// zero-retention lists (no key needed). The answer is cached in dir a day.
func (c Client) PrivateModels(ctx context.Context, dir string) (map[string]string, error) {
	path := filepath.Join(dir, "openrouter-private.json")
	out := map[string]string{}
	if st, err := os.Stat(path); err == nil && time.Since(st.ModTime()) < 24*time.Hour {
		if b, err := os.ReadFile(path); err == nil && json.Unmarshal(b, &out) == nil {
			return out, nil
		}
	}
	var models struct {
		Data []struct {
			ID   string `json:"id"`
			HFID string `json:"hugging_face_id"`
		} `json:"data"`
	}
	var zdr struct {
		Data []Endpoint `json:"data"`
	}
	if err := c.Do(ctx, "GET", "/models", nil, &models); err != nil {
		return nil, err
	}
	if err := c.Do(ctx, "GET", "/endpoints/zdr", nil, &zdr); err != nil {
		return nil, err
	}
	private := map[string]bool{}
	for _, e := range zdr.Data {
		if e.SafetyRetention != nil && !*e.SafetyRetention {
			private[e.ModelID] = true
		}
	}
	for _, m := range models.Data {
		// ":free" variants route differently and fail the trial's privacy rule.
		if m.HFID != "" && private[m.ID] && !strings.Contains(m.ID, ":") {
			out[strings.ToLower(m.HFID)] = m.ID
		}
	}
	if b, err := json.Marshal(out); err == nil && os.MkdirAll(dir, 0o700) == nil {
		_ = os.WriteFile(path, b, 0o600)
	}
	return out, nil
}
