package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(Handler())
}

func TestIndexServed(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("content-type = %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	for _, want := range []string{"<title>chordpro</title>", `id="source"`, `id="preview"`} {
		if !bytes.Contains(body, []byte(want)) {
			t.Errorf("missing %q in index", want)
		}
	}
}

func TestStaticAssetsServed(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	for _, path := range []string{"/static/app.js", "/static/app.css"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != 200 {
			t.Errorf("%s -> status %d", path, resp.StatusCode)
		}
	}
}

func TestUINotCached(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	for _, path := range []string{"/", "/static/app.js", "/static/app.css"} {
		resp, err := http.Get(ts.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if cc := resp.Header.Get("Cache-Control"); cc != "no-cache" {
			t.Errorf("%s -> Cache-Control %q, want no-cache", path, cc)
		}
	}
}

func TestRenderHTML(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?format=html", "text/plain",
		strings.NewReader("{title: Hi}\n[C]hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("<title>Hi</title>")) {
		t.Errorf("html render missing title:\n%s", body)
	}
	if !bytes.Contains(body, []byte(`class="chord"`)) {
		t.Errorf("html render missing chord span:\n%s", body)
	}
}

func TestRenderText(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?format=text", "text/plain",
		strings.NewReader("{title: Hi}\nhello"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/plain") {
		t.Errorf("content-type = %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte("Hi")) {
		t.Errorf("text render missing title:\n%s", body)
	}
}

func TestRenderPDF(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?format=pdf", "text/plain",
		strings.NewReader("{title: Hi}\n[C]hello"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("content-type = %q", ct)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.HasPrefix(body, []byte("%PDF-")) {
		t.Errorf("not a pdf: %q", body[:8])
	}
}

func TestRenderTranspose(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?format=html&transpose=2", "text/plain",
		strings.NewReader("[C]hi [G]there"))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`>D<`)) {
		t.Errorf("expected transposed D in output:\n%s", body)
	}
}

func TestDownloadDisposition(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?format=pdf&download=1&name=mysong", "text/plain",
		strings.NewReader("hi"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "mysong.pdf") {
		t.Errorf("content-disposition = %q", cd)
	}
}

func TestDownloadNameSanitized(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+`/api/render?format=text&download=1&name=../etc/passwd`, "text/plain",
		strings.NewReader("hi"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	cd := resp.Header.Get("Content-Disposition")
	// Path separators are the actual concern in Content-Disposition.
	if strings.ContainsAny(cd, `/\`) {
		t.Errorf("disposition leaked path separator: %q", cd)
	}
}

func TestBadFormat(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?format=latex", "text/plain", strings.NewReader("hi"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}

func TestRenderMethodNotAllowed(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Get(ts.URL + "/api/render")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d", resp.StatusCode)
	}
}

func TestBadTransposeRejected(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()
	resp, err := http.Post(ts.URL+"/api/render?transpose=foo", "text/plain", strings.NewReader("hi"))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
}
