package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed static/*
var embeddedWeb embed.FS

func newFrontendFS() http.FileSystem {
	sub, err := fs.Sub(embeddedWeb, "static")
	if err != nil {
		return http.FS(embeddedWeb)
	}
	return http.FS(sub)
}

// serveFrontend returns a handler that serves embedded frontend assets with SPA fallback.
func serveFrontend(frontendFS http.FileSystem) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Try exact file first
		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" {
			requestPath = "index.html"
		}

		tryServeFile := func(p string) bool {
			f, err := frontendFS.Open(p)
			if err != nil {
				return false
			}
			defer f.Close()
			info, err := f.Stat()
			if err != nil || info.IsDir() {
				return false
			}
			http.ServeContent(w, r, info.Name(), info.ModTime(), f)
			return true
		}

		if tryServeFile(requestPath) {
			return
		}

		if placeholder := mapDynamicToPlaceholder(requestPath); placeholder != "" {
			if tryServeFile(placeholder) {
				return
			}
		}

		// Try with index.html fallback for SPA routes
		if tryServeFile("index.html") {
			return
		}

		http.NotFound(w, r)
	}
}

// mapDynamicToPlaceholder maps dynamic routes to prerendered placeholder HTML produced by static export.
func mapDynamicToPlaceholder(p string) string {
	if strings.HasPrefix(p, "instances/") {
		if strings.HasSuffix(p, "/console") || strings.Contains(p, "/console/") {
			return "instances/placeholder-node/placeholder-id/console.html"
		}
		return "instances/placeholder-node/placeholder-id.html"
	}
	if strings.HasPrefix(p, "nodes/") {
		return "nodes/placeholder-node.html"
	}
	if strings.HasPrefix(p, "storage-pools/") {
		return "storage-pools/placeholder.html"
	}
	return ""
}

// mountFrontend registers handlers for embedded assets.
func (a *API) mountFrontend() {
	a.frontendFS = newFrontendFS()

	// Static assets
	a.engine.StaticFS("/_next", a.frontendFS)
	a.engine.StaticFS("/static", a.frontendFS)

	// SPA fallback
	a.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
		serveFrontend(a.frontendFS).ServeHTTP(c.Writer, c.Request)
	})
}
