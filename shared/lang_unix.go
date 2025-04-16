//go:build linux || darwin

package shared

import (
	"os"
)

// Detect language on Linux and macOS
func DetectSystemLanguage() string {
	lang := os.Getenv("LC_ALL")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	return parseLangCode(lang)
}
