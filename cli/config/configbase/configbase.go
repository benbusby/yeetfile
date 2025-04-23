package configbase

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config.yml
var defaultConfig string

// Paths hält alle relevanten Pfade innerhalb des Config‑Ordners.
type Paths struct {
	Directory     string
	Config        string
	Gitignore     string
	Session       string
	EncPrivateKey string
	PublicKey     string
	LongWordlist  string
	ShortWordlist string
}

type SendConfig struct {
	Downloads        int    `yaml:"downloads,omitempty"`
	ExpirationAmount int    `yaml:"expiration_amount,omitempty"`
	ExpirationUnits  string `yaml:"expiration_units,omitempty"`
}

type Config struct {
	Server      string     `yaml:"server,omitempty"`
	DefaultView string     `yaml:"default_view,omitempty"`
	DebugMode   bool       `yaml:"debug_mode,omitempty"`
	DebugFile   string     `yaml:"debug_file,omitempty"`
	Send        SendConfig `yaml:"send,omitempty"`
	Locale      string     `yaml:"locale,omitempty"`
	Paths       Paths
}

var baseConfigPath = filepath.Join(".config", "yeetfile")

const (
	configFileName    = "config.yml"
	gitignoreName     = ".gitignore"
	sessionName       = "session"
	encPrivateKeyName = "enc-priv-key"
	publicKeyName     = "pub-key"
	longWordlistName  = "long-wordlist.json"
	shortWordlistName = "short-wordlist.json"
)

// GetConfigFilePath liefert den absoluten Pfad unterhalb des Config‑Ordners.
func (p Paths) GetConfigFilePath(filename string) string {
	return filepath.Join(p.Directory, filename)
}

// SetupConfigDir legt das Config‑Verzeichnis im User‑Scope an.
func SetupConfigDir() (Paths, error) {
	var baseDir string
	var err error
	if runtime.GOOS == "darwin" {
		baseDir, err = os.UserHomeDir()
	} else {
		baseDir, err = os.UserConfigDir()
	}
	if err != nil {
		return Paths{}, err
	}
	cfgDir, err := makeConfigDirectories(baseDir, baseConfigPath)
	if err != nil {
		return Paths{}, err
	}
	return makePaths(cfgDir), nil
}

// SetupTempConfigDir nutzt das OS‑Temp‑Verzeichnis (für Tests).
func SetupTempConfigDir() (Paths, error) {
	cfgDir, err := makeConfigDirectories(os.TempDir(), baseConfigPath)
	if err != nil {
		return Paths{}, err
	}
	return makePaths(cfgDir), nil
}

func makeConfigDirectories(baseDir, cfgPath string) (string, error) {
	full := filepath.Join(baseDir, cfgPath)
	if err := os.MkdirAll(full, os.ModePerm); err != nil {
		return "", err
	}
	return full, nil
}

func makePaths(dir string) Paths {
	return Paths{
		Directory:     dir,
		Config:        filepath.Join(dir, configFileName),
		Gitignore:     filepath.Join(dir, gitignoreName),
		Session:       filepath.Join(dir, sessionName),
		EncPrivateKey: filepath.Join(dir, encPrivateKeyName),
		PublicKey:     filepath.Join(dir, publicKeyName),
		LongWordlist:  filepath.Join(dir, longWordlistName),
		ShortWordlist: filepath.Join(dir, shortWordlistName),
	}
}

// ReadConfig lädt die YAML‑Config oder legt bei Fehlen die Default‑Config an.
func ReadConfig(p Paths) (Config, error) {
	var cfg Config
	cfg.Paths = p

	if _, err := os.Stat(p.Config); err == nil {
		data, err := os.ReadFile(p.Config)
		if err != nil {
			return cfg, err
		}
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return cfg, err
		}
		cfg.Server = strings.TrimSuffix(cfg.Server, "/")
		return cfg, nil
	}
	// Datei existiert nicht → Default anlegen und erneut lesen
	if err := setupDefaultConfig(p); err != nil {
		return cfg, err
	}
	return ReadConfig(p)
}

func setupDefaultConfig(p Paths) error {
	if err := CopyToFile(defaultConfig, p.Config); err != nil {
		return err
	}
	gitignore := fmt.Sprintf("%s\n%s\n%s", sessionName, encPrivateKeyName, publicKeyName)
	if err := CopyToFile(gitignore, p.Gitignore); err != nil {
		return err
	}
	return CopyToFile("", p.Session)
}

// LoadConfig ist der einfache Factory‑Wrapper.
func LoadConfig() (*Config, error) {
	paths, err := SetupConfigDir()
	if err != nil {
		return nil, err
	}
	cfg, err := ReadConfig(paths)
	return &cfg, err
}

// CopyToFile/CopyBytesToFile schreiben Text bzw. Bytes mit 0644‑Rechten.
func CopyToFile(content string, dest string) error {
	return CopyBytesToFile([]byte(content), dest)
}

func CopyBytesToFile(content []byte, dest string) error {
	return os.WriteFile(dest, content, 0o644)
}
