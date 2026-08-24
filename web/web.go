package web

import (
	"embed"
	"io/fs"
)

//go:embed console.html
var consoleFiles embed.FS

// ConsoleHTML returns the embedded operator console page.
func ConsoleHTML() ([]byte, error) {
	return fs.ReadFile(consoleFiles, "console.html")
}
