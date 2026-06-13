package web

import (
	"embed"
	"io/fs"
)

//go:embed index.html app.js style.css
var files embed.FS

// FS returns the embedded web asset filesystem.
func FS() fs.FS { return files }
