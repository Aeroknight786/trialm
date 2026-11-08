package ui

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/aeroknight786/trialm/internal/core"
	"github.com/aeroknight786/trialm/internal/figure"
	"github.com/aeroknight786/trialm/internal/openrouter"
	"github.com/aeroknight786/trialm/internal/signin"
	"github.com/aeroknight786/trialm/internal/store"
	"io"
	"net/url"
	"strings"
	"time"
)

type KeyFlow struct {
	Store    store.Store
	Client   openrouter.Client
	AuthURL  string
	Open     func(string) error
	Secret   func(string) (string, error)
	Headless bool
	// Trial makes the free trial sign-in at mortiphi.com the default;
	// --own then uses the user's own OpenRouter account.
	Trial bool

	Output io.Writer
	Emit   func(string, any, ...core.Next) error
	Needs  func(string, string) error
}

func (f KeyFlow) Run(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("key", flag.ContinueOnError)
	paste := fs.Bool("paste", false, "paste a key into a masked terminal input")
	remove := fs.Bool("remove", false, "remove stored keys")
	headless := fs.Bool("headless", f.Headless, "copy an authorization code in a terminal")
	own := fs.Bool("own", false, "use your own OpenRouter account instead of the free trial sign-in")
	fs.Bool("json", false, "NDJSON output")
	err := fs.Parse(args)
	pos := fs.Args()
	if err != nil {
		return err
	}
	if len(pos) > 0 {
		return errors.New("usage: trialm key [--own] [--paste|--headless|--remove]")
	}
	if *paste && *remove {
		return errors.New("usage: choose --paste or --remove")
	}
	if *remove {
		if err := f.Store.RemoveKey(); err != nil {
			return err
		}
		fmt.Fprintln(f.Output, "Stored OpenRouter key removed. OPENROUTER_API_KEY still takes precedence when set.")
		return f.Emit("result", map[string]any{"removed": true}, core.Next{Cmd: "trialm key", Why: "sign in again"})
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	key, token := f.Store.Key(), ""
	trial := f.Trial && !*own && !*paste
	authURL, client := f.AuthURL, f.Client
	if trial {
		authURL, client.BaseURL = openrouter.TrialURLs()
	}
	if *paste {
		key, err = f.Secret("OpenRouter key: ")
		if err != nil {
			return err
		}
		if strings.TrimSpace(key) == "" {
			return f.Needs("--paste", "no key was entered")
		}
	} else if key == "" {
		pkce, err := openrouter.NewPKCE()
		if err != nil {
			return err
		}
		callback, err := openrouter.ListenCallback()
		if err != nil {
			return f.Needs("--paste", "cannot bind a localhost OAuth callback")
		}
		defer callback.Close()
		auth := pkce.URL(authURL, callback.URL)
		// This URL contains the public challenge, never a key, code or verifier.
		if err := f.Emit("needs_user", map[string]any{"needs": "authorization", "url": auth, "run": "trialm key --headless"}); err != nil {
			return err
		}
		var code string
		if *headless {
			fmt.Fprintln(f.Output, "Open this URL on a browser, authorize triaLM, then copy the code from the redirect URL:", auth)
			code, err = f.Secret("Authorization code: ")
			if err != nil {
				return err
			}
			if u, e := url.Parse(code); e == nil && u.Query().Get("code") != "" {
				code = u.Query().Get("code")
			}
		} else {
			if err = f.Open(auth); err != nil {
				return f.Needs("--headless or --paste", "cannot open a browser")
			}
			fmt.Fprintln(f.Output, map[bool]string{true: "Sign in with GitHub or Google in your browser.", false: "Authorize triaLM in your browser."}[trial])
			code, err = callback.Wait(ctx)
		}
		if err != nil && trial {
			return errors.New(signin.Explain(err, true, "run `trialm key --own`"))
		}
		if err != nil {
			return err
		}
		g, err := client.Grant(ctx, strings.TrimSpace(code), pkce.Verifier)
		if err != nil {
			return err
		}
		key, token = g.Key, g.Token
	}
	if strings.TrimSpace(key) == "" {
		return f.Needs("--paste", "no OpenRouter key")
	}
	c := f.Client
	c.Key = key
	credits := c.Credits
	if trial {
		c.BaseURL = "" // the key is OpenRouter's, wherever it came from
		credits = c.NewKeyCredits
	}
	balance, err := credits(ctx)
	if err != nil {
		return err
	}
	if *paste || f.Store.Key() == "" {
		if err = f.Store.SaveKey(key); err != nil {
			return err
		}
	}
	if token != "" {
		if err = f.Store.Write("trial.json", map[string]string{"token": token}); err != nil {
			return err
		}
	}
	credit := figure.Exact(balance, "USD", figure.Measured, "openrouter:/credits")
	fmt.Fprintln(f.Output, map[bool]string{true: "Signed in · $" + fmt.Sprintf("%.2f", balance) + " of free trial credit", false: "OpenRouter ready · " + credit.String() + " credit"}[trial])
	return f.Emit("result", map[string]any{"key_configured": true, "balance": credit}, core.Next{Cmd: "trialm find --json", Why: "find a model"})
}
