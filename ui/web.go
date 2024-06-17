package ui

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
)

//go:embed all:build
var build embed.FS

func NewUIStaticFS() http.FileSystem {
	uiFS, err := fs.Sub(build, "build")
	if err != nil {
		log.Fatal("cannot load ui files:", err)
	}
	return http.FS(uiFS)
}
