package endpoints

import (
	"fmt"
	"strings"
)

type Endpoint string

type HTMLEndpoints struct {
	Account        string
	Send           string
	Vault          string
	Pass           string
	Login          string
	Signup         string
	Forgot         string
	ChangeHint     string
	ChangePassword string
	VerifyEmail    string
	TwoFactor      string
	VaultFile      string
	Info           string
	Upgrade        string
	Admin          string
}

type BillingEndpoints struct {
	BTCPayCheckout string
	StripeCheckout string
}

var HTMLPageEndpoints HTMLEndpoints
var BillingPageEndpoints BillingEndpoints

var (
	Signup           = Endpoint("/api/signup")
	Login            = Endpoint("/api/login")
	Logout           = Endpoint("/api/logout")
	Account          = Endpoint("/api/account")
	AccountUsage     = Endpoint("/api/account/usage")
	RecyclePaymentID = Endpoint("/api/account/recycle/payment_id")
	Forgot           = Endpoint("/api/forgot")
	Session          = Endpoint("/api/session")
	TwoFactor        = Endpoint("/api/2fa")
	VerifyAccount    = Endpoint("/api/verify/account")
	VerifyEmail      = Endpoint("/api/verify/email")
	ChangeEmail      = Endpoint("/api/change/email/*")
	ChangePassword   = Endpoint("/api/change/password")
	ChangeHint       = Endpoint("/api/change/hint")
	ServerInfo       = Endpoint("/api/info")

	AdminUserActions = Endpoint("/api/admin/user/*")
	AdminFileActions = Endpoint("/api/admin/files/*")

	Up = Endpoint("/up")

	PassRoot     = Endpoint("/api/pass")
	PassFolder   = Endpoint("/api/pass/folder/*")
	PassEntry    = Endpoint("/api/pass/entry/*")
	NewPassEntry = Endpoint("/api/pass/u")

	VaultRoot   = Endpoint("/api/vault")
	VaultFolder = Endpoint("/api/vault/folder/*")
	VaultFile   = Endpoint("/api/vault/file/*")

	UploadVaultFileMetadata   = Endpoint("/api/vault/u")
	UploadVaultFileData       = Endpoint("/api/vault/u/*/*")
	DownloadVaultFileMetadata = Endpoint("/api/vault/d/*")
	DownloadVaultFileData     = Endpoint("/api/vault/d/*/*")

	UploadSendFileMetadata   = Endpoint("/api/send/u")
	UploadSendFileData       = Endpoint("/api/send/u/*/*")
	UploadSendText           = Endpoint("/api/send/plaintext")
	DownloadSendFileMetadata = Endpoint("/api/send/d/*")
	DownloadSendFileData     = Endpoint("/api/send/d/*/*")

	ShareFile    = Endpoint("/api/share/file/*")
	ShareFolder  = Endpoint("/api/share/folder/*")
	PubKey       = Endpoint("/api/pubkey")
	ProtectedKey = Endpoint("/api/protectedkey")

	StripeWebhook  = Endpoint("/stripe/webhook")
	StripeCheckout = Endpoint("/stripe/checkout")
	BTCPayWebhook  = Endpoint("/btcpay/webhook")
	BTCPayCheckout = Endpoint("/btcpay/checkout")

	StaticFile = Endpoint("/static/*/*")

	HTMLAccount          = Endpoint("/account")
	HTMLHome             = Endpoint("/")
	HTMLSend             = Endpoint("/send")
	HTMLSendDownload     = Endpoint("/send/*")
	HTMLPass             = Endpoint("/pass")
	HTMLPassFolder       = Endpoint("/pass/*")
	HTMLPassEntry        = Endpoint("/pass/*/entry/*")
	HTMLPassIndex        = Endpoint("/pass/index")
	HTMLVault            = Endpoint("/vault")
	HTMLVaultFolder      = Endpoint("/vault/*")
	HTMLVaultFile        = Endpoint("/vault/*/file/*")
	HTMLLogin            = Endpoint("/login")
	HTMLSignup           = Endpoint("/signup")
	HTMLForgot           = Endpoint("/forgot")
	HTMLChangeEmail      = Endpoint("/change/email/*")
	HTMLChangePassword   = Endpoint("/change/password")
	HTMLChangeHint       = Endpoint("/change/hint")
	HTMLVerifyEmail      = Endpoint("/verify/email")
	HTMLTwoFactor        = Endpoint("/2fa")
	HTMLServerInfo       = Endpoint("/info")
	HTMLCheckoutComplete = Endpoint("/checkout/complete")
	HTMLUpgrade          = Endpoint("/upgrade")
	HTMLAdmin            = Endpoint("/admin")
)

var JSVarNameMap = map[Endpoint]string{
	Signup:           "Signup",
	Login:            "Login",
	Logout:           "Logout",
	Forgot:           "Forgot",
	Session:          "Session",
	Account:          "Account",
	AccountUsage:     "AccountUsage",
	RecyclePaymentID: "RecyclePaymentID",
	TwoFactor:        "TwoFactor",
	VerifyAccount:    "VerifyAccount",
	VerifyEmail:      "VerifyEmail",
	ChangeEmail:      "ChangeEmail",
	ChangePassword:   "ChangePassword",
	ChangeHint:       "ChangeHint",
	ServerInfo:       "ServerInfo",

	AdminUserActions: "AdminUserActions",
	AdminFileActions: "AdminFileActions",

	PassRoot:     "PassRoot",
	PassFolder:   "PassFolder",
	PassEntry:    "PassEntry",
	NewPassEntry: "NewPassEntry",

	VaultRoot:   "VaultRoot",
	VaultFolder: "VaultFolder",
	VaultFile:   "VaultFile",

	UploadVaultFileMetadata:   "UploadVaultFileMetadata",
	UploadVaultFileData:       "UploadVaultFileData",
	DownloadVaultFileMetadata: "DownloadVaultFileMetadata",
	DownloadVaultFileData:     "DownloadVaultFileData",

	UploadSendFileMetadata:   "UploadSendFileMetadata",
	UploadSendFileData:       "UploadSendFileData",
	UploadSendText:           "UploadSendText",
	DownloadSendFileMetadata: "DownloadSendFileMetadata",
	DownloadSendFileData:     "DownloadSendFileData",

	ShareFile:    "ShareFile",
	ShareFolder:  "ShareFolder",
	PubKey:       "PubKey",
	ProtectedKey: "ProtectedKey",

	StaticFile: "StaticFile",

	StripeCheckout: "StripeCheckout",

	HTMLHome:             "HTMLHome",
	HTMLAccount:          "HTMLAccount",
	HTMLSend:             "HTMLSend",
	HTMLSendDownload:     "HTMLSendDownload",
	HTMLPass:             "HTMLPass",
	HTMLPassFolder:       "HTMLPassFolder",
	HTMLPassIndex:        "HTMLPassIndex",
	HTMLVault:            "HTMLVault",
	HTMLVaultFolder:      "HTMLVaultFolder",
	HTMLVaultFile:        "HTMLVaultFile",
	HTMLLogin:            "HTMLLogin",
	HTMLSignup:           "HTMLSignup",
	HTMLChangeEmail:      "HTMLChangeEmail",
	HTMLChangePassword:   "HTMLChangePassword",
	HTMLChangeHint:       "HTMLChangeHint",
	HTMLVerifyEmail:      "HTMLVerifyEmail",
	HTMLTwoFactor:        "HTMLTwoFactor",
	HTMLServerInfo:       "HTMLServerInfo",
	HTMLCheckoutComplete: "HTMLCheckoutComplete",
	HTMLAdmin:            "HTMLAdmin",
}

func (e Endpoint) Format(server string, args ...string) string {
	strEndpoint := string(e)
	for _, arg := range args {
		strEndpoint = strings.Replace(strEndpoint, "*", arg, 1)
	}

	// Remove remaining wildcards
	strEndpoint = strings.ReplaceAll(strEndpoint, "*", "")

	server = strings.TrimSuffix(server, "/")
	strEndpoint = strings.TrimPrefix(strEndpoint, "/")
	url := fmt.Sprintf("%s/%s", server, strEndpoint)
	return url
}

func init() {
	HTMLPageEndpoints = HTMLEndpoints{
		Account:        string(HTMLAccount),
		Send:           string(HTMLSend),
		Pass:           string(HTMLPass),
		Vault:          string(HTMLVault),
		VaultFile:      string(HTMLVaultFile),
		Login:          string(HTMLLogin),
		Signup:         string(HTMLSignup),
		Forgot:         string(HTMLForgot),
		ChangePassword: string(HTMLChangePassword),
		ChangeHint:     string(HTMLChangeHint),
		VerifyEmail:    string(HTMLVerifyEmail),
		TwoFactor:      string(HTMLTwoFactor),
		Info:           string(HTMLServerInfo),
		Upgrade:        string(HTMLUpgrade),
		Admin:          string(HTMLAdmin),
	}

	BillingPageEndpoints = BillingEndpoints{
		BTCPayCheckout: string(BTCPayCheckout),
		StripeCheckout: string(StripeCheckout),
	}
}
