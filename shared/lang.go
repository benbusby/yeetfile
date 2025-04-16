package shared

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS

type I18n struct {
	Messages map[string]string
}

func LoadI18n(lang string) (*I18n, error) {
	filePath := filepath.Join("locales", lang+".json")
	bytes, err := localeFiles.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not load language file: %w", err)
	}

	var messages map[string]string
	err = json.Unmarshal(bytes, &messages)
	if err != nil {
		return nil, fmt.Errorf("could not parse language file: %w", err)
	}

	return &I18n{Messages: messages}, nil
}

// T returns a translated message with optional placeholders
func (i *I18n) T(key string, vars map[string]string) string {
	msg, ok := i.Messages[key]
	if !ok {
		return key
	}

	for k, v := range vars {
		msg = strings.ReplaceAll(msg, "{"+k+"}", v)
	}

	return msg
}

func DetectSystemLanguage() string {
	if runtime.GOOS == "windows" {
		return detectWindowsLanguage()
	}
	return detectUnixLikeLanguage()
}

// Detect language on Linux and macOS
func detectUnixLikeLanguage() string {
	lang := os.Getenv("LC_ALL")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	return parseLangCode(lang)
}

// Detect language on Winbdows
func detectWindowsLanguage() string {
	/*
		kernel32 := syscall.NewLazyDLL("kernel32.dll")
		getLocaleName := kernel32.NewProc("GetUserDefaultLocaleName")

		buf := make([]uint16, 85) // max size according to Microsoft docs
		_, _, _ = getLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))

		for i, v := range buf {
			if v == 0 {
				buf = buf[:i]
				break
			}
		}

		locale := string(utf16.Decode(buf))
		return parseLangCode(locale)
	*/
	return "en" // Placeholder for Windows language detection
}

// Normalize language identifier
func parseLangCode(code string) string {
	code = strings.ToLower(code)
	switch {
	case strings.HasPrefix(code, "de"):
		return "de"
	case strings.HasPrefix(code, "en"):
		return "en"
	default:
		return "en"
	}
}
