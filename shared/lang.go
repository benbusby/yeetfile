package shared

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
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
