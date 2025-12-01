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
		// Try to serve the requested file
		requestPath := strings.TrimPrefix(r.URL.Path, "/")
		if requestPath == "" {
			requestPath = "index.html"
		}

		// Try to open and serve the file
		f, err := frontendFS.Open(requestPath)
		if err == nil {
			defer f.Close()
			info, err := f.Stat()
			if err == nil && !info.IsDir() {
				http.ServeContent(w, r, info.Name(), info.ModTime(), f)
				return
			}
		}

		// Fallback to index.html for SPA routes
		f, err = frontendFS.Open("index.html")
		if err == nil {
			defer f.Close()
			info, err := f.Stat()
			if err == nil && !info.IsDir() {
				http.ServeContent(w, r, info.Name(), info.ModTime(), f)
				return
			}
		}

		http.NotFound(w, r)
	}
}

// mountFrontend registers handlers for embedded assets.
func (a *API) mountFrontend() {
	a.frontendFS = newFrontendFS()

	// Static assets (Vite uses /assets for JS/CSS files)
	a.engine.StaticFS("/assets", a.frontendFS)
	a.engine.StaticFS("/static", a.frontendFS)

	// SPA fallback: all non-API routes serve index.html
	a.engine.NoRoute(func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Not Found"})
			return
		}
		serveFrontend(a.frontendFS).ServeHTTP(c.Writer, c.Request)
	})
}
