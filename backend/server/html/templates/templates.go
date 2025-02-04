package templates

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
	"strings"
	"yeetfile/backend/config"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

const (
	SendHTML             = "send.html"
	VaultHTML            = "vault.html"
	DownloadHTML         = "download.html"
	VerificationHTML     = "verify.html"
	SignupHTML           = "signup.html"
	LoginHTML            = "login.html"
	AccountHTML          = "account.html"
	UpgradeHTML          = "upgrade.html"
	ForgotHTML           = "forgot.html"
	ChangeEmailHTML      = "change_email.html"
	ChangePasswordHTML   = "change_password.html"
	ChangeHintHTML       = "change_hint.html"
	TwoFactorHTML        = "enable_2fa.html"
	ServerInfoHTML       = "server_info.html"
	CheckoutCompleteHTML = "checkout_complete.html"
	AdminHTML            = "admin.html"
)

//go:embed *.html
//go:embed items/*.html
var HTML embed.FS

var templates *template.Template

type BaseTemplate struct {
	LoggedIn   bool
	Title      string
	Page       string
	Javascript []string
	CSS        []string
	Version    string
	Config     config.TemplateConfig
	Endpoints  endpoints.HTMLEndpoints
}

type Template struct {
	Base BaseTemplate
}

type SignupTemplate struct {
	Base                   BaseTemplate
	ServerPasswordRequired bool
	EmailConfigured        bool
}

type LoginTemplate struct {
	Base  BaseTemplate
	Meter int
}

type SendTemplate struct {
	Base            BaseTemplate
	SendUsed        int64
	SendAvailable   int64
	ShowUpgradeLink bool
}

type VaultTemplate struct {
	Base             BaseTemplate
	VaultName        string
	FolderName       string
	StorageAvailable int64
	StorageUsed      int64
	IsPasswordVault  bool
}

type InfoTemplate struct {
	Base               BaseTemplate
	StorageBackend     string
	HasRestrictions    bool
	PasswordRestricted bool
	MaxUserCountSet    bool
	EmailConfigured    bool
	BillingEnabled     bool
	StripeEnabled      bool
	BTCPayEnabled      bool
	DefaultStorage     string
	DefaultSend        string

	SendUpgrades  []*shared.Upgrade
	VaultUpgrades []*shared.Upgrade
}

type CheckoutCompleteTemplate struct {
	Base        BaseTemplate
	Title       string
	Description string
	Note        string
}

type AdminTemplate struct {
	Base BaseTemplate
}

type AccountTemplate struct {
	Base              BaseTemplate
	Email             string
	EmailConfigured   bool
	Meter             int
	IsActive          bool
	PaymentID         string
	ExpString         string
	IsPrevUpgraded    bool
	StorageAvailable  string
	StorageUsed       string
	SendAvailable     string
	SendUsed          string
	StripeConfigured  bool
	BTCPayConfigured  bool
	BillingConfigured bool
	HasPasswordHint   bool
	Has2FA            bool
	ErrorMessage      string
	SuccessMessage    string
	IsAdmin           bool
}

type UpgradeTemplate struct {
	Base BaseTemplate

	IsYearly         bool
	IsBTCPay         bool
	BillingEndpoints endpoints.BillingEndpoints
	SendUpgrades     []*shared.Upgrade
	VaultUpgrades    []*shared.Upgrade

	ShowVaultUpgradeNote bool
}

type VerificationTemplate struct {
	Base  BaseTemplate
	Email string
	Code  string
}

type ForgotPasswordTemplate struct {
	Base  BaseTemplate
	Email string
	Code  string
}

type ChangeEmailTemplate struct {
	Base         BaseTemplate
	CurrentEmail string
}

// ServeTemplate uses the name of a template and a generic struct and attempts to
// generate the template content using the provided struct. If there's an issue
// using the provided struct to fill out the template, an error will be thrown
// back to the caller.
func ServeTemplate[T any](w http.ResponseWriter, name string, fields T) error {
	err := templates.ExecuteTemplate(w, name, fields)
	if err != nil {
		return err
	}

	return nil
}

// init loads all HTML template files in the HTML embedded collection of files
func init() {
	// Load templates
	var templateList []string
	err := fs.WalkDir(HTML, ".", func(path string, d fs.DirEntry, err error) error {
		if strings.HasSuffix(path, ".html") {
			templateList = append(templateList, path)
		}
		return err
	})

	templates = template.Must(template.ParseFS(HTML, templateList...))
	if err != nil {
		panic(err)
	}
}
