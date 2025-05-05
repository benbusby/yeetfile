package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"yeetfile/cli/config/configfile"
	"yeetfile/cli/lang"
	"yeetfile/shared"
)

// Config embeddet das Base‑Config‑Struct, um Methoden hinzuzufügen.
type Config struct {
	*configfile.Config
}

// Paths ist alias für die base Paths.
type Paths = configfile.Paths

// LoadConfig lädt die Konfiguration und beendet bei Fatal‑Error.
func LoadConfig() *Config {
	baseCfg, err := configfile.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	return &Config{baseCfg}
}

// SetSession schreibt den Session‑Token in die Session–Datei.
/*
func (c *Config) SetSession(sessionVal string) error {
	return configfile.CopyToFile(sessionVal, c.Paths.Session)
}
*/
// SetSession sets the session to the value returned by the server when signing
// up or logging in, and saves it to a (gitignored) file in the config directory
func (c Config) SetSession(sessionVal string) error {
	err := configfile.CopyToFile(sessionVal, c.Paths.Session)
	if err != nil {
		return err
	}

	return nil
}

// ReadSession liest den Session‑Token (oder nil).
/*
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
*/
// ReadSession reads the value in $config_path/session
func (c Config) ReadSession() []byte {
	if _, err := os.Stat(c.Paths.Session); err == nil {
		session, err := os.ReadFile(c.Paths.Session)
		if err != nil {
			return nil
		}

		return session
	} else {
		return nil
	}
}

// Reset löscht Session‑ und Key‑Dateien.
/*
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
*/

func (c Config) Reset() error {
	if _, err := os.Stat(c.Paths.Session); err == nil {
		err := os.Remove(c.Paths.Session)
		if err != nil {
			log.Println("error removing session file")
			return err
		}
	}

	if _, err := os.Stat(c.Paths.EncPrivateKey); err == nil {
		err = os.Remove(c.Paths.EncPrivateKey)
		if err != nil {
			log.Println("error removing private key")
			return err
		}
	}

	if _, err := os.Stat(c.Paths.PublicKey); err == nil {
		err = os.Remove(c.Paths.PublicKey)
		if err != nil {
			log.Println("error removing public key")
			return err
		}
	}

	return nil
}

// SetKeys schreibt verschlüsselte Priv‑ und Pub‑Keys.
/*
func (c *Config) SetKeys(encPrivateKey, publicKey []byte) error {
	if err := configfile.CopyBytesToFile(encPrivateKey, c.Paths.EncPrivateKey); err != nil {
		return err
	}
	return configfile.CopyBytesToFile(publicKey, c.Paths.PublicKey)
}
*/
// SetKeys writes the encrypted private key bytes and the (unencrypted) public
// key bytes to their respective file paths
func (c Config) SetKeys(encPrivateKey, publicKey []byte) error {
	err := configfile.CopyBytesToFile(encPrivateKey, c.Paths.EncPrivateKey)
	if err != nil {
		return err
	}

	err = configfile.CopyBytesToFile(publicKey, c.Paths.PublicKey)
	return err
}

// GetKeys liefert Priv‑ und Pub‑Key oder Error mit lokalisierter Msg.
/*
func (c *Config) GetKeys() ([]byte, []byte, error) {
	if _, err := os.Stat(c.Paths.EncPrivateKey); err != nil {
		return nil, nil, errors.New(lang.I18n.T("cli.config.error.no_keys"))
	}
	if _, err := os.Stat(c.Paths.PublicKey); err != nil {
		return nil, nil, errors.New(lang.I18n.T("cli.config.error.no_keys"))
	}
	priv, err1 := os.ReadFile(c.Paths.EncPrivateKey)
	pub, err2 := os.ReadFile(c.Paths.PublicKey)
	if err1 != nil || err2 != nil {
		msg := fmt.Sprintf("%s:\nprivkey: %v\npubkey: %v",
			lang.I18n.T("cli.config.error.read_keys"), err1, err2)
		return nil, nil, errors.New(msg)
	}
	return priv, pub, nil
}
*/
// GetKeys returns the user's encrypted private key and their public key from
// the config directory. Returns private key, public key, and error.
func (c Config) GetKeys() ([]byte, []byte, error) {
	var privateKey []byte
	var publicKey []byte

	_, privKeyErr := os.Stat(c.Paths.EncPrivateKey)
	_, pubKeyErr := os.Stat(c.Paths.PublicKey)

	if privKeyErr != nil || pubKeyErr != nil {
		return nil, nil, errors.New(lang.I18n.T("cli.config.error.no_keys"))
	}

	privateKey, privKeyErr = os.ReadFile(c.Paths.EncPrivateKey)
	publicKey, pubKeyErr = os.ReadFile(c.Paths.PublicKey)

	if privKeyErr != nil || pubKeyErr != nil {
		errMsg := fmt.Sprintf(lang.I18n.T("cli.config.error.read_keys")+":"+
			"\nprivkey: %v\n"+
			"pubkey: %v", privKeyErr, pubKeyErr)
		return nil, nil, errors.New(errMsg)
	}

	return privateKey, publicKey, nil
}

// SetLongWordlist schreibt die lange Wortliste.
/*
func (c *Config) SetLongWordlist(contents []byte) error {
	return configfile.CopyBytesToFile(contents, c.Paths.LongWordlist)
}
*/

func (c Config) SetLongWordlist(contents []byte) error {
	err := configfile.CopyBytesToFile(contents, c.Paths.LongWordlist)
	return err
}

func (c Config) SetShortWordlist(contents []byte) error {
	err := configfile.CopyBytesToFile(contents, c.Paths.ShortWordlist)
	return err
}

// SetShortWordlist schreibt die kurze Wortliste.
/*
func (c *Config) SetShortWordlist(contents []byte) error {
	return configfile.CopyBytesToFile(contents, c.Paths.ShortWordlist)
}
*/

// GetWordlists lädt beide Wortlisten oder liefert lokalisierten Error.
/*
func (c *Config) GetWordlists() ([]string, []string, error) {
	if _, err := os.Stat(c.Paths.LongWordlist); err != nil {
		return nil, nil, errors.New(lang.I18n.T("cli.config.error.no_wordlist"))
	}
	if _, err := os.Stat(c.Paths.ShortWordlist); err != nil {
		return nil, nil, errors.New(lang.I18n.T("cli.config.error.no_wordlist"))
	}
	longB, err1 := os.ReadFile(c.Paths.LongWordlist)
	shortB, err2 := os.ReadFile(c.Paths.ShortWordlist)
	if err1 != nil || err2 != nil {
		msg := fmt.Sprintf("%s:\nlong: %v\nshort: %v",
			lang.I18n.T("cli.config.error.read_wordlist"), err1, err2)
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
*/

func (c Config) GetWordlists() ([]string, []string, error) {
	var longWordlist []byte
	var shortWordlist []byte

	_, longWordlistErr := os.Stat(c.Paths.LongWordlist)
	_, shortWordlistErr := os.Stat(c.Paths.ShortWordlist)

	if longWordlistErr != nil || shortWordlistErr != nil {
		return nil, nil, errors.New(lang.I18n.T("cli.config.error.no_wordlist"))
	}

	longWordlist, longWordlistErr = os.ReadFile(c.Paths.LongWordlist)
	shortWordlist, shortWordlistErr = os.ReadFile(c.Paths.ShortWordlist)

	if longWordlistErr != nil || shortWordlistErr != nil {
		errMsg := fmt.Sprintf(lang.I18n.T("cli.config.error.read_wordlist")+":"+
			"\nlong wordlist: %v\n"+
			"short wordlist: %v", longWordlistErr, shortWordlistErr)
		return nil, nil, errors.New(errMsg)
	}

	var (
		longWordlistStrings  []string
		shortWordlistStrings []string
	)

	err := json.Unmarshal(longWordlist, &longWordlistStrings)
	if err != nil {
		return nil, nil, err
	}

	err = json.Unmarshal(shortWordlist, &shortWordlistStrings)
	if err != nil {
		return nil, nil, err
	}

	return longWordlistStrings, shortWordlistStrings, nil
}

// GetServerInfo gibt den gecachten ServerInfo‑Struct zurück.
/*
func (c *Config) GetServerInfo() (shared.ServerInfo, error) {
	if len(c.Server) == 0 {
		return shared.ServerInfo{}, errors.New(lang.I18n.T("cli.config.error.no_server"))
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
		return shared.ServerInfo{}, errors.New(lang.I18n.T("cli.config.error.out_of_date"))
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
*/

// GetServerInfo returns information related to the currently configured server,
// if it has been recently fetched within the last 24 hours. If it doesn't exist
// or is out of date, an error is returned.
func (c Config) GetServerInfo() (shared.ServerInfo, error) {
	if len(c.Server) == 0 {
		return shared.ServerInfo{}, errors.New(lang.I18n.T("cli.config.error.no_server"))
	}

	server, err := url.Parse(c.Server)
	if err != nil {
		return shared.ServerInfo{}, err
	}

	serverInfoName := fmt.Sprintf(configfile.ServerInfoNameFmt, server.Host)
	serverInfoPath := c.Paths.GetConfigFilePath(serverInfoName)
	infoStat, err := os.Stat(serverInfoPath)

	if err != nil {
		return shared.ServerInfo{}, err
	} else if infoStat.ModTime().Add(24 * time.Hour).Before(time.Now()) {
		return shared.ServerInfo{}, errors.New(lang.I18n.T("cli.config.error.out_of_date"))
	}

	var serverInfo shared.ServerInfo
	serverInfoBytes, err := os.ReadFile(serverInfoPath)
	if err != nil {
		return shared.ServerInfo{}, err
	}

	err = json.Unmarshal(serverInfoBytes, &serverInfo)
	if err != nil {
		return shared.ServerInfo{}, err
	}

	return serverInfo, nil
}

// SetServerInfo cached den ServerInfo‑Struct für 24 Std.
/*
func (c *Config) SetServerInfo(info shared.ServerInfo) error {
	if len(c.Server) == 0 {
		return errors.New(lang.I18n.T("cli.config.error.no_server"))
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
	return configfile.CopyBytesToFile(b, path)
}
*/
// SetServerInfo writes the information about the currently configured server to
// a file in the user's yeetfile config dir. This can be used to skip re-fetching
// server info for the next 24 hours.
func (c Config) SetServerInfo(info shared.ServerInfo) error {
	if len(c.Server) == 0 {
		return errors.New(lang.I18n.T("cli.config.error.no_server"))
	}

	server, err := url.Parse(c.Server)
	if err != nil {
		return err
	}

	serverInfoName := fmt.Sprintf(configfile.ServerInfoNameFmt, server.Host)
	serverInfoPath := c.Paths.GetConfigFilePath(serverInfoName)

	serverInfoBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	err = configfile.CopyBytesToFile(serverInfoBytes, serverInfoPath)
	if err != nil {
		return err
	}

	return nil
}
