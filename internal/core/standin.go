package core

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aeroknight786/trialm/internal/catalog"
	"github.com/aeroknight786/trialm/internal/gguf"
	"github.com/aeroknight786/trialm/internal/hub"
	"github.com/aeroknight786/trialm/internal/openrouter"
	"github.com/aeroknight786/trialm/internal/store"
)

// standIn is the cloud model used for a trial of a model no provider hosts
// (fine-tunes, abliterated or community models): its base model. Speed is
// still simulated from the exact file; behaviour is the base model's.
type standIn struct {
	Base     string // e.g. "Qwen/Qwen3.6-35B-A3B"
	Upstream string // OpenRouter id
	Source   string // where the base came from
	Known    bool   // Upstream comes from the catalog (not a guessed id)
}

func (s standIn) Describe() string {
	if s.Upstream == "" || s.Base == "" {
		return ""
	}
	return fmt.Sprintf("%s (found in %s)", s.Base, s.Source)
}

// privateModels maps Hugging Face ids to privately hosted OpenRouter models
// (a variable so tests stay offline).
var privateModels = func(ctx context.Context) map[string]string {
	if os.Getenv("TRIALM_OFFLINE") == "1" {
		return nil
	}
	m, _ := openrouter.Client{}.PrivateModels(ctx, filepath.Join(store.Default().Dir, "cache"))
	return m
}

// FindStandIn finds the cloud model for a trial: the repo's own model
// ("Org/X-GGUF" is "Org/X"), else its base from the GGUF file's
// general.base_model.* keys or the Hugging Face model card chain, each
// matched to the catalog or to OpenRouter's privately hosted models; else
// the closest catalog model of the same architecture and size.
func FindStandIn(ref string, f *gguf.File, s *gguf.Summary) standIn {
	type cand struct{ repo, source string } // source "" = the exact model
	var cands []cand
	repo, isRepo := hub.RepoFromURL(ref)
	if isRepo && strings.HasSuffix(strings.ToLower(repo), "-gguf") {
		cands = append(cands, cand{repo: repo[:len(repo)-5]})
	}
	bases := 0
	if n, ok := f.Uint("general.base_model.count"); ok {
		for i := uint64(0); i < n && i < 4; i++ {
			k := fmt.Sprintf("general.base_model.%d.", i)
			if u, ok := f.String(k + "repo_url"); ok {
				if r, ok := hub.RepoFromURL(u); ok {
					cands, bases = append(cands, cand{r, "the GGUF file's metadata"}), bases+1
					continue
				}
			}
			org, _ := f.String(k + "organization")
			name, _ := f.String(k + "name")
			if org != "" && name != "" {
				cands, bases = append(cands, cand{strings.ReplaceAll(org, " ", "-") + "/" + strings.ReplaceAll(name, " ", "-"), "the GGUF file's metadata"}), bases+1
			}
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	var hosted map[string]string
	match := func(cands []cand) (standIn, bool) {
		for _, c := range cands {
			id, known := "", true
			if m, ok := catalog.Lookup(c.repo); ok && m.OpenRouterID != "" {
				id, known = m.OpenRouterID, !m.UpstreamUnverified
			} else {
				if hosted == nil {
					hosted = privateModels(ctx)
				}
				id = hosted[strings.ToLower(c.repo)]
			}
			if id == "" {
				continue
			}
			if c.source == "" {
				return standIn{Upstream: id, Known: known}, true // the exact model, not a stand-in
			}
			return standIn{Base: c.repo, Upstream: id, Source: c.source, Known: known}, true
		}
		return standIn{}, false
	}
	if si, ok := match(cands); ok {
		return si
	}
	if bases == 0 && isRepo {
		var chain []cand
		for _, r := range hub.BaseChain(ctx, repo, 4) {
			chain = append(chain, cand{r, "the Hugging Face model card"})
		}
		if si, ok := match(chain); ok {
			return si
		}
	}
	// Same architecture and roughly the same size.
	params := float64(s.ParameterCount) / 1e9
	var best *catalog.Model
	bestDiff := math.Inf(1)
	for _, m := range catalog.All() {
		if m.Family != s.Architecture || m.OpenRouterID == "" {
			continue
		}
		d := math.Abs(m.TotalParamsB-params) / params
		if d < 0.15 && d < bestDiff {
			mm := m
			best, bestDiff = &mm, d
		}
	}
	if best != nil {
		return standIn{Base: best.Name, Upstream: best.OpenRouterID, Source: "the catalog: same architecture (" + s.Architecture + ") and size — a guess", Known: !best.UpstreamUnverified}
	}
	return standIn{}
}

// ggufModelName derives a display name from a file or URL.
func ggufModelName(ref string) string {
	base := ref
	if u, err := url.Parse(ref); err == nil && u.Path != "" {
		base = u.Path
	}
	if i := strings.LastIndex(base, "/"); i >= 0 {
		base = base[i+1:]
	}
	return strings.TrimSuffix(base, ".gguf")
}
