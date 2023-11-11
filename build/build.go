package build

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/russross/blackfriday/v2"
)

var titlePat = regexp.MustCompile(`^=== ([^=]+) ===$`)

func readTitle(r *bufio.Reader) (string, error) {
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("error reading title: %w", err)
	}
	ms := titlePat.FindStringSubmatch(line[:len(line)-1])
	if len(ms) != 2 {
		return "", errors.New("title is missing")
	}
	return ms[1], nil
}

type Page struct {
	Title   string
	Content string
}

const extensions = blackfriday.CommonExtensions | blackfriday.AutoHeadingIDs

func buildMarkdown(dst io.Writer, src io.Reader, tmpl *template.Template) error {
	r := bufio.NewReader(src)
	w := bufio.NewWriter(dst)

	// process the title
	title, err := readTitle(r)
	if err != nil {
		return fmt.Errorf("error building markdown: %w", err)
	}

	// process the content
	input, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("error reading markdown source: %w", err)
	}
	output := blackfriday.Run(input,
		blackfriday.WithNoExtensions(),
		blackfriday.WithExtensions(extensions))

	// process the post template
	if err := tmpl.Execute(w, Page{title, string(output)}); err != nil {
		return fmt.Errorf("error rendering post template: %w", err)
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("error writing to destination: %w", err)
	}
	return nil
}

func BuildFile(
	path string, dst io.Writer, src io.Reader, tmpl *template.Template,
) error {
	log.Printf("building file: %s", path)
	if strings.HasSuffix(path, "md") {
		return buildMarkdown(dst, src, tmpl)
	}
	_, err := io.Copy(dst, src)
	return err
}

func makeDstPath(srcDir, dstDir, filePath string) (string, error) {
	relPath, err := filepath.Rel(srcDir, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative dest path: %w", err)
	}
	if strings.HasSuffix(relPath, "md") {
		relPath = strings.TrimSuffix(relPath, filepath.Ext(relPath))
		relPath = relPath + ".html"
	}
	dstPath := path.Join(dstDir, relPath)
	if err := os.MkdirAll(path.Dir(dstPath), 0755); err != nil {
		return "", fmt.Errorf("failed to make dest dirs: %w", err)
	}
	return dstPath, nil
}

func walk(dstDir, srcDir string, tmpl *template.Template) fs.WalkDirFunc {
	return func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			// io error unrelated to building
			return err
		}
		if d.IsDir() {
			return nil
		}

		dstPath, err := makeDstPath(srcDir, dstDir, filePath)
		if err != nil {
			return fmt.Errorf("error creating dest path: %w", err)
		}

		src, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("error opening src: %w", err)
		}
		defer src.Close()

		dst, err := os.Create(dstPath)
		if err != nil {
			return fmt.Errorf("error opening dst: %w", err)
		}
		defer dst.Close()

		if err := BuildFile(filePath, dst, src, tmpl); err != nil {
			return fmt.Errorf("error building file %s: %s", filePath, err)
		}

		return nil
	}
}

func BuildTree(dst, src string, tmpl *template.Template) error {
	return filepath.WalkDir(src, walk(dst, src, tmpl))
}
