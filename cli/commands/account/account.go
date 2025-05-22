package account

import (
	"errors"
	"fmt"
	"time"
	"yeetfile/cli/crypto"
	"yeetfile/cli/globals"
	"yeetfile/cli/lang"
	"yeetfile/cli/utils"
	"yeetfile/shared"
)

type ChangePasswordForm struct {
	Identifier  string
	OldPassword string
	NewPassword string
}

func getUpgradeString(exp time.Time) string {
	if exp.Year() < 2024 {
		return lang.I18n.T("cli.command.account.inactive")
	} else if exp.Before(time.Now()) {
		return lang.I18n.T("cli.command.account.expired") + exp.Format(time.DateOnly)
	} else {
		return lang.I18n.T("cli.command.account.expired") + utils.LocalTimeFromUTC(exp).
			Format(time.DateOnly) + ")"
	}
}

func getStorageString(used, available int64, isSend bool) string {
	if available == 0 && used == 0 {
		return lang.I18n.T("cli.command.account.storage_none")
	} else if available <= 0 && used >= 0 {
		return fmt.Sprintf("%s "+lang.I18n.T("cli.command.account.storage_used"), shared.ReadableFileSize(used))
	} else {
		return fmt.Sprintf("%s / %s (%s "+lang.I18n.T("cli.command.account.storage_remain")+")",
			shared.ReadableFileSize(used),
			shared.ReadableFileSize(available),
			shared.ReadableFileSize(available-used))
	}
}

func changePassword(identifier, password, newPassword string) error {
	userKey := crypto.GenerateUserKey([]byte(identifier), []byte(password))
	oldLoginKeyHash := crypto.GenerateLoginKeyHash(userKey, []byte(password))

	newUserKey := crypto.GenerateUserKey([]byte(identifier), []byte(newPassword))
	newLoginKeyHash := crypto.GenerateLoginKeyHash(newUserKey, []byte(newPassword))

	protectedKey, err := globals.API.GetUserProtectedKey()
	if err != nil {
		return errors.New(lang.I18n.T("cli.command.account.error.fetching_protected_key"))
	}

	privateKey, err := crypto.DecryptChunk(userKey, protectedKey)
	if err != nil {
		return errors.New(lang.I18n.T("cli.command.account.error.decrypting_protected_key"))
	}

	newProtectedKey, err := crypto.EncryptChunk(newUserKey, privateKey)
	if err != nil {
		return errors.New(lang.I18n.T("cli.command.account.error.encrypting_protected_key"))
	}

	return globals.API.ChangePassword(shared.ChangePassword{
		OldLoginKeyHash: oldLoginKeyHash,
		NewLoginKeyHash: newLoginKeyHash,
		ProtectedKey:    newProtectedKey,
	})
}

func changePasswordHint(passwordHint string) error {
	return globals.API.ChangePasswordHint(passwordHint)
}

func changeEmail(identifier, password, newEmail, changeID string) error {
	userKey := crypto.GenerateUserKey([]byte(identifier), []byte(password))
	oldLoginKeyHash := crypto.GenerateLoginKeyHash(userKey, []byte(password))

	newUserKey := crypto.GenerateUserKey([]byte(newEmail), []byte(password))
	newLoginKeyHash := crypto.GenerateLoginKeyHash(newUserKey, []byte(password))

	protectedKey, err := globals.API.GetUserProtectedKey()
	if err != nil {
		return errors.New(lang.I18n.T("cli.command.account.error.fetching_protected_key"))
	}

	privateKey, err := crypto.DecryptChunk(userKey, protectedKey)
	if err != nil {
		return errors.New(lang.I18n.T("cli.command.account.error.decrypting_protected_key"))
	}

	newProtectedKey, err := crypto.EncryptChunk(newUserKey, privateKey)
	if err != nil {
		return errors.New(lang.I18n.T("cli.command.account.error.encrypting_protected_key"))
	}

	return globals.API.ChangeEmail(shared.ChangeEmail{
		NewEmail:        newEmail,
		OldLoginKeyHash: oldLoginKeyHash,
		NewLoginKeyHash: newLoginKeyHash,
		ProtectedKey:    newProtectedKey,
	}, changeID)
}

func FetchAccountDetails() (shared.AccountResponse, string) {
	account, err := globals.API.GetAccountInfo()
	if err != nil {
		msg := fmt.Sprintf(lang.I18n.T("cli.command.account.error.fecthing_account_details")+": %v\n", err)
		return account, msg
	}

	upgradeStr := getUpgradeString(account.UpgradeExp)
	storageStr := getStorageString(account.StorageUsed, account.StorageAvailable, false)
	sendStr := getStorageString(account.SendUsed, account.SendAvailable, true)

	emailStr := account.Email
	if len(account.Email) == 0 {
		emailStr = lang.I18n.T("cli.command.account.email_none")
	}

	passwordHintStr := lang.I18n.T("cli.command.account.passhint_notset")
	if account.HasPasswordHint {
		passwordHintStr = lang.I18n.T("cli.command.account.passhint_enabled")
	}

	twoFactorStr := lang.I18n.T("cli.command.account.tfa_notset")
	if account.Has2FA {
		twoFactorStr = lang.I18n.T("cli.command.account.tfa_enabled")
	}

	accountDetails := fmt.Sprintf(""+
		lang.I18n.T("cli.command.account.details.email")+": %s\n"+
		lang.I18n.T("cli.command.account.details.vault")+": %s\n"+
		lang.I18n.T("cli.command.account.details.send")+":  %s\n\n"+
		lang.I18n.T("cli.command.account.details.upgrades")+":      %s\n"+
		lang.I18n.T("cli.command.account.details.passhint")+": %s\n"+
		lang.I18n.T("cli.command.account.details.tfa")+":    %s\n"+
		lang.I18n.T("cli.command.account.details.paymentid")+":    %s",
		shared.EscapeString(emailStr),
		storageStr,
		sendStr,
		upgradeStr,
		passwordHintStr,
		twoFactorStr,
		shared.EscapeString(account.PaymentID))

	return account, accountDetails
}

func generateUpgradeDesc(upgrade shared.Upgrade) string {
	descStr := fmt.Sprintf(
		`%s

** $%d **`,
		upgrade.Description,
		upgrade.Price)
	return shared.EscapeString(descStr)
}
