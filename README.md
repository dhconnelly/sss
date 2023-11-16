# sss

a simple static site compiler and server

## Overview

`sss` will recursively compile all files under a source directory into a target
directory. The logic for doing this is very simple:

-   Markdown (.md) files will be rendered as HTML (.html) files into the
    target directory. They should have the form:

        === $TITLE ===
        $CONTENT

    All Markdown files will be rendered using a single Go template (by default
    `templates/post-template.html`). During rendering, $TITLE and $CONTENT data
    will be accessible in the `{{.Title}}` and `{{.Content}}` variables.

-   All other files are copied to the target directory without modification.

## Installation

Make sure you have [Go](https://go.dev/dl/). Then:

    go install github.com/dhconnelly/sss@latest

## Usage

To build a site from the directory `SOURCE` (by default `pages/`) into the
directory `DEST` (by default `target/`) and serve it on port 8080:

    sss [-srcDir=SOURCE] [-targetDir=DEST]

## Advanced usage

To skip serving, so that you can pre-compile the site (e.g. into a Docker
image like [here](https://github.com/dhconnelly/dhcdev):

    sss -serveSite=false

Similarly, to skip building and serve a pre-compiled site:

    sss -buildSite=false

To see additional options you can print the usage message with `sss -h`.

## Development

    go build        # build
    go test ./...   # run unit tests

## Implementation

-   `package build`: implements a simple static site builder. It copies files
    from one directory to another, rendering any Markdown (`.md`) files into
    HTML and copying all other files without modification.

-   `package serve`: implements an http handler for static files. This is a
    glorified wrapper around the Go standard library's `http.FileServer` that
    logs requests, caches files in memory, and exposes basic metrics using
    `expvar` at the path `/debug/vars`, 

-   `package cache`: implements an LFU cache for static files.

-   `package main` wraps up `build` and `serve`. The `Dockerfile` uses this
    to build and serve the site, but separates the build and serving stages in
    order to deploy the static assets in the Docker image.

## License

MIT
