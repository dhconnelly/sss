package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"text/template"

	"github.com/dhconnelly/sss/build"
	"github.com/dhconnelly/sss/serve"
)

var (
	port      = flag.Int("port", 8080, "port on which to serve")
	srcDir    = flag.String("srcDir", "pages", "source directory to build")
	dstDir    = flag.String("dstDir", "target", "build output directory")
	postTmpl  = flag.String("postTmpl", "templates/post-template.html", "path to the post template html")
	buildSite = flag.Bool("build", true, "whether to build the site")
	serveSite = flag.Bool("serve", true, "whether to start the server")
)

func main() {
	flag.Parse()

	if *buildSite {
		log.Println("source directory:", *srcDir)
		log.Println("output directory:", *dstDir)
		log.Println("post template:", *postTmpl)
		tmplPath, err := filepath.Abs(*postTmpl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: can't load template %s: %s\n", *postTmpl, err)
			os.Exit(1)
		}
		tmplName := filepath.Base(*postTmpl)
		tmpl := template.Must(template.New(tmplName).ParseFiles(tmplPath))
		if err := build.BuildTree(*dstDir, *srcDir, tmpl); err != nil {
			fmt.Fprintf(os.Stderr, "error: %s\n", err)
			os.Exit(1)
		}
	}

	if *serveSite {
		log.Println("serving static files from:", *dstDir)
		log.Printf("serving at http://localhost:%d\n", *port)
		http.HandleFunc("/healthz", func(resp http.ResponseWriter, req *http.Request) {
			resp.WriteHeader(200)
		})
		// TODO: expose in-memory cache at /cachez
		http.Handle("/", serve.NewHandler(http.Dir(*dstDir)))
		if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), nil); err != nil {
			log.Println("fatal:", err)
		}
	}
}
