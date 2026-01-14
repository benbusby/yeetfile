package config

import (
	"strings"
	"testing"

	"yeetfile/cli/config/configfile"
)

const session = "test_session"

func TestReadConfig(t *testing.T) {
	paths, err := configfile.SetupTempConfigDir()
	if err != nil {
		t.Fatal("Failed to set up temporary config directories")
	}

	config, err := configfile.ReadConfig(paths)
	if err != nil {
		t.Fatal("Failed to read config")
	}

	if !strings.Contains(config.Server, "http") {
		t.Fatal("Invalid config server")
	}
}

func TestReadSession(t *testing.T) {
	paths, err := configfile.SetupTempConfigDir()
	if err != nil {
		t.Fatal("Failed to set up temporary config directories")
	}

	config, _ := configfile.ReadConfig(paths)
	var cfg *Config
	cfg = &Config{&config}

	err = cfg.SetSession(session)
	if err != nil {
		t.Fatal("Failed to set user session")
	}

	readSession := cfg.ReadSession()
	if len(readSession) == 0 {
		t.Fatal("Failed to read user session")
	} else if string(readSession) != session {
		t.Fatalf("Unexpected session value\n"+
			"(expected %s, got %s)", session, string(readSession))
	}
}
