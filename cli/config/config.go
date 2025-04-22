package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"yeetfile/cli/clilang"
	"yeetfile/cli/config/configbase"
	"yeetfile/shared"
)

// Config embeddet das Base‑Config‑Struct, um Methoden hinzuzufügen.
type Config struct {
	*configbase.Config
}

// Paths ist alias für die base Paths.
type Paths = configbase.Paths

// LoadConfig lädt die Konfiguration und beendet bei Fatal‑Error.
func LoadConfig() *Config {
	baseCfg, err := configbase.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	return &Config{baseCfg}
}

// SetSession schreibt den Session‑Token in die Session–Datei.
func (c *Config) SetSession(sessionVal string) error {
	return configbase.CopyToFile(sessionVal, c.Paths.Session)
}

// ReadSession liest den Session‑Token (oder nil).
func (c *Config) ReadSession() []byte {
	if _, err := os.Stat(c.Paths.Session); err == nil {
		data, err := os.ReadFile(c.Paths.Session)
		if err != nil {
			return nil
		}
		return data
	}
	return nil
}

// Reset löscht Session‑ und Key‑Dateien.
func (c *Config) Reset() error {
	if _, err := os.Stat(c.Paths.Session); err == nil {
		if err := os.Remove(c.Paths.Session); err != nil {
			log.Println("error removing session file")
			return err
		}
	}
	if _, err := os.Stat(c.Paths.EncPrivateKey); err == nil {
		if err := os.Remove(c.Paths.EncPrivateKey); err != nil {
			log.Println("error removing private key")
			return err
		}
	}
	if _, err := os.Stat(c.Paths.PublicKey); err == nil {
		if err := os.Remove(c.Paths.PublicKey); err != nil {
			log.Println("error removing public key")
			return err
		}
	}
	return nil
}

// SetKeys schreibt verschlüsselte Priv‑ und Pub‑Keys.
func (c *Config) SetKeys(encPrivateKey, publicKey []byte) error {
	if err := configbase.CopyBytesToFile(encPrivateKey, c.Paths.EncPrivateKey); err != nil {
		return err
	}
	return configbase.CopyBytesToFile(publicKey, c.Paths.PublicKey)
}

// GetKeys liefert Priv‑ und Pub‑Key oder Error mit lokalisierter Msg.
func (c *Config) GetKeys() ([]byte, []byte, error) {
	if _, err := os.Stat(c.Paths.EncPrivateKey); err != nil {
		return nil, nil, errors.New(clilang.I18n.T("cli.config.error.no_keys"))
	}
	if _, err := os.Stat(c.Paths.PublicKey); err != nil {
		return nil, nil, errors.New(clilang.I18n.T("cli.config.error.no_keys"))
	}
	priv, err1 := os.ReadFile(c.Paths.EncPrivateKey)
	pub, err2 := os.ReadFile(c.Paths.PublicKey)
	if err1 != nil || err2 != nil {
		msg := fmt.Sprintf("%s:\nprivkey: %v\npubkey: %v",
			clilang.I18n.T("cli.config.error.read_keys"), err1, err2)
		return nil, nil, errors.New(msg)
	}
	return priv, pub, nil
}

// SetLongWordlist schreibt die lange Wortliste.
func (c *Config) SetLongWordlist(contents []byte) error {
	return configbase.CopyBytesToFile(contents, c.Paths.LongWordlist)
}

// SetShortWordlist schreibt die kurze Wortliste.
func (c *Config) SetShortWordlist(contents []byte) error {
	return configbase.CopyBytesToFile(contents, c.Paths.ShortWordlist)
}

// GetWordlists lädt beide Wortlisten oder liefert lokalisierten Error.
func (c *Config) GetWordlists() ([]string, []string, error) {
	if _, err := os.Stat(c.Paths.LongWordlist); err != nil {
		return nil, nil, errors.New(clilang.I18n.T("cli.config.error.no_wordlist"))
	}
	if _, err := os.Stat(c.Paths.ShortWordlist); err != nil {
		return nil, nil, errors.New(clilang.I18n.T("cli.config.error.no_wordlist"))
	}
	longB, err1 := os.ReadFile(c.Paths.LongWordlist)
	shortB, err2 := os.ReadFile(c.Paths.ShortWordlist)
	if err1 != nil || err2 != nil {
		msg := fmt.Sprintf("%s:\nlong: %v\nshort: %v",
			clilang.I18n.T("cli.config.error.read_wordlist"), err1, err2)
		return nil, nil, errors.New(msg)
	}
	var longList, shortList []string
	if err := json.Unmarshal(longB, &longList); err != nil {
		return nil, nil, err
	}
	if err := json.Unmarshal(shortB, &shortList); err != nil {
		return nil, nil, err
	}
	return longList, shortList, nil
}

// GetServerInfo gibt den gecachten ServerInfo‑Struct zurück.
func (c *Config) GetServerInfo() (shared.ServerInfo, error) {
	if len(c.Server) == 0 {
		return shared.ServerInfo{}, errors.New(clilang.I18n.T("cli.config.error.no_server"))
	}
	u, err := url.Parse(c.Server)
	if err != nil {
		return shared.ServerInfo{}, err
	}
	fname := fmt.Sprintf("%s.json", u.Host)
	path := c.Paths.GetConfigFilePath(fname)
	infoStat, err := os.Stat(path)
	if err != nil {
		return shared.ServerInfo{}, err
	}
	if infoStat.ModTime().Add(24 * time.Hour).Before(time.Now()) {
		return shared.ServerInfo{}, errors.New(clilang.I18n.T("cli.config.error.out_of_date"))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return shared.ServerInfo{}, err
	}
	var info shared.ServerInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return shared.ServerInfo{}, err
	}
	return info, nil
}

// SetServerInfo cached den ServerInfo‑Struct für 24 Std.
func (c *Config) SetServerInfo(info shared.ServerInfo) error {
	if len(c.Server) == 0 {
		return errors.New(clilang.I18n.T("cli.config.error.no_server"))
	}
	u, err := url.Parse(c.Server)
	if err != nil {
		return err
	}
	fname := fmt.Sprintf("%s.json", u.Host)
	path := c.Paths.GetConfigFilePath(fname)
	b, err := json.Marshal(info)
	if err != nil {
		return err
	}
	return configbase.CopyBytesToFile(b, path)
}
