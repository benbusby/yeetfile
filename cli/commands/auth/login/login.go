package login

import (
	"strings"
	"yeetfile/cli/crypto"
	"yeetfile/cli/globals"
	"yeetfile/cli/utils"
	"yeetfile/shared"
)

// LogIn logs into YeetFile by using the provided identifier and password to
// generate the login key hash, and stores the user's key pair in their config
// directory
func LogIn(identifier, password, code string, sessionKey, vaultKey []byte) error {
	identifier = strings.TrimSpace(identifier)
	password = strings.TrimSpace(password)

	userKey, loginKeyHash := crypto.GenerateUserKeys(identifier, password)

	login := shared.Login{
		Identifier:   identifier,
		LoginKeyHash: loginKeyHash,
		Code:         code,
	}

	loginResponse, session, err := globals.API.Login(login)
	if err != nil {
		return err
	}

	privateKey, err := crypto.DecryptChunk(userKey, loginResponse.ProtectedKey)
	utils.HandleCLIError("failed to decrypt private key", err)

	encPrivateKey, _ := crypto.EncryptChunk(vaultKey, privateKey)
	err = globals.Config.SetKeys(encPrivateKey, loginResponse.PublicKey)
	if err != nil {
		return err
	}

	encSession, err := crypto.EncryptChunk(sessionKey, []byte(session))
	err = globals.Config.SetSession(string(encSession))
	if err != nil {
		return err
	}

	return nil
}

// RequestPasswordHint sends a request for the password hint set for the account
// matching the provided email.
func RequestPasswordHint(email string) error {
	err := globals.API.ForgotPassword(email)
	return err
}
