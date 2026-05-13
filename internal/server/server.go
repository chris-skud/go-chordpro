// Package server hosts the chordpro web UI: a single-binary HTTP server with
// embedded HTML/CSS/JS that exposes a /api/render endpoint backed by the
// existing renderers.
package server

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"strconv"

	"github.com/chris-skud/go-chordpro/parser"
	"github.com/chris-skud/go-chordpro/render"
	htmlrender "github.com/chris-skud/go-chordpro/render/html"
	pdfrender "github.com/chris-skud/go-chordpro/render/pdf"
	textrender "github.com/chris-skud/go-chordpro/render/text"
)

//go:embed static
var staticFS embed.FS

// Config configures the server.
type Config struct {
	Addr string // host:port to listen on (default 127.0.0.1:8080)
}

// Run starts the server and blocks. Returns the first error from ListenAndServe.
func Run(cfg Config) error {
	if cfg.Addr == "" {
		cfg.Addr = "127.0.0.1:8080"
	}
	srv := &http.Server{Addr: cfg.Addr, Handler: Handler()}
	log.Printf("chordpro web UI: http://%s/", cfg.Addr)
	return srv.ListenAndServe()
}

// Handler returns the http.Handler tree for the server, exposed separately so
// tests can drive it without binding a real socket.
func Handler() http.Handler {
	mux := http.NewServeMux()

	sub, err := fs.Sub(staticFS, "static")
	if err != nil {
		// Embed paths are compile-time constants — failure here is a bug, not
		// a runtime error.
		panic(err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(sub))))
	mux.HandleFunc("/", indexHandler(sub))
	mux.HandleFunc("/api/render", renderHandler)
	return mux
}

// indexHandler serves the embedded index.html for the root path. Other paths
// 404 so we don't accidentally shadow assets.
func indexHandler(sub fs.FS) http.HandlerFunc {
	indexBytes, err := fs.ReadFile(sub, "index.html")
	if err != nil {
		panic(err)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(indexBytes)
	}
}

// renderHandler parses ChordPro source from the request body and writes the
// rendered output. Query parameters control format, transposition, and
// whether to attach the response as a download.
func renderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	q := r.URL.Query()
	format := q.Get("format")
	if format == "" {
		format = "html"
	}
	transpose := 0
	if t := q.Get("transpose"); t != "" {
		n, err := strconv.Atoi(t)
		if err != nil {
			http.Error(w, "transpose must be an integer", http.StatusBadRequest)
			return
		}
		transpose = n
	}
	download := q.Get("download") == "1"
	name := sanitizeName(q.Get("name"))

	src, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 5<<20)) // 5 MiB cap
	if err != nil {
		http.Error(w, "read body: "+err.Error(), http.StatusBadRequest)
		return
	}

	song, err := parser.Parse(bytes.NewReader(src))
	if err != nil {
		http.Error(w, "parse: "+err.Error(), http.StatusBadRequest)
		return
	}

	opts := render.Options{Transpose: transpose}
	var (
		renderer    render.Renderer
		contentType string
		ext         string
	)
	switch format {
	case "text", "txt":
		renderer = textrender.New(opts)
		contentType = "text/plain; charset=utf-8"
		ext = "txt"
	case "html":
		renderer = htmlrender.New(opts)
		contentType = "text/html; charset=utf-8"
		ext = "html"
	case "pdf":
		renderer = pdfrender.New(opts)
		contentType = "application/pdf"
		ext = "pdf"
	default:
		http.Error(w, fmt.Sprintf("unknown format %q (want text, html, or pdf)", format), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", contentType)
	if download {
		w.Header().Set("Content-Disposition",
			fmt.Sprintf(`attachment; filename=%q`, name+"."+ext))
	}
	if err := renderer.Render(w, song); err != nil {
		// Headers are already written; log and bail.
		log.Printf("render: %v", err)
	}
}

// sanitizeName strips path separators and characters that would be hostile in
// a Content-Disposition filename, and falls back to "song" when empty.
func sanitizeName(s string) string {
	if s == "" {
		return "song"
	}
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '/', '\\', '"', '\x00', '\r', '\n':
			// drop
		default:
			out = append(out, c)
		}
	}
	if len(out) == 0 {
		return "song"
	}
	return string(out)
}
