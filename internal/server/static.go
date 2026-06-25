package server

import (
	"io/fs"
	"net/http"
	"path"
	"strings"
)

// spaHandler serves the embedded frontend, falling back to index.html for client-side
// routes (anything that isn't an existing asset and isn't an /api path).
func (s *Server) spaHandler() http.Handler {
	fileServer := http.FileServer(http.FS(s.webFS))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
		if upath == "" {
			upath = "index.html"
		}

		if f, err := s.webFS.Open(upath); err == nil {
			_ = f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Unknown path → serve index.html so the SPA router can handle it.
		serveIndex(w, r, s.webFS)
	})
}

func serveIndex(w http.ResponseWriter, r *http.Request, fsys fs.FS) {
	data, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		http.Error(w, "frontend not built", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(data)
}
