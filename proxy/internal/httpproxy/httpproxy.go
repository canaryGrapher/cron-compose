// Package httpproxy builds the HTTP reverse-proxy handler that path-routes
// between the web UI and the REST API behind the single entry point.
package httpproxy

import (
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/croncompose/croncompose/proxy/internal/config"
)

// New returns an http.Handler with three routes behind the single entry point:
//
//   - APIPrefix (default /api/*) -> control plane, prefix rewritten to
//     APIUpstreamPrefix (default /api/v1).
//   - WebPrefix (default /app/*) -> the Next.js app, path passed through
//     unchanged (Next serves under the matching basePath).
//   - anything else, including bare "/" -> 302 into WebPrefix, so visiting the
//     domain lands on the app. When WebPrefix is "" the UI is served at the
//     root and nothing is redirected.
//
// A tiny /__health endpoint answers locally so the container can be probed.
func New(cfg config.Config) (http.Handler, error) {
	webURL, err := url.Parse(cfg.WebUpstream)
	if err != nil {
		return nil, err
	}
	apiURL, err := url.Parse(cfg.APIUpstream)
	if err != nil {
		return nil, err
	}

	rp := &httputil.ReverseProxy{
		// A negative interval flushes after every write, keeping the run-log SSE
		// stream (text/event-stream) responsive when proxied.
		FlushInterval: -1,
		Rewrite: func(pr *httputil.ProxyRequest) {
			path := pr.In.URL.Path
			if isAPIPath(path, cfg.APIPrefix) {
				setTarget(pr, apiURL)
				rest := strings.TrimPrefix(path, cfg.APIPrefix)
				pr.Out.URL.Path = singleJoin(cfg.APIUpstreamPrefix, rest)
			} else {
				setTarget(pr, webURL)
				pr.Out.URL.Path = path
			}
			pr.Out.URL.RawQuery = pr.In.URL.RawQuery
			pr.SetXForwarded()
			pr.Out.Host = pr.Out.URL.Host
		},
		ErrorHandler: func(w http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(w, "upstream unavailable", http.StatusBadGateway)
		},
	}

	webPrefix := cfg.WebPrefix // already trimmed of any trailing slash

	root := func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case isAPIPath(path, cfg.APIPrefix):
			rp.ServeHTTP(w, r) // -> control plane
		case webPrefix == "":
			rp.ServeHTTP(w, r) // UI served at the root (legacy)
		case path == webPrefix || strings.HasPrefix(path, webPrefix+"/"):
			rp.ServeHTTP(w, r) // -> web, unchanged
		default:
			// Bare "/" or a stray root path: send it into the app, keeping the
			// path and query so old links resolve (e.g. /jobs -> /app/jobs).
			target := webPrefix + path
			if r.URL.RawQuery != "" {
				target += "?" + r.URL.RawQuery
			}
			http.Redirect(w, r, target, http.StatusFound)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/__health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/", root)
	return mux, nil
}

func setTarget(pr *httputil.ProxyRequest, u *url.URL) {
	pr.Out.URL.Scheme = u.Scheme
	pr.Out.URL.Host = u.Host
}

func isAPIPath(path, prefix string) bool {
	return path == prefix || strings.HasPrefix(path, prefix+"/")
}

// singleJoin concatenates two URL path segments with exactly one slash.
func singleJoin(a, b string) string {
	if b == "" {
		return a
	}
	return strings.TrimSuffix(a, "/") + "/" + strings.TrimPrefix(b, "/")
}
