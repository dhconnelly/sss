package build

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

var tmpl = template.Must(template.New("test").Parse(`
<html>
	<head><title>{{.Title}}</title></head>
	<body>{{.Content}}</body>
</html>
`))

func shallowSliceEq[T comparable](a, b []T) bool {
	if len(a) != len(b) {
		return false
	}
	for i, x := range a {
		if x != b[i] {
			return false
		}
	}
	return true
}

func collectPaths(t *testing.T, dir string) ([]string, error) {
	t.Helper()
	var dstPaths []string
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			dstPaths = append(dstPaths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dstPaths, nil
}

func check(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestBuildTreeInvalid(t *testing.T) {
	if err := BuildTree(t.TempDir(), "blah", tmpl); err == nil {
		t.Fatal("expected error on nonexistent source directory")
	}
}

func TestBuildTreeEmpty(t *testing.T) {
	dst := t.TempDir()
	src := t.TempDir()
	if err := BuildTree(dst, src, tmpl); err != nil {
		t.Fatal(err)
	}
	filepath.WalkDir(dst, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			t.Fatal(err)
		}
		if path != dst {
			t.Fatal("no files expected!")
		}
		return nil
	})
}

func TestBuildTree(t *testing.T) {
	// create the site source
	src := t.TempDir()
	html := "<html><body>Foo</body></html>"
	check(t, os.WriteFile(path.Join(src, "a.html"), []byte(html), 0750))
	md := "=== title ===\n# hello\n"
	check(t, os.WriteFile(path.Join(src, "b.md"), []byte(md), 0750))
	check(t, os.Mkdir(path.Join(src, "foo"), 0750))
	check(t, os.WriteFile(path.Join(src, "foo/c.md"), []byte(md), 0750))

	// build the site
	dst := t.TempDir()
	if err := BuildTree(dst, src, tmpl); err != nil {
		t.Fatal(err)
	}

	// verify the files were all built in the right places
	dstPaths, err := collectPaths(t, dst)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a.html", "b.html", "foo/c.html"}
	for i := range want {
		want[i] = path.Join(dst, want[i])
	}
	if !shallowSliceEq(dstPaths, want) {
		t.Fatalf("want paths: %v, got %v", want, dstPaths)
	}

	// only verify that the files are non-empty: build logic is validated
	// in a different case
	for _, path := range want {
		if bs, err := os.ReadFile(path); err != nil {
			t.Fatal(err)
		} else if len(bs) == 0 {
			t.Fatalf("built file is empty: %s", path)
		}
	}
}

func TestBuildFileHTML(t *testing.T) {
	// create an html file
	dir := t.TempDir()
	srcPath, dstPath := path.Join(dir, "src.html"), path.Join(dir, "dest.html")
	html := "<html><body>foo</body></html>"
	if err := os.WriteFile(srcPath, []byte(html), 0750); err != nil {
		t.Fatal(err)
	}

	// create the source and destination files
	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	// build
	if err := BuildFile(srcPath, dst, src, tmpl); err != nil {
		t.Fatal(err)
	}
	check(t, src.Close())
	check(t, dst.Close())

	// verify the contents were unmodified
	got, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(got); got != html {
		t.Fatalf("want %s, got %s", html, got)
	}
}

func TestBuildFileMarkdownMissingTitle(t *testing.T) {
	// create a markdown file
	dir := t.TempDir()
	srcPath, dstPath := path.Join(dir, "src.md"), path.Join(dir, "dest.md")
	md := "# bar\n"
	if err := os.WriteFile(srcPath, []byte(md), 0750); err != nil {
		t.Fatal(err)
	}

	// create the source and destination files
	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	if err := BuildFile(srcPath, dst, src, tmpl); err == nil {
		t.Fatal("expected missing title error")
	}
}

func TestBuildFileMarkdown(t *testing.T) {
	// create a markdown file
	dir := t.TempDir()
	srcPath, dstPath := path.Join(dir, "src.md"), path.Join(dir, "dest.md")
	md := "=== foo ===\n# bar\n"
	if err := os.WriteFile(srcPath, []byte(md), 0750); err != nil {
		t.Fatal(err)
	}

	// create the source and destination files
	src, err := os.Open(srcPath)
	if err != nil {
		t.Fatal(err)
	}
	dst, err := os.Create(dstPath)
	if err != nil {
		t.Fatal(err)
	}

	// build
	if err := BuildFile(srcPath, dst, src, tmpl); err != nil {
		t.Fatal(err)
	}
	check(t, src.Close())
	check(t, dst.Close())

	// verify the contents were rendered
	gotb, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatal(err)
	}
	got := string(gotb)
	wantTitle := "<title>foo</title>"
	if !strings.Contains(got, wantTitle) {
		t.Fatalf("%s not found in %s", wantTitle, got)
	}
	wantH1 := `<h1 id="bar">bar</h1>`
	if !strings.Contains(got, wantH1) {
		t.Fatalf("%s not found in %s", wantH1, got)
	}
}
