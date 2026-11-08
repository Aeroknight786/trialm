package openrouter

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type PKCE struct{ Verifier, Challenge string }

func NewPKCE() (PKCE, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return PKCE{}, err
	}
	v := base64.RawURLEncoding.EncodeToString(b)
	h := sha256.Sum256([]byte(v))
	return PKCE{v, base64.RawURLEncoding.EncodeToString(h[:])}, nil
}
func (p PKCE) URL(authURL, callback string) string {
	if authURL == "" {
		authURL = "https://openrouter.ai/auth"
	}
	v := url.Values{"callback_url": {callback}, "code_challenge": {p.Challenge}, "code_challenge_method": {"S256"}}
	return authURL + "?" + v.Encode()
}

// Callback has an unpredictable path as well as a random loopback port. Only
// the browser that received the authorization URL knows this path.
type Callback struct {
	URL      string
	codes    chan string
	srv      *http.Server
	listener net.Listener
}

func ListenCallback() (*Callback, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	b := make([]byte, 24)
	if _, err = rand.Read(b); err != nil {
		l.Close()
		return nil, err
	}
	path := "/callback/" + base64.RawURLEncoding.EncodeToString(b)
	c := &Callback{URL: "http://localhost:" + fmt.Sprint(l.Addr().(*net.TCPAddr).Port) + path, codes: make(chan string, 1), listener: l}
	mux := http.NewServeMux()
	mux.HandleFunc("GET "+path, func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if e := r.URL.Query().Get("error"); e != "" && code == "" {
			select {
			case c.codes <- errorPrefix + e:
			default:
			}
			fmt.Fprintln(w, "Sign-in stopped. Go back to your terminal to see why.")
			return
		}
		if code == "" {
			http.Error(w, "No authorization code.", 400)
			return
		}
		select {
		case c.codes <- code:
			fmt.Fprintln(w, "triaLM is signed in. You can close this tab.")
		default:
			http.Error(w, "Already received.", 409)
		}
	})
	c.srv = &http.Server{Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() { _ = c.srv.Serve(l) }()
	return c, nil
}
func (c *Callback) Close() { _ = c.srv.Close() }
func (c *Callback) Wait(ctx context.Context) (string, error) {
	select {
	case code := <-c.codes:
		if e, ok := strings.CutPrefix(code, errorPrefix); ok {
			return "", SignInError(e)
		}
		return code, nil
	case <-ctx.Done():
		return "", ctx.Err()
	}
}

const errorPrefix = "\x00error:"

// SignInError is the reason a sign-in page gave for stopping (for example
// "trial_used"), sent back to the callback as ?error=.
type SignInError string

func (e SignInError) Error() string { return "sign-in stopped: " + string(e) }

// Trial sign-in at mortiphi.com speaks the same PKCE contract as OpenRouter
// and hands back a $1 OpenRouter key funded by Mortiphi. TRIALM_CONNECT_URL
// points it at another site (a local build of it, in tests).
func TrialURLs() (authURL, api string) {
	base := "https://mortiphi.com"
	if v := strings.TrimRight(os.Getenv("TRIALM_CONNECT_URL"), "/"); v != "" {
		base = v
	}
	return base + "/trialm/connect", base + "/api/trialm"
}
