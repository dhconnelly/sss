package serve

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func readBuffer(t *testing.T, buf *bytes.Buffer) string {
	t.Helper()
	b, err := io.ReadAll(buf)
	if err != nil {
		t.Fatalf("failed to read buffer: %s", err)
		return ""
	}
	return string(b)
}

func simpleFile(content string) *fstest.MapFile {
	return &fstest.MapFile{Data: []byte(content), Mode: 0444}
}

func TestServe(t *testing.T) {
	fs := fstest.MapFS(map[string]*fstest.MapFile{
		"index.html":       simpleFile("index"),
		"foo.html":         simpleFile("foo"),
		"foo/bar/baz.html": simpleFile("baz"),
	})
	handler := NewHandler(http.FS(fs))

	for _, test := range []struct {
		descr            string
		method           string
		path             string
		data             string
		expectedCode     int
		expectedContains string
	}{
		{"success", http.MethodGet, "/foo.html", "", http.StatusOK, "foo"},
		{"nonexistent file", http.MethodGet, "/bar.html", "", http.StatusNotFound, ""},
		{"/ routes to index", http.MethodGet, "/", "", http.StatusOK, "index"},
		{"nested file", http.MethodGet, "/foo/bar/baz.html", "", http.StatusOK, "baz"},
		{"only GET allowed", http.MethodPost, "/foo.html", "", http.StatusMethodNotAllowed, ""},
	} {
		t.Run(test.descr, func(t *testing.T) {
			req := httptest.NewRequest(
				test.method, test.path, bytes.NewReader([]byte(test.data)))
			resp := httptest.NewRecorder()
			handler.ServeHTTP(resp, req)

			if test.expectedCode != resp.Code {
				t.Errorf("expected %d, got %d", test.expectedCode, resp.Code)
			}

			respData := readBuffer(t, resp.Body)
			if strings.Index(respData, test.expectedContains) < 0 {
				t.Errorf("expected to find %s in input, got %s", test.expectedContains, respData)
			}
		})
	}
}
