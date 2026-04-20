//go:build windows && amd64

package extensions

import "embed"

//go:embed embed/windows_amd64/*
var embeddedFiles embed.FS
