package shared

import (
	"time"
	"yeetfile/shared/constants"
)

type AccountResponse struct {
	Email            string    `json:"email"`
	PaymentID        string    `json:"paymentID"`
	HasPasswordHint  bool      `json:"hasPasswordHint"`
	Has2FA           bool      `json:"has2FA"`
	StorageAvailable int64     `json:"storageAvailable"`
	StorageUsed      int64     `json:"storageUsed"`
	SendAvailable    int64     `json:"sendAvailable"`
	SendUsed         int64     `json:"sendUsed"`
	UpgradeExp       time.Time `json:"upgradeExp" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
}

type UsageResponse struct {
	StorageAvailable int64 `json:"storageAvailable"`
	StorageUsed      int64 `json:"storageUsed"`
	SendAvailable    int64 `json:"sendAvailable"`
	SendUsed         int64 `json:"sendUsed"`
}

type UploadMetadata struct {
	Name       string `json:"name"`
	Chunks     int    `json:"chunks"`
	Size       int64  `json:"size"`
	Downloads  int    `json:"downloads"`
	Expiration string `json:"expiration"`
}

type VaultUpload struct {
	Name         string `json:"name"`
	Length       int64  `json:"length"`
	Chunks       int    `json:"chunks"`
	FolderID     string `json:"folderID"`
	ProtectedKey []byte `json:"protectedKey"`
	PasswordData []byte `json:"passwordData"`
}

type ModifyVaultItem struct {
	Name         string `json:"name"`
	PasswordData []byte `json:"passwordData" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type MetadataUploadResponse struct {
	ID string `json:"id"`
}

type NewFolderResponse struct {
	ID string `json:"id"`
}

type VaultItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	Modified     time.Time `json:"modified" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
	ProtectedKey []byte    `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	SharedWith   int       `json:"sharedWith"`
	SharedBy     string    `json:"sharedBy"`
	LinkTag      string    `json:"linkTag"`
	CanModify    bool      `json:"canModify"`
	IsOwner      bool      `json:"isOwner"`
	RefID        string    `json:"refID"`
	PasswordData []byte    `json:"passwordData" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type VaultItemInfo struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	Modified     time.Time `json:"modified" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
	ProtectedKey []byte    `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	CanModify    bool      `json:"canModify"`
	IsOwner      bool      `json:"isOwner"`
	RefID        string    `json:"refID"`
	KeySequence  [][]byte  `json:"keySequence" ts_type:"Uint8Array[]" ts_transform:"__VALUE__.map(base64ToArray)"`
}

type NewVaultFolder struct {
	Name         string `json:"name"`
	ProtectedKey []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ParentID     string `json:"parentID"`
}

type NewPublicVaultFolder struct {
	ID           string `json:"id"`
	ProtectedKey []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	LinkTag      string `json:"linkTag"`
}

type VaultFolder struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	Modified       time.Time `json:"modified" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
	ParentID       string    `json:"parentID"`
	ProtectedKey   []byte    `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	SharedWith     int       `json:"sharedWith"`
	SharedBy       string    `json:"sharedBy"`
	LinkTag        string    `json:"linkTag"`
	CanModify      bool      `json:"canModify"`
	RefID          string    `json:"refID"`
	IsOwner        bool      `json:"isOwner"`
	PasswordFolder bool      `json:"passwordFolder"`
}

type VaultFolderResponse struct {
	Items         []VaultItem   `json:"items"`
	Folders       []VaultFolder `json:"folders"`
	CurrentFolder VaultFolder   `json:"folder"`
	KeySequence   [][]byte      `json:"keySequence" ts_type:"Uint8Array[]" ts_transform:"__VALUE__.map(base64ToArray)"`
}

type VaultDownloadResponse struct {
	Name         string `json:"name"`
	ID           string `json:"id"`
	Size         int64  `json:"size"`
	Chunks       int    `json:"chunks"`
	ProtectedKey []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	PasswordData []byte `json:"passwordData" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type PlaintextUpload struct {
	Name       string `json:"name"`
	Salt       []byte `json:"salt" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	Downloads  int    `json:"downloads"`
	Expiration string `json:"expiration"`
	Text       []byte `json:"text" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type DownloadResponse struct {
	Name       string    `json:"name"`
	ID         string    `json:"id"`
	Size       int64     `json:"size"`
	Chunks     int       `json:"chunks"`
	Downloads  int       `json:"downloads"`
	Expiration time.Time `json:"expiration" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`
}

type Signup struct {
	Identifier              string `json:"identifier"`
	LoginKeyHash            []byte `json:"loginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	PublicKey               []byte `json:"publicKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedPrivateKey     []byte `json:"protectedPrivateKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedVaultFolderKey []byte `json:"protectedVaultFolderKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	PasswordHint            string `json:"passwordHint"`
	ServerPassword          string `json:"serverPassword"`
}

type SignupResponse struct {
	Identifier string `json:"identifier"`
	Captcha    string `json:"captcha"`
	Error      string `json:"error"`
}

type VerifyAccount struct {
	ID                      string `json:"id"`
	Code                    string `json:"code"`
	LoginKeyHash            []byte `json:"loginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	PublicKey               []byte `json:"publicKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedPrivateKey     []byte `json:"protectedPrivateKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedVaultFolderKey []byte `json:"protectedVaultFolderKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type Login struct {
	Identifier   string `json:"identifier"`
	LoginKeyHash []byte `json:"loginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	Code         string `json:"code"`
}

type LoginResponse struct {
	PublicKey    []byte `json:"publicKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedKey []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type SessionInfo struct {
	Meter int `json:"meter"`
}

type ForgotPassword struct {
	Email string `json:"email"`
}

type VerifyEmail struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type ResetPassword struct {
	Email           string `json:"email"`
	Code            string `json:"code"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirmPassword"`
}

type ChangePasswordHint struct {
	Hint string `json:"hint"`
}

type PubKeyResponse struct {
	PublicKey []byte `json:"publicKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type ProtectedKeyResponse struct {
	ProtectedKey []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type ShareItemRequest struct {
	User         string `json:"user"`
	CanModify    bool   `json:"canModify"`
	ProtectedKey []byte `json:"protectedKey"`
}

type NewSharedItem struct {
	ItemID       string
	UserID       string
	SharerName   string
	RecipientID  string
	ProtectedKey []byte
	CanModify    bool
}

type FileOwnershipInfo struct {
	CanModify bool `json:"canModify"`
}

type FolderOwnershipInfo struct {
	ID        string `json:"id"`
	RefID     string `json:"refID"`
	CanModify bool   `json:"canModify"`
	IsOwner   bool   `json:"isOwner"`
}

type ShareInfo struct {
	ID        string `json:"id"`
	Recipient string `json:"recipientName"`
	CanModify bool   `json:"canModify"`
}

type ShareEdit struct {
	ID        string `json:"id"`
	ItemID    string `json:"itemID"`
	CanModify bool   `json:"canModify"`
}

type DeleteResponse struct {
	FreedSpace int64 `json:"freedSpace"`
}

type DeleteAccount struct {
	Identifier string `json:"identifier"`
}

type StartEmailChangeResponse struct {
	ChangeID string `json:"changeID"`
}

type ChangeEmail struct {
	NewEmail        string `json:"newEmail"`
	OldLoginKeyHash []byte `json:"oldLoginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	NewLoginKeyHash []byte `json:"newLoginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedKey    []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type ChangePassword struct {
	OldLoginKeyHash []byte `json:"oldLoginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	NewLoginKeyHash []byte `json:"newLoginKeyHash" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
	ProtectedKey    []byte `json:"protectedKey" ts_type:"Uint8Array" ts_transform:"__VALUE__ ? base64ToArray(__VALUE__) : new Uint8Array()"`
}

type NewTOTP struct {
	B64Image string `json:"b64Image"`
	Secret   string `json:"secret"`
	URI      string `json:"uri"`
}

type SetTOTP struct {
	Secret string `json:"secret"`
	Code   string `json:"code"`
}

type SetTOTPResponse struct {
	RecoveryCodes [6]string `json:"recoveryCodes"`
}

type ServerInfo struct {
	StorageBackend     string `json:"storageBackend"`
	PasswordRestricted bool   `json:"passwordRestricted"`
	MaxUserCountSet    bool   `json:"maxUserCountSet"`
	EmailConfigured    bool   `json:"emailConfigured"`
	BillingEnabled     bool   `json:"billingEnabled"`
	StripeEnabled      bool   `json:"stripeEnabled"`
	BTCPayEnabled      bool   `json:"btcPayEnabled"`
	DefaultStorage     int64  `json:"defaultStorage"`
	DefaultSend        int64  `json:"defaultSend"`

	Upgrades      Upgrades   `json:"upgrades"`
	MonthUpgrades []*Upgrade `json:"monthUpgrades"`
	YearUpgrades  []*Upgrade `json:"yearUpgrades"`
}

type PassEntry struct {
	Username        string   `json:"username"`
	Password        string   `json:"password"`
	PasswordHistory []string `json:"passwordHistory"`
	URLs            []string `json:"urls"`
	Notes           string   `json:"notes"`
}

type ItemIndex struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Folder string   `json:"folder"`
	URIs   []string `json:"uris"`
}

type Upgrade struct {
	Tag         string `json:"tag"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Price       int64  `json:"price"`
	Bytes       int64  `json:"bytes"`
	Annual      bool   `json:"annual,omitempty"`
	BTCPayLink  string `json:"btcpay_link"`

	ReadableBytes  string
	IsVaultUpgrade bool
	Quantity       int
}

type VaultUpgrade struct {
	Tag         string                    `json:"tag"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Price       int64                     `json:"price"`
	Bytes       int64                     `json:"bytes"`
	Duration    constants.UpgradeDuration `json:"duration"`

	SendGB     int `json:"send_gb"`
	SendGBReal int64

	StorageGB     int `json:"storage_gb"`
	StorageGBReal int64

	BTCPayLink string `json:"btcpay_link"`
}

type Upgrades struct {
	SendUpgrades  []*Upgrade `json:"send_upgrades"`
	VaultUpgrades []*Upgrade `json:"vault_upgrades"`
}

type AdminUserInfoResponse struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	StorageUsed string `json:"storageUsed"`
	SendUsed    string `json:"sendUsed"`

	Files []AdminFileInfoResponse `json:"files"`
}

type AdminFileInfoResponse struct {
	ID         string    `json:"id"`
	BucketName string    `json:"bucketName"`
	Size       string    `json:"size"`
	OwnerID    string    `json:"ownerID"`
	Modified   time.Time `json:"modified" ts_type:"Date" ts_transform:"new Date(__VALUE__)"`

	RawSize int64
}
