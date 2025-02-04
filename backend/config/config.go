package config

import (
	"github.com/gorilla/securecookie"
	"golang.org/x/crypto/bcrypt"
	"log"
	"os"
	"slices"
	"strings"
	"yeetfile/backend/server/upgrades"
	"yeetfile/backend/utils"
	"yeetfile/shared"
	"yeetfile/shared/constants"
)

// =============================================================================
// General configuration
// =============================================================================

const LocalStorage = "local"
const B2Storage = "b2"

var defaultSecret = []byte("yeetfile-debug-secret-key-123456")
var storageType = utils.GetEnvVar("YEETFILE_STORAGE", LocalStorage)
var domain = os.Getenv("YEETFILE_DOMAIN")
var defaultUserMaxPasswords = utils.GetEnvVarInt("YEETFILE_DEFAULT_MAX_PASSWORDS", -1)
var defaultUserStorage = utils.GetEnvVarInt64("YEETFILE_DEFAULT_USER_STORAGE", -1)
var defaultUserSend = utils.GetEnvVarInt64("YEETFILE_DEFAULT_USER_SEND", -1)
var maxNumUsers = utils.GetEnvVarInt("YEETFILE_MAX_NUM_USERS", -1)
var password = []byte(utils.GetEnvVar("YEETFILE_SERVER_PASSWORD", ""))
var secret = utils.GetEnvVarBytesB64("YEETFILE_SERVER_SECRET", defaultSecret)
var fallbackWebSecret = utils.GetEnvVarBytesB64(
	"YEETFILE_FALLBACK_WEB_SECRET",
	securecookie.GenerateRandomKey(32))

var TLSCert = utils.GetEnvVar("YEETFILE_TLS_CERT", "")
var TLSKey = utils.GetEnvVar("YEETFILE_TLS_KEY", "")

var IsDebugMode = utils.GetEnvVarBool("YEETFILE_DEBUG", false)

var InstanceAdmin = utils.GetEnvVar("YEETFILE_INSTANCE_ADMIN", "")

// =============================================================================
// Email configuration (used in account verification and billing reminders)
// =============================================================================

type EmailConfig struct {
	Configured     bool
	Address        string
	Host           string
	User           string
	Port           string
	Password       string
	NoReplyAddress string
}

var email = EmailConfig{
	Configured:     false,
	Address:        os.Getenv("YEETFILE_EMAIL_ADDR"),
	Host:           os.Getenv("YEETFILE_EMAIL_HOST"),
	User:           os.Getenv("YEETFILE_EMAIL_USER"),
	Port:           os.Getenv("YEETFILE_EMAIL_PORT"),
	Password:       os.Getenv("YEETFILE_EMAIL_PASSWORD"),
	NoReplyAddress: os.Getenv("YEETFILE_EMAIL_NO_REPLY"),
}

// =============================================================================
// Billing configuration (Stripe)
// =============================================================================

type StripeBillingConfig struct {
	Configured    bool
	Key           string
	WebhookSecret string
}

var stripeBilling = StripeBillingConfig{
	Key:           os.Getenv("YEETFILE_STRIPE_KEY"),
	WebhookSecret: os.Getenv("YEETFILE_STRIPE_WEBHOOK_SECRET"),
}

// =============================================================================
// Billing configuration (BTCPay)
// =============================================================================

type BTCPayBillingConfig struct {
	Configured    bool
	WebhookSecret string
}

var btcPayBilling = BTCPayBillingConfig{
	WebhookSecret: os.Getenv("YEETFILE_BTCPAY_WEBHOOK_SECRET"),
}

// =============================================================================
// Full server config
// =============================================================================

type ServerConfig struct {
	StorageType         string
	Domain              string
	DefaultMaxPasswords int
	DefaultUserStorage  int64
	DefaultUserSend     int64
	MaxUserCount        int
	CurrentUserCount    int
	Email               EmailConfig
	StripeBilling       StripeBillingConfig
	BTCPayBilling       BTCPayBillingConfig
	BillingEnabled      bool
	Version             string
	PasswordHash        []byte
	ServerSecret        []byte
	FallbackWebSecret   []byte
}

type TemplateConfig struct {
	Version          string
	CurrentUserCount int
	MaxUserCount     int
	EmailEnabled     bool
	BillingEnabled   bool
	StripeEnabled    bool
	BTCPayEnabled    bool
}

var YeetFileConfig ServerConfig
var HTMLConfig TemplateConfig

func init() {
	email.Configured = !utils.IsStructMissingAnyField(email)
	stripeBilling.Configured = !utils.IsStructMissingAnyField(stripeBilling)
	btcPayBilling.Configured = !utils.IsStructMissingAnyField(btcPayBilling)

	var passwordHash []byte
	var err error
	if len(password) > 0 {
		passwordHash, err = bcrypt.GenerateFromPassword(password, 8)
		if err != nil {
			panic(err)
		}
	}

	if slices.Equal(secret, defaultSecret) {
		logWarning(
			"Server secret is set to the default value.",
			"YEETFILE_SERVER_SECRET should be set to a ",
			"unique, 32-byte base-64 encoded value in production.")
	} else if len(secret) != constants.KeySize {
		log.Fatalf("ERROR: YEETFILE_SERVER_SECRET is %d bytes, but %d "+
			"bytes are required.", len(secret), constants.KeySize)
	}

	YeetFileConfig = ServerConfig{
		StorageType:         storageType,
		Domain:              domain,
		DefaultMaxPasswords: defaultUserMaxPasswords,
		DefaultUserStorage:  defaultUserStorage,
		DefaultUserSend:     defaultUserSend,
		MaxUserCount:        maxNumUsers,
		Email:               email,
		StripeBilling:       stripeBilling,
		BTCPayBilling:       btcPayBilling,
		BillingEnabled:      stripeBilling.Configured || btcPayBilling.Configured,
		Version:             constants.VERSION,
		PasswordHash:        passwordHash,
		ServerSecret:        secret,
		FallbackWebSecret:   fallbackWebSecret,
	}

	// Subset of main server config to use in HTML templating
	HTMLConfig = TemplateConfig{
		Version:        YeetFileConfig.Version,
		MaxUserCount:   YeetFileConfig.MaxUserCount,
		EmailEnabled:   YeetFileConfig.Email.Configured,
		BillingEnabled: YeetFileConfig.BillingEnabled,
		StripeEnabled:  YeetFileConfig.StripeBilling.Configured,
		BTCPayEnabled:  YeetFileConfig.BTCPayBilling.Configured,
	}

	log.Printf("Configuration:\n"+
		"  Email:            %v\n"+
		"  Billing (Stripe): %v\n"+
		"  Billing (BTCPay): %v\n",
		email.Configured,
		stripeBilling.Configured,
		btcPayBilling.Configured,
	)

	if IsDebugMode {
		logWarning(
			"DEBUG MODE IS ACTIVE!",
			"DO NOT USE THIS SETTING IN PRODUCTION!")
	}
}

func logWarning(warnings ...string) {
	log.Println(strings.Repeat("@", 57))
	for _, warning := range warnings {
		log.Printf("!!! " + warning + "\n")
	}
	log.Println(strings.Repeat("@", 57))
}

func GetServerInfoStruct() shared.ServerInfo {
	var storageBackend string
	if storageType == B2Storage {
		storageBackend = "Backblaze B2"
	} else {
		storageBackend = "Server Storage"
	}

	allUpgrades := upgrades.GetAllUpgrades()

	return shared.ServerInfo{
		StorageBackend:     storageBackend,
		PasswordRestricted: YeetFileConfig.PasswordHash != nil,
		MaxUserCountSet:    YeetFileConfig.MaxUserCount > 0,
		EmailConfigured:    YeetFileConfig.Email.Configured,
		BillingEnabled:     YeetFileConfig.BillingEnabled,
		StripeEnabled:      YeetFileConfig.BTCPayBilling.Configured,
		BTCPayEnabled:      YeetFileConfig.StripeBilling.Configured,
		DefaultStorage:     YeetFileConfig.DefaultUserStorage,
		DefaultSend:        YeetFileConfig.DefaultUserSend,

		Upgrades:      *allUpgrades,
		MonthUpgrades: upgrades.GetVaultUpgrades(false, allUpgrades.VaultUpgrades),
		YearUpgrades:  upgrades.GetVaultUpgrades(true, allUpgrades.VaultUpgrades),
	}
}
