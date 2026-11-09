package core

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aeroknight786/trialm/internal/calib"
	"github.com/aeroknight786/trialm/internal/catalog"
	"github.com/aeroknight786/trialm/internal/engine"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/harness"
	"github.com/aeroknight786/trialm/internal/predict"
	"github.com/aeroknight786/trialm/internal/store"
	"os"
	"slices"
	"sort"
	"strings"
)

type ConfigCell struct {
	Row                         Row
	Context                     int
	Fits, PastNative, Measured  bool
	Speed, Empty, Memory, First figure.Figure
}
type ConfigFile struct {
	Name, LocalPath string
	Size            figure.Figure
	Cells           []ConfigCell
}
type ConfigGrid struct {
	MaxContext int
	Files      []ConfigFile
	Contexts   []int
	Folded     int
}
type ConfigModel struct {
	EmptyMeasured               bool
	MinContext                  int
	Default                     Row
	Engines                     []engine.Profile
	CacheTypes                  []string
	Grids                       map[string]ConfigGrid
	MaxContext                  int
	OnDisk                      bool
	File, Column, Engine, Cache int
}
type ConfigPicker struct {
	Models   []ConfigModel
	Target   predict.Target
	Harness  harness.Harness
	Scenario predict.Scenario
}

// Assumes says in a few words what an agent's time estimates assume.
func Assumes(tool string, tokens int) string {
	name := strings.TrimSuffix(harness.DisplayName(tool), " CLI")
	if tool == "generic" {
		name = "your agent"
	}
	return fmt.Sprintf("Assumes %s starts with %dK tokens; plugins add more", name, (tokens+500)/1000)
}

func gridKey(engine, cache string) string { return engine + "/" + cache }
func (m *ConfigModel) Grid() ConfigGrid {
	if len(m.Engines) == 0 {
		return ConfigGrid{}
	}
	return m.Grids[gridKey(m.Engines[m.Engine].ID, m.CacheTypes[m.Cache])]
}
func (m *ConfigModel) Cell() (ConfigCell, bool) {
	g := m.Grid()
	if m.File >= len(g.Files) || len(g.Files) == 0 || m.Column >= len(g.Contexts) {
		return ConfigCell{}, false
	}
	return g.Files[m.File].Cells[m.Column], true
}

// Select preserves the file across engines where possible, then snaps to the
// largest fitting context in that row, otherwise the smallest fitting file.
func (m *ConfigModel) Select(key string) {
	old := m.Grid()
	df, dc := 0, 0
	switch key {
	case "up":
		df = -1
	case "down":
		df = 1
	case "left":
		dc = -1
	case "right":
		dc = 1
	}
	if df != 0 || dc != 0 {
		for f, c := m.File+df, m.Column+dc; f >= 0 && f < len(old.Files) && c >= 0 && c < len(old.Contexts); f, c = f+df, c+dc {
			if old.Files[f].Cells[c].Fits {
				m.File, m.Column = f, c
				return
			}
		}
		return
	}
	name := ""
	if m.File < len(old.Files) {
		name = old.Files[m.File].Name
	}
	switch key {
	case "c":
		m.Cache = (m.Cache + 1) % len(m.CacheTypes)
	case "e":
		m.Engine = (m.Engine + 1) % len(m.Engines)
	}
	g := m.Grid()
	if key == "e" || key == "c" {
		for i, f := range g.Files {
			if f.Name == name {
				m.File = i
				break
			}
		}
	}
	m.File = min(m.File, max(0, len(g.Files)-1))
	m.Column = min(m.Column, max(0, len(g.Contexts)-1))
	if c, ok := m.Cell(); ok && c.Fits {
		return
	}
	snap := func(i int) bool {
		for j := len(g.Contexts) - 1; j >= 0; j-- {
			if g.Files[i].Cells[j].Fits {
				m.File, m.Column = i, j
				return true
			}
		}
		return false
	}
	if len(g.Files) > 0 && snap(m.File) {
		return
	}
	for i := len(g.Files) - 1; i >= 0; i-- {
		if snap(i) {
			return
		}
	}
}
func (m *ConfigModel) Reset() {
	m.Engine, m.Cache, m.File, m.Column = 0, 0, 0, 0
	for i, e := range m.Engines {
		if e.ID == m.Default.Result.Runtime.ID {
			m.Engine = i
		}
	}
	for i, c := range m.CacheTypes {
		if c == m.Default.Result.Settings.KVCacheType {
			m.Cache = i
		}
	}
	g := m.Grid()
	for i, f := range g.Files {
		if f.Name == m.Default.Result.Quant.Name {
			m.File = i
		}
	}
	for j, c := range g.Contexts {
		if c >= m.Default.Result.Settings.ContextTokens {
			m.Column = j
			break
		}
	}
	m.Select("")
	if cell, ok := m.Cell(); ok && cell.Context < m.MinContext {
		for ci := m.Cache; ci < len(m.CacheTypes); ci++ {
			m.Cache = ci
			g = m.Grid()
			for i := len(g.Files) - 1; i >= 0; i-- {
				for j, depth := range g.Contexts {
					if depth >= m.MinContext && g.Files[i].Cells[j].Fits {
						m.File, m.Column = i, j
						return
					}
				}
			}
		}
	}
}
func (s Session) Picker(ctx context.Context, d Detection, p store.Preferences, o Options, home string) (ConfigPicker, error) {
	h, e := harness.Get(p.Tool)
	if e != nil {
		return ConfigPicker{}, e
	}
	h.Scenario = harness.OpeningScenario(h.Scenario, d.Target.OS)
	o.Context, o.Use, o.Priority, o.QualityFloor = p.Context, p.Use, p.Priority, p.QualityFloor
	o.Runtime = "llamacpp"
	l, e := s.Find(ctx, o, d.Target, h.Scenario, h, "", 0)
	if e != nil {
		return ConfigPicker{}, e
	}
	// Include models whose minimum agent context fits only a smaller cache.
	seen := map[string]bool{}
	for _, r := range l.Rows {
		seen[r.Result.Model.ID] = true
	}
	for _, cache := range []string{"q8_0", "q4_0"} {
		extra := o
		extra.KV, extra.Context = cache, predict.MinContext(h.Scenario)
		more, err := s.Find(ctx, extra, d.Target, h.Scenario, h, "", 0)
		if err != nil {
			return ConfigPicker{}, err
		}
		for _, r := range more.Rows {
			if !seen[r.Result.Model.ID] {
				l.Rows = append(l.Rows, r)
				seen[r.Result.Model.ID] = true
			}
		}
	}
	for _, rt := range engine.All() {
		if rt.ID == "llamacpp" || !l.Installed.Has(rt.ID) {
			continue
		}
		for _, cache := range []string{"f16", "q8_0", "q4_0"} {
			extra := o
			extra.Runtime, extra.KV, extra.Context = rt.ID, cache, predict.MinContext(h.Scenario)
			more, err := s.Find(ctx, extra, d.Target, h.Scenario, h, "", 0)
			if err != nil {
				return ConfigPicker{}, err
			}
			for _, r := range more.Rows {
				if !seen[r.Result.Model.ID] {
					l.Rows = append(l.Rows, r)
					seen[r.Result.Model.ID] = true
				}
			}
		}
	}
	sort.SliceStable(l.Rows, func(i, j int) bool {
		a, b := l.Rows[i], l.Rows[j]
		if a.Result.Model.NotForCoding != b.Result.Model.NotForCoding && p.Use != "chat" {
			return b.Result.Model.NotForCoding
		}
		if p.Priority == "speed" {
			return a.Result.Scenario.Session < b.Result.Scenario.Session
		}
		if p.Priority == "quality" && a.Result.Model.TotalParamsB != b.Result.Model.TotalParamsB {
			return a.Result.Model.TotalParamsB > b.Result.Model.TotalParamsB
		}
		if p.Priority == "long" && a.Result.Settings.ContextTokens != b.Result.Settings.ContextTokens {
			return a.Result.Settings.ContextTokens > b.Result.Settings.ContextTokens
		}
		if a.Rank != b.Rank {
			return a.Rank < b.Rank
		}
		return a.Result.Model.TotalParamsB > b.Result.Model.TotalParamsB
	})
	out := ConfigPicker{Target: d.Target, Harness: h, Scenario: h.Scenario}
	c := CurrentCalibration(ctx, home)
	userHome, _ := os.UserHomeDir()
	for _, row := range l.Rows {
		if err := ctx.Err(); err != nil {
			return out, err
		}
		if p.Model != "" && p.Model != "any" && !queryMatches(row.Result.Model, p.Model) {
			continue
		}
		cm := ConfigModel{MinContext: predict.MinContext(h.Scenario), Default: row, CacheTypes: []string{"f16", "q8_0", "q4_0"}, Grids: map[string]ConfigGrid{}}
		for _, rt := range engine.All() { // setup installs llama.cpp and mlx-lm itself
			if rt.ID != "llamacpp" && rt.ID != "mlx-lm" && !l.Installed.Has(rt.ID) || !hasFile(row.Result.Model, rt) {
				continue
			}
			probe := predict.Predict(predict.Input{Model: row.Result.Model, Runtime: rt, Target: d.Target, Tuned: true, Scenario: h.Scenario})
			if probe.Unsupported == "" {
				cm.Engines = append(cm.Engines, rt)
			}
		}
		sort.SliceStable(cm.Engines, func(i, j int) bool { return cm.Engines[i].ID == "llamacpp" && cm.Engines[j].ID != "llamacpp" })
		local := map[string]string{}
		for _, q := range row.Result.Model.Quants {
			local[q.Name] = cachedModel(userHome, home, row.Result.Model, q)
			cm.OnDisk = cm.OnDisk || local[q.Name] != ""
		}
		engines := cm.Engines[:0]
		for _, rt := range cm.Engines {
			files := 0
			for _, cache := range cm.CacheTypes {
				g := BuildConfigGrid(row.Result.Model, rt, cache, d.Target, h.Scenario, local, c)
				cm.Grids[gridKey(rt.ID, cache)] = g
				cm.MaxContext = max(cm.MaxContext, g.MaxContext)
				files += len(g.Files)
			}
			if files > 0 { // an engine none of whose files fit is not offered
				engines = append(engines, rt)
			}
		}
		cm.Engines = engines
		if cm.MaxContext == 0 {
			continue
		}
		cm.Reset()
		if cell, ok := cm.Cell(); ok {
			cm.Default.Result = cell.Row.Result
			cm.Default.LocalPath = cell.Row.LocalPath
			cell.Context = 0
			cm.EmptyMeasured = measuredCell(cell, cell.Row.Result.Settings.KVCacheType, c)
		}
		out.Models = append(out.Models, cm)
	}
	return out, nil
}

// BuildConfigGrid uses exactly the predictor's memory and depth timing. Each
// immutable table is cached by model/runtime/cache for the life of a picker.
// hasFile reports whether the engine has a file it can set up (MLX files
// need a known repository).
func hasFile(m catalog.Model, rt engine.Profile) bool {
	return slices.ContainsFunc(m.Quants, func(q catalog.Quant) bool {
		return slices.Contains(rt.Formats, q.Format) && (q.Format != "mlx" || m.MLXRepo(q) != "")
	})
}

func BuildConfigGrid(m catalog.Model, rt engine.Profile, cache string, t predict.Target, sc predict.Scenario, local map[string]string, evidence *calib.Calibration) ConfigGrid {
	g := ConfigGrid{Contexts: []int{32768, 65536, 131072, 262144}}
	if m.NativeCtx > 262144 {
		g.Contexts = append(g.Contexts, m.NativeCtx)
	}
	quants := slices.Clone(m.Quants)
	sort.SliceStable(quants, func(i, j int) bool { return quants[i].FileBytes > quants[j].FileBytes })
	for _, q := range quants {
		if !slices.Contains(rt.Formats, q.Format) || q.Format == "mlx" && m.MLXRepo(q) == "" {
			continue
		}
		// Fold only files that cannot fit even the smallest column with any cache.
		smallest := predict.Predict(predict.Input{Model: m, QuantName: q.Name, Runtime: rt, Target: t, Tuned: true, CtxOverride: min(g.Contexts[0], m.NativeCtx), KVCacheType: "q4_0", Scenario: sc})
		if smallest.Unsupported != "" || smallest.Memory.TotalBytes > smallest.Memory.BudgetBytes {
			g.Folded++
			continue
		}
		f := ConfigFile{Name: q.Name, LocalPath: local[q.Name], Size: figure.Bytes(q.FileBytes, "catalog:"+m.ID)}
		for _, depth := range g.Contexts {
			r := predict.Predict(predict.Input{Model: m, QuantName: q.Name, Runtime: rt, Target: t, Tuned: true, CtxOverride: depth, KVCacheType: cache, Scenario: sc})
			c := ConfigCell{Row: Row{Result: r, LocalPath: local[q.Name]}, Context: depth, PastNative: depth > m.NativeCtx}
			c.Fits = !c.PastNative && r.Unsupported == "" && r.Memory.TotalBytes <= r.Memory.BudgetBytes
			c.Measured = measuredCell(c, cache, evidence)
			kind := figure.Simulated
			if c.Measured {
				kind = figure.Measured
			}
			c.Speed = figure.Band(0, r.Accuracy.DecodePct, "t/s", kind, "prediction:"+m.ID)
			if r.Timing != nil {
				c.Speed = figure.Band(r.Timing.DecodeTPS(depth), r.Accuracy.DecodePct, "t/s", kind, "prediction:"+m.ID)
			}
			c.Empty = PredictionMetric(r, r.DecodeTPS0, "t/s", false)
			c.Memory = figure.Exact(r.Memory.TotalBytes/(1<<30), "GB", figure.Simulated, "memory:predict")
			c.First = PredictionMetric(r, r.Scenario.ColdFirstReply.Seconds(), "s", true)
			f.Cells = append(f.Cells, c)
		}
		for _, cell := range f.Cells {
			if cell.Fits {
				g.MaxContext = max(g.MaxContext, cell.Context)
			}
		}
		g.Files = append(g.Files, f)
	}
	return g
}
func measuredCell(c ConfigCell, cache string, evidence *calib.Calibration) bool {
	r := c.Row.Result
	// Legacy real measurements establish f16; newer rows name their cache.
	if evidence == nil || r.Runtime.ID != "llamacpp" || c.Row.LocalPath == "" || !c.Fits {
		return false
	}
	for _, s := range evidence.Rows {
		if s.Probe != "real" || s.Quant != r.Quant.Name || s.Seconds <= 0 {
			continue
		}
		var tag struct{ Model, Engine, Mode, Cache string }
		_ = json.Unmarshal(s.Raw, &tag)
		if tag.Cache == "" {
			tag.Cache = "f16"
		}
		if tag.Cache == cache && tag.Model == r.Model.ID && tag.Engine == "llamacpp" && strings.EqualFold(tag.Mode, "decode") && abs(s.Depth-c.Context) <= max(1024, c.Context/10) {
			return true
		}
	}
	return false
}
func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// SelectOnDisk is used only for the first local shortcut. Model navigation
// still resets to the recommender's default via Reset.
func (m *ConfigModel) SelectOnDisk() bool {
	original := m.Cache
	for n := range m.CacheTypes {
		m.Cache = (original + n) % len(m.CacheTypes)
		g := m.Grid()
		for i, f := range g.Files {
			if f.LocalPath == "" {
				continue
			}
			for offset := range g.Contexts {
				j := (m.Column + offset) % len(g.Contexts)
				depth := g.Contexts[j]
				if depth >= m.MinContext && f.Cells[j].Fits {
					m.File, m.Column = i, j
					return true
				}
			}
		}
	}
	m.Cache = original
	return false
}
