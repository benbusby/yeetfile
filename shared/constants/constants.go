package constants

type UpgradeDuration string

const (
	VERSION = "0.2.0"

	// JSSessionKey determines the hardcoded string that gets replaced in db.ts in
	// order to have a semi-random value used in encrypting the user's key pair in
	// IndexedDB.
	// --- Changes to this value must be reflected in db.ts
	JSSessionKey = "JS_SESSION_KEY"

	CLIUserAgent                    = "yeetfile-cli"
	AuthSessionStore                = "auth"
	Argon2Mem                uint32 = 64 // MB
	Argon2Iter               uint32 = 2
	TotalBandwidthMultiplier        = 3 // 3x available storage
	BandwidthMonitorDuration        = 7 // 7 day period
	IVSize                          = 12
	KeySize                         = 32
	ChunkSize                       = 10000000 // 10 mb
	TotalOverhead                   = 28       // encryption overhead (16) + iv size (12)
	MaxPlaintextLen                 = 2000
	MaxHintLen                      = 200
	PlaintextIDPrefix               = "text"
	FileIDPrefix                    = "file"
	VerificationCodeLength          = 6
	ChangeIDLength                  = 9
	MaxTransferThreads              = 3
	MaxSendAgeDays                  = 30 //days
	MaxPassNoteLen                  = 500
	RecoveryCodeLen                 = 8
)
