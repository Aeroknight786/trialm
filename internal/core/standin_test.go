package core

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/aeroknight786/trialm/internal/gguf"
	"github.com/aeroknight786/trialm/internal/hub"
)

// A GGUF repo resolves to its own model when OpenRouter hosts it privately;
// otherwise there is no trial, never a guessed id or a pretrained base.
func TestStandInMatchesHuggingFaceID(t *testing.T) {
	old, oldAPI := privateModels, hub.API
	defer func() { privateModels, hub.API = old, oldAPI }()
	privateModels = func(context.Context) map[string]string {
		return map[string]string{"liquidai/lfm3-1.7b": "liquid/lfm-3-1.7b"}
	}
	cards := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cards++
		_, _ = w.Write([]byte(`{"cardData":{"base_model":"LiquidAI/LFM2.5-8B-A1B-Base"}}`))
	}))
	defer srv.Close()
	hub.API = srv.URL
	f, s := &gguf.File{}, &gguf.Summary{Architecture: "lfm2moe", ParameterCount: 8e9}
	si := FindStandIn("https://huggingface.co/LiquidAI/LFM3-1.7B-GGUF/resolve/main/LFM3-1.7B-Q4_K_M.gguf", f, s)
	if si.Upstream != "liquid/lfm-3-1.7b" || si.Describe() != "" || cards != 0 {
		t.Fatalf("%+v cards=%d", si, cards)
	}
	si = FindStandIn("https://huggingface.co/LiquidAI/LFM2.5-8B-A1B-GGUF/resolve/main/LFM2.5-8B-A1B-Q4_K_M.gguf", f, s)
	if si.Upstream != "" || cards == 0 {
		t.Fatalf("%+v cards=%d", si, cards)
	}
	if readableName("Feb5E04Da4910Dd56D33F0Cd03747Ff298C1E801") != "" || readableName("LFM3 1.7B") != "LFM3 1.7B" {
		t.Fatal("hash names")
	}
}
