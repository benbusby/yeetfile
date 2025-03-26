package config

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"yeetfile/cli/utils"
	"yeetfile/shared"

	"gopkg.in/yaml.v3"
)

type Paths struct {
	directory string

	config        string
	gitignore     string
	session       string
	encPrivateKey string
	publicKey     string

	longWordlist  string
	shortWordlist string
}

type Config struct {
	Server      string     `yaml:"server,omitempty"`
	DefaultView string     `yaml:"default_view,omitempty"`
	DebugMode   bool       `yaml:"debug_mode,omitempty"`
	DebugFile   string     `yaml:"debug_file,omitempty"`
	Send        SendConfig `yaml:"send,omitempty"`
	Paths       Paths
}

type SendConfig struct {
	Downloads        int    `yaml:"downloads,omitempty"`
	ExpirationAmount int    `yaml:"expiration_amount,omitempty"`
	ExpirationUnits  string `yaml:"expiration_units,omitempty"`
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

	serverInfoNameFmt = "%s.json" // ie "yeetfile.com.json"
)

//go:embed config.yml
var defaultConfig string

func (p Paths) getConfigFilePath(filename string) string {
	return filepath.Join(p.directory, filename)
}

// setupConfigDir ensures that the directory necessary for yeetfile's config
// have been created. This path defaults to $HOME/.config/yeetfile.
func setupConfigDir() (Paths, error) {
	var localConfig string
	var configErr error
	if runtime.GOOS == "darwin" {
		baseDir, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}

		localConfig, configErr = makeConfigDirectories(baseDir, baseConfigPath)
	} else {
		baseDir, err := os.UserConfigDir()
		if err != nil {
			return Paths{}, err
		}

		localConfig, configErr = makeConfigDirectories(baseDir, "yeetfile")
	}

	if configErr != nil {
		return Paths{}, configErr
	}

	return Paths{
		directory:     localConfig,
		config:        filepath.Join(localConfig, configFileName),
		gitignore:     filepath.Join(localConfig, gitignoreName),
		session:       filepath.Join(localConfig, sessionName),
		encPrivateKey: filepath.Join(localConfig, encPrivateKeyName),
		publicKey:     filepath.Join(localConfig, publicKeyName),
		longWordlist:  filepath.Join(localConfig, longWordlistName),
		shortWordlist: filepath.Join(localConfig, shortWordlistName),
	}, nil
}

// setupTempConfigDir creates a config directory for the current user in the
// OS's temporary directory. Used for testing.
func setupTempConfigDir() (Paths, error) {
	dirname := os.TempDir()
	localConfig, err := makeConfigDirectories(dirname, baseConfigPath)
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		config:        filepath.Join(localConfig, configFileName),
		gitignore:     filepath.Join(localConfig, gitignoreName),
		session:       filepath.Join(localConfig, sessionName),
		encPrivateKey: filepath.Join(localConfig, encPrivateKeyName),
		publicKey:     filepath.Join(localConfig, publicKeyName),
		longWordlist:  filepath.Join(localConfig, longWordlistName),
		shortWordlist: filepath.Join(localConfig, shortWordlistName),
	}, nil
}

// makeConfigDirectories creates the necessary directories for storing the
// user's local yeetfile config
func makeConfigDirectories(baseDir, configPath string) (string, error) {
	localConfig := filepath.Join(baseDir, configPath)
	err := os.MkdirAll(localConfig, os.ModePerm)
	if err != nil {
		return "", err
	}

	return localConfig, nil
}

// ReadConfig reads the config file (config.yml) for current configuration
func ReadConfig(p Paths) (Config, error) {
	if _, err := os.Stat(p.config); err == nil {
		config := Config{Paths: p}
		data, err := os.ReadFile(p.config)
		if err != nil {
			return config, err
		}

		err = yaml.Unmarshal(data, &config)
		if err != nil {
			return config, err
		}

		// Strip trailing slash
		if strings.HasSuffix(config.Server, "/") {
			config.Server = config.Server[0 : len(config.Server)-1]
		}

		return config, nil
	} else {
		err = setupDefaultConfig(p)
		if err != nil {
			return Config{}, err
		}
		return ReadConfig(p)
	}
}

// setupDefaultConfig copies default config files from the repo to the user's
// config directory
func setupDefaultConfig(p Paths) error {
	err := utils.CopyToFile(defaultConfig, p.config)
	if err != nil {
		return err
	}

	defaultGitignore := fmt.Sprintf(`
%s
%s
%s`, sessionName, encPrivateKeyName, publicKeyName)

	err = utils.CopyToFile(defaultGitignore, p.gitignore)
	if err != nil {
		return err
	}

	err = utils.CopyToFile("", p.session)
	if err != nil {
		return err
	}

	return nil
}

// SetSession sets the session to the value returned by the server when signing
// up or logging in, and saves it to a (gitignored) file in the config directory
func (c Config) SetSession(sessionVal string) error {
	err := utils.CopyToFile(sessionVal, c.Paths.session)
	if err != nil {
		return err
	}

	return nil
}

// ReadSession reads the value in $config_path/session
func (c Config) ReadSession() []byte {
	if _, err := os.Stat(c.Paths.session); err == nil {
		session, err := os.ReadFile(c.Paths.session)
		if err != nil {
			return nil
		}

		return session
	} else {
		return nil
	}
}

func (c Config) Reset() error {
	if _, err := os.Stat(c.Paths.session); err == nil {
		err := os.Remove(c.Paths.session)
		if err != nil {
			log.Println("error removing session file")
			return err
		}
	}

	if _, err := os.Stat(c.Paths.encPrivateKey); err == nil {
		err = os.Remove(c.Paths.encPrivateKey)
		if err != nil {
			log.Println("error removing private key")
			return err
		}
	}

	if _, err := os.Stat(c.Paths.publicKey); err == nil {
		err = os.Remove(c.Paths.publicKey)
		if err != nil {
			log.Println("error removing public key")
			return err
		}
	}

	return nil
}

// SetKeys writes the encrypted private key bytes and the (unencrypted) public
// key bytes to their respective file paths
func (c Config) SetKeys(encPrivateKey, publicKey []byte) error {
	err := utils.CopyBytesToFile(encPrivateKey, c.Paths.encPrivateKey)
	if err != nil {
		return err
	}

	err = utils.CopyBytesToFile(publicKey, c.Paths.publicKey)
	return err
}

// GetKeys returns the user's encrypted private key and their public key from
// the config directory. Returns private key, public key, and error.
func (c Config) GetKeys() ([]byte, []byte, error) {
	var privateKey []byte
	var publicKey []byte

	_, privKeyErr := os.Stat(c.Paths.encPrivateKey)
	_, pubKeyErr := os.Stat(c.Paths.publicKey)

	if privKeyErr != nil || pubKeyErr != nil {
		return nil, nil, errors.New("key files do not exist in config dir")
	}

	privateKey, privKeyErr = os.ReadFile(c.Paths.encPrivateKey)
	publicKey, pubKeyErr = os.ReadFile(c.Paths.publicKey)

	if privKeyErr != nil || pubKeyErr != nil {
		errMsg := fmt.Sprintf("error reading key files:\n"+
			"privkey: %v\n"+
			"pubkey: %v", privKeyErr, pubKeyErr)
		return nil, nil, errors.New(errMsg)
	}

	return privateKey, publicKey, nil
}

func (c Config) SetLongWordlist(contents []byte) error {
	err := utils.CopyBytesToFile(contents, c.Paths.longWordlist)
	return err
}

func (c Config) SetShortWordlist(contents []byte) error {
	err := utils.CopyBytesToFile(contents, c.Paths.shortWordlist)
	return err
}

func (c Config) GetWordlists() ([]string, []string, error) {
	var longWordlist []byte
	var shortWordlist []byte

	_, longWordlistErr := os.Stat(c.Paths.longWordlist)
	_, shortWordlistErr := os.Stat(c.Paths.shortWordlist)

	if longWordlistErr != nil || shortWordlistErr != nil {
		return nil, nil, errors.New("wordlist files do not exist in config dir")
	}

	longWordlist, longWordlistErr = os.ReadFile(c.Paths.longWordlist)
	shortWordlist, shortWordlistErr = os.ReadFile(c.Paths.shortWordlist)

	if longWordlistErr != nil || shortWordlistErr != nil {
		errMsg := fmt.Sprintf("error reading wordlist files:\n"+
			"long wordlist: %v\n"+
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

// GetServerInfo returns information related to the currently configured server,
// if it has been recently fetched within the last 24 hours. If it doesn't exist
// or is out of date, an error is returned.
func (c Config) GetServerInfo() (shared.ServerInfo, error) {
	if len(c.Server) == 0 {
		return shared.ServerInfo{}, errors.New("missing server in config file")
	}

	server, err := url.Parse(c.Server)
	if err != nil {
		return shared.ServerInfo{}, err
	}

	serverInfoName := fmt.Sprintf(serverInfoNameFmt, server.Host)
	serverInfoPath := c.Paths.getConfigFilePath(serverInfoName)
	infoStat, err := os.Stat(serverInfoPath)

	if err != nil {
		return shared.ServerInfo{}, err
	} else if infoStat.ModTime().Add(24 * time.Hour).Before(time.Now()) {
		return shared.ServerInfo{}, errors.New("server info is out of date")
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

// SetServerInfo writes the information about the currently configured server to
// a file in the user's yeetfile config dir. This can be used to skip re-fetching
// server info for the next 24 hours.
func (c Config) SetServerInfo(info shared.ServerInfo) error {
	if len(c.Server) == 0 {
		return errors.New("missing server in config file")
	}

	server, err := url.Parse(c.Server)
	if err != nil {
		return err
	}

	serverInfoName := fmt.Sprintf(serverInfoNameFmt, server.Host)
	serverInfoPath := c.Paths.getConfigFilePath(serverInfoName)

	serverInfoBytes, err := json.Marshal(info)
	if err != nil {
		return err
	}

	err = utils.CopyBytesToFile(serverInfoBytes, serverInfoPath)
	if err != nil {
		return err
	}

	return nil
}

func LoadConfig() *Config {
	var err error

	// Setup config dir
	userConfigPaths, err := setupConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	userConfig, err := ReadConfig(userConfigPaths)
	if err != nil {
		log.Fatal(err)
	}

	return &userConfig
}
