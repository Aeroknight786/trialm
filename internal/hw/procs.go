package hw

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

type ModelServer struct {
	PID         int
	Name        string
	MemoryBytes uint64
}

// ModelServers takes one process snapshot. PID records under this home's run
// directory identify its own servers. Failure to inspect never blocks setup.
func ModelServers(ctx context.Context, home string) []ModelServer {
	return modelServers(ctx, home, runtime.GOOS, osRunner{})
}

func modelServers(ctx context.Context, home, goos string, r runner) []ModelServer {
	if goos != "darwin" && goos != "linux" {
		return nil
	}
	b, err := r.Run(ctx, "ps", "-axo", "pid,rss,args")
	if err != nil {
		return nil
	}
	owned := map[int][]string{}
	paths, _ := r.Glob(filepath.Join(home, "run", "*.pid"))
	for _, path := range paths {
		data, _ := r.ReadFile(path)
		var record struct {
			PID  int
			Argv []string
		}
		if json.Unmarshal(data, &record) == nil && len(record.Argv) > 0 {
			owned[record.PID] = record.Argv
		}
	}
	return parseModelServers(string(b), owned)
}

func parseModelServers(output string, owned map[int][]string) []ModelServer {
	var found []ModelServer
	for _, line := range strings.Split(output, "\n") {
		f := strings.Fields(line)
		if len(f) < 3 {
			continue
		}
		pid, err := strconv.Atoi(f[0])
		rss, rssErr := strconv.ParseUint(f[1], 10, 64)
		if err != nil || rssErr != nil || pid <= 0 || rss == 0 {
			continue
		}
		args := strings.Join(f[2:], " ")
		matches := len(owned[pid]) > 0
		for _, arg := range owned[pid] {
			matches = matches && strings.Contains(args, arg)
		}
		if matches {
			continue
		}
		name := ""
		exe := filepath.Base(f[2])
		switch {
		case exe == "mlx_lm.server" || (strings.HasPrefix(exe, "python") && (strings.Contains(args, " -m mlx_lm.server") || strings.Contains(args, "/mlx_lm.server"))):
			name = "mlx-lm"
		case exe == "llama-server" || (strings.Contains(args, "LM Studio.app/") && strings.Contains(args, "/llama-server")):
			name = "llama-server"
			if strings.Contains(args, "LM Studio") || strings.Contains(args, ".lmstudio/") {
				name = "LM Studio"
			}
		case (exe == "ollama" || strings.Contains(f[2], "/ollama/runners/")) && (strings.Contains(args, " runner ") || strings.Contains(args, "/runners/")) && strings.Contains(args, "--model"):
			name = "Ollama"
		case (strings.Contains(f[2], ".lmstudio/") || strings.Contains(args, "LM Studio.app/")) && strings.Contains(args, "--model"):
			name = "LM Studio"
		}
		if name != "" {
			found = append(found, ModelServer{PID: pid, Name: name, MemoryBytes: rss * 1024})
		}
	}
	return found
}
