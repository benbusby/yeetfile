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
	fallbackLang := "en"

	// Load fallback locale first
	fallbackMessages, err := loadLangFile(fallbackLang)
	if err != nil {
		return nil, fmt.Errorf("could not load fallback language: %w", err)
	}

	// Load main locale unless en is requested
	var messages map[string]string
	if lang != fallbackLang {
		messages, err = loadLangFile(lang)
		if err != nil {
			// On error keep fallback messages
			messages = map[string]string{}
		}
	} else {
		messages = map[string]string{}
	}

	// Use fallback on missing keys
	for k, v := range fallbackMessages {
		if _, ok := messages[k]; !ok {
			messages[k] = v
		}
	}

	return &I18n{Messages: messages}, nil
}

func loadLangFile(lang string) (map[string]string, error) {
	filePath := filepath.Join("locales", lang+".json")
	bytes, err := localeFiles.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var messages map[string]string
	err = json.Unmarshal(bytes, &messages)
	if err != nil {
		return nil, err
	}

	return messages, nil
}

// T returns a translated message with optional placeholders
func (i *I18n) T(key string, vars ...map[string]string) string {
	msg, ok := i.Messages[key]
	if !ok {
		return key
	}

	var replacements map[string]string
	if len(vars) > 0 && vars[0] != nil {
		replacements = vars[0]
	} else {
		replacements = map[string]string{}
	}

	for k, v := range replacements {
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
