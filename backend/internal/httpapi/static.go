package httpapi

import (
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"strings"
)

// Option configures NewHandler.
type Option func(*options)

type options struct {
	staticDir string
}

// WithStaticDir serves the built SPA from dir: existing files as-is, any other
// GET/HEAD as index.html. /healthz and /api/ keep precedence.
func WithStaticDir(dir string) Option {
	return func(o *options) { o.staticDir = dir }
}

// staticHandler serves files from dir with an index.html fallback for client-side routes.
func staticHandler(dir string) http.Handler {
	fsys := os.DirFS(dir)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			methodNotAllowed(w, "GET, HEAD")
			return
		}
		// Clean against "/" so ".." can never climb out of dir.
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if info, err := fs.Stat(fsys, name); name == "" || err != nil || info.IsDir() {
			name = "index.html"
		}
		f, err := fsys.Open(name)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		info, err := f.Stat()
		rs, ok := f.(io.ReadSeeker)
		if err != nil || !ok {
			http.NotFound(w, r)
			return
		}
		// ServeContent, unlike FileServer, does not redirect /index.html to /.
		http.ServeContent(w, r, name, info.ModTime(), rs)
	})
}
