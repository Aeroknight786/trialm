package hw

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

const serverPS = `  PID   RSS COMMAND
11 7340032 /opt/homebrew/bin/llama-server -m /models/9b.gguf
12 6815744 /other/trialm/engines/llama-server -m /other/9b.gguf
13 2097152 /usr/bin/ollama runner --model /models/sha256-abc --port 1234
14 32000 /usr/bin/ollama serve
15 3145728 /Applications/LM Studio.app/Contents/Resources/llama-server -m /models/a.gguf
16 1048576 /venv/bin/python3 -m mlx_lm.server --model Qwen/9B
17 524288 /home/user/.lmstudio/extensions/llm-engine --model /models/b.gguf
18 10 ps -axo pid,rss,args
19 100 vim llama-server
20 100 zsh -c 'ollama runner --model abc'
21 100 /Applications/LM Studio.app/Contents/MacOS/LM Studio
22 0 llama-server -m dead.gguf
bad output
23 nope llama-server
24 100 python /tmp/check.py mlx_lm.server
25 100 /usr/lib/ollama/runners/cpu/ollama_llama_server --model /models/blob
`

func TestModelServersSnapshot(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		t.Run(goos, func(t *testing.T) {
			home := t.TempDir()
			r := &fakeRunner{cmds: map[string]fakeOut{"ps -axo pid,rss,args": {out: serverPS}}, files: map[string]string{
				filepath.Join(home, "run", "owned.pid"):  `{"pid":11,"argv":["/opt/homebrew/bin/llama-server","-m","/models/9b.gguf"]}`,
				filepath.Join(home, "run", "stale.pid"):  `{"pid":12,"argv":["llama-server","-m","/old/model.gguf"]}`,
				filepath.Join(home, "run", "broken.pid"): `bad json`,
			}}
			got := modelServers(context.Background(), home, goos, r)
			want := 6
			if goos == "windows" {
				want = 0
			}
			if len(got) != want {
				t.Fatalf("%+v", got)
			}
			calls := 0
			for _, call := range r.calls {
				if strings.HasPrefix(call, "run ") {
					calls++
				}
			}
			if goos == "windows" {
				if calls != 0 {
					t.Fatal(r.calls)
				}
				return
			}
			if calls != 1 || got[0].PID != 12 || got[0].Model != "9b" || got[1].Model != "" || got[3].Model != "Qwen/9B" || got[2].Name != "LM Studio" || got[3].Name != "mlx-lm" {
				t.Fatal(got, r.calls)
			}
		})
	}
}

func TestModelServersUnavailable(t *testing.T) {
	r := &fakeRunner{cmds: map[string]fakeOut{"ps": {out: serverPS, err: errors.New("denied")}}}
	if got := modelServers(context.Background(), t.TempDir(), "darwin", r); len(got) != 0 {
		t.Fatal(got)
	}
}
