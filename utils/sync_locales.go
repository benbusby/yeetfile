package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run sync_locales.go <lang> <locales-dir> [--clean] [--dry-run]")
		os.Exit(1)
	}

	targetLang := os.Args[1]
	localesDir := os.Args[2]

	flags := map[string]bool{}
	for _, arg := range os.Args[3:] {
		flags[arg] = true
	}

	cleanExtra := flags["--clean"]
	dryRun := flags["--dry-run"]

	if targetLang == "en" {
		fmt.Println("No need to self sync 'en'.")
		os.Exit(0)
	}

	err := syncLocales(localesDir, "en", targetLang, cleanExtra, dryRun)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	fmt.Printf("✅ %s/%s.json checked and synced against en.json\n", localesDir, targetLang)
}

func compareOrder(ref, actual []string) bool {
	if len(ref) != len(actual) {
		return false
	}
	for i := range ref {
		if ref[i] != actual[i] {
			return false
		}
	}
	return true
}

func syncLocales(dir, baseLang, targetLang string, cleanExtra, dryRun bool) error {
	basePath := filepath.Join(dir, baseLang+".json")
	targetPath := filepath.Join(dir, targetLang+".json")

	baseData, baseOrder, err := loadLocaleOrdered(basePath)
	if err != nil {
		return fmt.Errorf("failed to load base language (%s): %w", baseLang, err)
	}

	targetData, targetOrder, err := loadLocaleOrdered(targetPath)
	if err != nil {
		fmt.Println("Target language file not found, creating new one.")
		targetData = map[string]string{}
	}

	merged := map[string]string{}
	updated := false
	addedKeys := []string{}
	removedKeys := []string{}

	for _, key := range baseOrder {
		if val, ok := targetData[key]; ok {
			merged[key] = val
		} else {
			merged[key] = "##Missing## " + baseData[key]
			addedKeys = append(addedKeys, key)
			updated = true
		}
	}

	if cleanExtra {
		updated = true
		for key := range targetData {
			if _, inBase := baseData[key]; !inBase {
				removedKeys = append(removedKeys, key)
			}
		}
	} else {
		for key, val := range targetData {
			if _, inBase := baseData[key]; !inBase {
				merged[key] = val
			}
		}
	}

	orderCorrect := compareOrder(baseOrder, targetOrder)

	if !orderCorrect {
		updated = true
		if dryRun {
			fmt.Println("🔄 Order would be corrected to match en.json")
		}
	}

	if dryRun {
		fmt.Println("💡 Dry Run: no file will be written.")
		if len(addedKeys) > 0 {
			fmt.Println("➕ Missing keys that would be added:")
			for _, k := range addedKeys {
				fmt.Printf("   + %s\n", k)
			}
		}
		if len(removedKeys) > 0 {
			fmt.Println("➖ Extra keys that would be removed (--clean):")
			for _, k := range removedKeys {
				fmt.Printf("   - %s\n", k)
			}
		}
		if !updated {
			fmt.Println("✅ Nothing to change.")
		}
		return nil
	}

	if updated {
		return saveLocaleOrdered(targetPath, merged, baseOrder)
	}

	fmt.Println("✅ No changes needed.")
	return nil
}

func loadLocaleOrdered(path string) (map[string]string, []string, error) {
	file, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	decoder := json.NewDecoder(bytes.NewReader(file))
	tokens := make(map[string]string)
	order := []string{}

	t, err := decoder.Token()
	if err != nil || t != json.Delim('{') {
		return nil, nil, fmt.Errorf("invalid JSON object: %w", err)
	}

	for decoder.More() {
		tk, err := decoder.Token()
		if err != nil {
			return nil, nil, err
		}
		key := tk.(string)

		var val string
		if err := decoder.Decode(&val); err != nil {
			return nil, nil, err
		}

		tokens[key] = val
		order = append(order, key)
	}

	return tokens, order, nil
}

func saveLocaleOrdered(path string, data map[string]string, order []string) error {
	// Add missing keys to the order
	// and sort the extras
	extras := []string{}
	for key := range data {
		if !contains(order, key) {
			extras = append(extras, key)
		}
	}
	sort.Strings(extras)
	fullOrder := append(order, extras...)

	buf := &bytes.Buffer{}
	buf.WriteString("{\n")

	for i, key := range fullOrder {
		val, ok := data[key]
		if !ok {
			continue
		}
		line, _ := json.Marshal(key)
		buf.WriteString("  ")
		buf.Write(line)
		buf.WriteString(": ")

		valJSON, _ := json.Marshal(val)
		buf.Write(valJSON)

		if i < len(fullOrder)-1 {
			buf.WriteString(",")
		}
		buf.WriteString("\n")
	}

	buf.WriteString("}\n")
	return os.WriteFile(path, buf.Bytes(), 0644)
}

func contains(slice []string, val string) bool {
	for _, item := range slice {
		if item == val {
			return true
		}
	}
	return false
}
