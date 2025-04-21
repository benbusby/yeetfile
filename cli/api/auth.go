package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"yeetfile/cli/requests"
	"yeetfile/cli/utils"
	"yeetfile/lang"
	"yeetfile/shared"
	"yeetfile/shared/endpoints"
)

var ServerPasswordError = errors.New(lang.I18n.T("cli.api.error.password_required"))
var TwoFactorError = errors.New(lang.I18n.T("cli.api.error.2fa_error"))

// GetAccountInfo fetches the current user's account info
func (ctx *Context) GetAccountInfo() (shared.AccountResponse, error) {
	url := endpoints.Account.Format(ctx.Server)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.AccountResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.AccountResponse{}, utils.ParseHTTPError(resp)
	}

	var accountResponse shared.AccountResponse
	err = json.NewDecoder(resp.Body).Decode(&accountResponse)
	if err != nil {
		return shared.AccountResponse{}, err
	}

	return accountResponse, nil
}

// GetAccountUsage fetches the current user's used/available storage and
// used/available send.
func (ctx *Context) GetAccountUsage() (shared.UsageResponse, error) {
	url := endpoints.AccountUsage.Format(ctx.Server)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.UsageResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.UsageResponse{}, utils.ParseHTTPError(resp)
	}

	var usageResponse shared.UsageResponse
	err = json.NewDecoder(resp.Body).Decode(&usageResponse)
	if err != nil {
		return shared.UsageResponse{}, err
	}

	return usageResponse, nil
}

// Login logs a user into a YeetFile server, returning the server response,
// the session cookie, and any errors.
func (ctx *Context) Login(login shared.Login) (shared.LoginResponse, string, error) {
	url := endpoints.Login.Format(ctx.Server)
	reqData, err := json.Marshal(login)
	if err != nil {
		return shared.LoginResponse{}, "", err
	}

	resp, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return shared.LoginResponse{}, "", err
	} else if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusForbidden {
			return shared.LoginResponse{}, "", TwoFactorError
		}
		return shared.LoginResponse{}, "", utils.ParseHTTPError(resp)
	}

	var loginResponse shared.LoginResponse
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	if err != nil {
		return shared.LoginResponse{}, "", err
	}

	var session string
	cookies := resp.Cookies()
	if len(cookies) > 0 {
		ctx.Session = cookies[0].Value
		session = cookies[0].Value
	}

	return loginResponse, session, nil
}

// VerifyAccount finalizes account for account-id-only accounts by verifying
// the N-digit verification code and submitting their keys
func (ctx *Context) VerifyAccount(account shared.VerifyAccount) error {
	reqData, err := json.Marshal(account)
	if err != nil {
		return err
	}

	url := endpoints.VerifyAccount.Format(ctx.Server)
	response, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusUnauthorized {
			return errors.New(lang.I18n.T("cli.api.error.incorrect_val_code"))
		}

		return utils.ParseHTTPError(response)
	}

	return nil
}

// SubmitSignup initiates the signup process for an account-ID-only signup,
// returning their new account ID and allowing the user to proceed with verifying
// their new account.
func (ctx *Context) SubmitSignup(signup shared.Signup) (shared.SignupResponse, error) {
	reqData, err := json.Marshal(signup)
	if err != nil {
		return shared.SignupResponse{}, err
	}

	url := endpoints.Signup.Format(ctx.Server)
	response, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return shared.SignupResponse{}, err
	} else if response.StatusCode != http.StatusOK {
		if response.StatusCode == http.StatusForbidden {
			return shared.SignupResponse{}, ServerPasswordError
		}
		return shared.SignupResponse{}, utils.ParseHTTPError(response)
	}

	decoder := json.NewDecoder(response.Body)
	var signupResponse shared.SignupResponse
	err = decoder.Decode(&signupResponse)
	if err != nil {
		return shared.SignupResponse{}, err
	} else if len(signupResponse.Error) > 0 {
		return shared.SignupResponse{}, errors.New(signupResponse.Error)
	}

	return signupResponse, nil
}

// VerifyEmail verifies a new user's email using their email and the code sent
// to their email address
func (ctx *Context) VerifyEmail(email, code string) error {
	url := endpoints.VerifyEmail.Format(ctx.Server)
	reqData, err := json.Marshal(shared.VerifyEmail{
		Email: email,
		Code:  code,
	})

	response, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	} else if response.StatusCode >= http.StatusBadRequest {
		if response.StatusCode == http.StatusUnauthorized {
			return errors.New(lang.I18n.T("cli.api.error.incorrect_val_code"))
		}
		return errors.New(lang.I18n.T("cli.api.error.server_error"))
	}

	return nil
}

// GetSession returns the current session info.
func (ctx *Context) GetSession() (shared.SessionInfo, error) {
	url := endpoints.Session.Format(ctx.Server)
	resp, err := requests.GetRequest(ctx.Session, url)

	if err != nil {
		return shared.SessionInfo{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.SessionInfo{}, utils.ParseHTTPError(resp)
	}

	//var sessionInfo shared.SessionInfo
	//err = json.NewDecoder(resp.Body).Decode(&sessionInfo)
	//if err != nil {
	//	return shared.SessionInfo{}, err
	//}

	return shared.SessionInfo{}, nil
}

// LogOut invalidates the current session for the logged-in user
func (ctx *Context) LogOut() error {
	url := endpoints.Logout.Format(ctx.Server)
	response, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return err
	} else if response.StatusCode >= http.StatusBadRequest {
		return utils.ParseHTTPError(response)
	}

	return nil
}

// GetUserProtectedKey retrieves the user's private key, which has been
// encrypted with their unique user key before upload.
func (ctx *Context) GetUserProtectedKey() ([]byte, error) {
	url := endpoints.ProtectedKey.Format(ctx.Server)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, utils.ParseHTTPError(resp)
	}

	var protectedKey shared.ProtectedKeyResponse
	err = json.NewDecoder(resp.Body).Decode(&protectedKey)
	if err != nil {
		return nil, err
	}

	return protectedKey.ProtectedKey, err
}

// StartChangeEmail initiates the process for changing a user's email. If the
// user doesn't have an email set, the response will contain the change ID
// needed to confirm setting a new email. If they do have an email set, this
// ID will be sent to their current email.
func (ctx *Context) StartChangeEmail() (shared.StartEmailChangeResponse, error) {
	url := endpoints.ChangeEmail.Format(ctx.Server, "")
	resp, err := requests.PostRequest(ctx.Session, url, nil)
	if err != nil {
		return shared.StartEmailChangeResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.StartEmailChangeResponse{}, utils.ParseHTTPError(resp)
	}

	var changeResponse shared.StartEmailChangeResponse
	err = json.NewDecoder(resp.Body).Decode(&changeResponse)
	if err != nil {
		return shared.StartEmailChangeResponse{}, err
	}

	return changeResponse, nil
}

// ChangeEmail finalizes the change email process, sending a verification code
// to the user's new email and temporarily storing their updated user details
// in the db until the verification code is confirmed.
func (ctx *Context) ChangeEmail(changeEmail shared.ChangeEmail, changeID string) error {
	reqData, err := json.Marshal(changeEmail)
	if err != nil {
		return err
	}

	url := endpoints.ChangeEmail.Format(ctx.Server, changeID)
	resp, err := requests.PutRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(resp)
	}

	return nil
}

// ChangePassword changes a user's password, updating their login key hash
// and their encrypted private key.
func (ctx *Context) ChangePassword(password shared.ChangePassword) error {
	url := endpoints.ChangePassword.Format(ctx.Server)
	reqData, err := json.Marshal(password)
	if err != nil {
		return err
	}

	resp, err := requests.PutRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(resp)
	}

	return nil
}

// ChangePasswordHint accepts a plaintext password hint that will be encrypted
// by the server and sent to the user's email if they forget their password
func (ctx *Context) ChangePasswordHint(hint string) error {
	change := shared.ChangePasswordHint{Hint: hint}
	reqData, err := json.Marshal(change)
	if err != nil {
		return err
	}

	url := endpoints.ChangeHint.Format(ctx.Server)
	resp, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(resp)
	}

	return nil
}

// DeleteAccount removes the current user's YeetFile account.
func (ctx *Context) DeleteAccount(id string) error {
	url := endpoints.Account.Format(ctx.Server)
	reqData, err := json.Marshal(shared.DeleteAccount{Identifier: id})
	if err != nil {
		return err
	}

	response, err := requests.DeleteRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	} else if response.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(response)
	}

	return nil
}

// ForgotPassword sends a request for the user's password to be sent to the
// provided email (must have an account and have a hint set first).
func (ctx *Context) ForgotPassword(email string) error {
	url := endpoints.Forgot.Format(ctx.Server)
	reqData, err := json.Marshal(shared.ForgotPassword{Email: email})
	if err != nil {
		return err
	}

	response, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return err
	} else if response.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(response)
	}

	return nil
}

// Generate2FA requests a TOTP secret from the server. This only succeeds if the
// user doesn't already have 2FA enabled.
func (ctx *Context) Generate2FA() (shared.NewTOTP, error) {
	url := endpoints.TwoFactor.Format(ctx.Server)
	resp, err := requests.GetRequest(ctx.Session, url)
	if err != nil {
		return shared.NewTOTP{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.NewTOTP{}, utils.ParseHTTPError(resp)
	}

	var newTOTP shared.NewTOTP
	err = json.NewDecoder(resp.Body).Decode(&newTOTP)
	if err != nil {
		return shared.NewTOTP{}, err
	}

	return newTOTP, nil
}

// Disable2FA disables two-factor authentication for a user's account
func (ctx *Context) Disable2FA(code string) error {
	endpoint := endpoints.TwoFactor.Format(ctx.Server)
	url := fmt.Sprintf("%s?code=%s", endpoint, code)
	resp, err := requests.DeleteRequest(ctx.Session, url, nil)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(resp)
	}

	return nil
}

// Finalize2FA submits the secret returned by Generate2FA as well as a 6-digit
// code generated in the user's 2FA app to finalize setting up 2FA for their
// account. In response, they receive a number of one-time recovery codes that
// they can use in the event that they lose their authentication app.
func (ctx *Context) Finalize2FA(totp shared.SetTOTP) (shared.SetTOTPResponse, error) {
	url := endpoints.TwoFactor.Format(ctx.Server)
	reqData, err := json.Marshal(totp)
	if err != nil {
		return shared.SetTOTPResponse{}, err
	}

	resp, err := requests.PostRequest(ctx.Session, url, reqData)
	if err != nil {
		return shared.SetTOTPResponse{}, err
	} else if resp.StatusCode != http.StatusOK {
		return shared.SetTOTPResponse{}, utils.ParseHTTPError(resp)
	}

	var setTOTP shared.SetTOTPResponse
	err = json.NewDecoder(resp.Body).Decode(&setTOTP)
	if err != nil {
		return shared.SetTOTPResponse{}, err
	}

	return setTOTP, nil
}

// RecyclePaymentID frees the user's current payment ID and grants them a new one.
func (ctx *Context) RecyclePaymentID() error {
	url := endpoints.RecyclePaymentID.Format(ctx.Server)
	resp, err := requests.PutRequest(ctx.Session, url, nil)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return utils.ParseHTTPError(resp)
	}

	return nil
}
