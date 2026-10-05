package web

import "embed"

//go:embed index.html
var Pages embed.FS

//go:embed static
var Static embed.FS
