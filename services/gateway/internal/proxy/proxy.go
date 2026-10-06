package proxy

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/planly/pkg/httpx"
)

// NewReverseProxy creates a reverse proxy that strips `/api/v1` and forwards requests to targetURL.
func NewReverseProxy(targetURL string) (http.Handler, error) {
	target, err := url.Parse(targetURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)

		// Set upstream target host
		req.Host = target.Host

		// Strip /api/v1 prefix
		path := req.URL.Path
		if strings.HasPrefix(path, "/api/v1") {
			path = strings.TrimPrefix(path, "/api/v1")
			if path == "" {
				path = "/"
			}
			req.URL.Path = path
		}
	}

	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		slog.Error("reverse proxy upstream error",
			"target", targetURL,
			"path", r.URL.Path,
			"error", err,
		)
		httpx.WriteError(w, http.StatusBadGateway, "bad_gateway", "Upstream service is currently unavailable")
	}

	return proxy, nil
}
