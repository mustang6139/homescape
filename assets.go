// Package homescape exposes the embedded, built frontend assets. The embed directive must
// live at module root because go:embed cannot reach across parent directories.
package homescape

import (
	"embed"
	"io/fs"
)

//go:embed all:web/dist
var distFS embed.FS

// WebFS returns a filesystem rooted at the built frontend (web/dist contents at top level).
func WebFS() fs.FS {
	sub, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		panic("homescape: cannot create web/dist sub FS: " + err.Error())
	}
	return sub
}
